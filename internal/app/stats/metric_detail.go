package stats

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderActiveMetricDetail(width int) string {
	switch m.tab {
	case statsTabOverview:
		return m.renderOverviewMetricDetail(width)
	case statsTabActivity:
		return m.renderActivityMetricDetail(width)
	case statsTabSessions:
		return m.renderSessionsMetricDetail(width)
	case statsTabTools:
		return m.renderToolsMetricDetail(width)
	case statsTabCache:
		return m.renderCacheMetricDetail(width)
	case statsTabPerformance:
		return m.renderPerformanceMetricDetail(width)
	default:
		return ""
	}
}

func renderStatsMetricDetail(title string, width int, chips []chip, lines ...string) string {
	innerWidth := max(width-4, 1)
	parts := []string{renderStatsTitle(title)}
	if len(chips) > 0 {
		parts = append(parts, renderSummaryChips(chips, innerWidth))
	}
	parts = append(parts, wrapStatsMetricDetailLines(lines, innerWidth)...)
	return renderFramedBox("Metric detail", width, colorPrimary, strings.Join(parts, "\n"))
}

func metricDetailLine(label, value string) string {
	return styleMetaLabel.Render(label) + " " + styleMetaValue.Render(value)
}

func selectedStatsTitle(title string, selected bool) string {
	if selected {
		return "▸ " + title
	}
	return title
}

func wrapStatsMetricDetailLines(lines []string, width int) []string {
	if len(lines) == 0 || width <= 0 {
		return lines
	}

	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		for segment := range strings.SplitSeq(line, "\n") {
			for wrappedLine := range strings.SplitSeq(ansi.Wordwrap(segment, width, ""), "\n") {
				wrapped = append(wrapped, wrappedLine)
			}
		}
	}
	return wrapped
}

func (m statsModel) renderOverviewMetricDetail(width int) string {
	lane, _, ok := m.selectedStatsLane()
	if !ok {
		return renderStatsMetricDetail("Overview", width, nil, noDataLabel)
	}

	overview := m.snapshot.Overview
	if lane.id == statsLaneOverviewModel {
		leader, share := leadingModelDetail(overview)
		return renderStatsMetricDetail(lane.title, width, []chip{
			{Label: "leading model", Value: leader},
			{Label: "share", Value: share},
			{Label: "tokens", Value: statspkg.FormatNumber(overview.Tokens.Total)},
		},
			metricDetailLine("Question", "Which models are driving token use?"),
			metricDetailLine("Reading", "Longer bars mean more total tokens in the active slice."),
		)
	}
	if lane.id == statsLaneOverviewProject {
		leader, share := leadingProjectDetail(overview)
		return renderStatsMetricDetail(lane.title, width, []chip{
			{Label: "leading project", Value: leader},
			{Label: "share", Value: share},
			{Label: "tokens", Value: statspkg.FormatNumber(overview.Tokens.Total)},
		},
			metricDetailLine("Question", "Which projects are driving token use?"),
			metricDetailLine("Reading", "Longer bars mean more total tokens in the active slice."),
		)
	}
	if lane.id == statsLaneOverviewProviderVersion {
		items := m.snapshot.Overview.ByProviderVersion
		if len(items) == 0 {
			return renderStatsMetricDetail(lane.title, width, nil, noDataLabel)
		}
		total := 0
		for _, item := range items {
			total += item.Tokens
		}
		return renderStatsMetricDetail(lane.title, width, []chip{
			{Label: "providers", Value: statspkg.FormatNumber(providerVersionProviderCount(items))},
			{Label: "entries", Value: statspkg.FormatNumber(len(items))},
			{Label: "tokens", Value: statspkg.FormatNumber(total)},
		},
			metricDetailLine("Question", "Which provider/version pairs are driving tokens in the active range?"),
			metricDetailLine(
				"Reading",
				"Each row shows provider, version, a proportional bar, total tokens, and share within the filtered dataset.",
			),
			metricDetailLine("Scope", "Use filters to narrow provider, model, and version."),
		)
	}

	session, index, selected := m.selectedOverviewSession()
	if !selected {
		return renderStatsMetricDetail(
			lane.title,
			width,
			nil,
			"No token-heavy sessions are available.",
		)
	}
	return renderStatsMetricDetail(lane.title, width, []chip{
		{Label: "session", Value: formatFractionInt(index+1, len(overview.TopSessions))},
		{Label: "project", Value: session.Project},
		{Label: "tokens", Value: statspkg.FormatNumber(session.Tokens)},
	},
		metricDetailLine("Slug", session.Slug),
		metricDetailLine("Date", session.Timestamp.Format("2006-01-02")),
		metricDetailLine("Messages", statspkg.FormatNumber(session.MessageCount)),
		metricDetailLine("Duration", session.Duration.String()),
		"Enter opens the selected session.",
	)
}

func (m statsModel) renderActivityMetricDetail(width int) string {
	lane, _, ok := m.selectedStatsLane()
	if !ok {
		return renderStatsMetricDetail("Activity", width, nil, noDataLabel)
	}

	if lane.id == statsLaneActivityDaily {
		label := activityMetricName(m.activityMetric)
		_, counts := m.activitySeries()
		peakDay, peakCount := peakDailyCount(counts)
		return renderStatsMetricDetail("Daily Activity", width, []chip{
			{Label: "metric", Value: label},
			{Label: "peak day", Value: peakDay},
			{Label: "total", Value: statspkg.FormatNumber(totalDailyCount(counts))},
		},
			metricDetailLine("Question", "How does work volume change across the active range?"),
			metricDetailLine(
				"Reading",
				"The X-axis is calendar day and the Y-axis is the selected daily metric.",
			),
			"Press m to cycle sessions, messages, and tokens.",
			fmt.Sprintf("Peak day reached %s %s.", statspkg.FormatNumber(peakCount), strings.ToLower(label)),
		)
	}

	slot, count := busiestHeatmapSlot(m.snapshot.Activity.Heatmap)
	return renderStatsMetricDetail(lane.title, width, []chip{
		{Label: "busiest slot", Value: slot},
		{Label: "sessions", Value: statspkg.FormatNumber(count)},
	},
		metricDetailLine("Question", "When does work tend to happen?"),
		metricDetailLine(
			"Reading",
			"Rows are weekdays, columns are hours, and darker cells mean more sessions.",
		),
	)
}

func (m statsModel) renderSessionsMetricDetail(width int) string {
	lane, _, ok := m.selectedStatsLane()
	if !ok {
		return renderStatsMetricDetail("Sessions", width, nil, noDataLabel)
	}

	sessionStats := m.snapshot.Sessions
	if lane.id == statsLaneSessionsDuration {
		bucket, count := dominantHistogramBucket(sessionStats.DurationHistogram)
		return renderStatsMetricDetail(lane.title, width, []chip{
			{Label: "dominant bucket", Value: bucket},
			{Label: "sessions", Value: statspkg.FormatNumber(count)},
		},
			metricDetailLine("Question", "Are sessions mostly quick checks or long runs?"),
			metricDetailLine("Reading", "The X-axis is duration bucket and the Y-axis is session count."),
		)
	}
	if lane.id == statsLaneSessionsMessages {
		bucket, count := dominantHistogramBucket(sessionStats.MessageHistogram)
		return renderStatsMetricDetail(lane.title, width, []chip{
			{Label: "dominant bucket", Value: bucket},
			{Label: "sessions", Value: statspkg.FormatNumber(count)},
		},
			metricDetailLine("Question", "Do sessions stay short or turn into long exchanges?"),
			metricDetailLine("Reading", "The X-axis is message-count bucket and the Y-axis is session count."),
		)
	}
	if lane.id == statsLaneSessionsContext {
		if m.sessionsGrouped {
			return renderStatsMetricDetail(lane.title, width, groupedTurnMetricDetailChips(m),
				metricDetailLine("Question", "How does prompt growth differ across versions for the selected provider?"),
				metricDetailLine(
					"Reading",
					"Each stacked bar is one main-thread user turn position, with colors splitting the total by version.",
				),
				metricDetailLine("Scope", groupedTurnMetricScope(m)),
			)
		}
		mode := m.sessionsPromptMode
		metrics := m.sessionTurnMetricsForMode(mode)
		return renderStatsMetricDetail(lane.title, width, append([]chip{
			{Label: "stat", Value: mode.ShortLabel()},
		}, promptMetricDetailChips(metrics, mode)...),
			metricDetailLine("Question", "How does prompt size grow as main-thread sessions go deeper?"),
			metricDetailLine(
				"Reading",
				fmt.Sprintf(
					"The X-axis is main-thread user turn number and the Y-axis is %s prompt-side tokens.",
					mode.TextLabel(),
				),
			),
			metricDetailLine(
				"Scope",
				"Excludes subagents, sidechains, system records, and assistant steps before the first real user prompt.",
			),
		)
	}
	if m.sessionsGrouped {
		return renderStatsMetricDetail(lane.title, width, groupedTurnMetricDetailChips(m),
			metricDetailLine(
				"Question",
				"How does full assistant-side turn cost differ across versions for the selected provider?",
			),
			metricDetailLine(
				"Reading",
				"Each stacked bar is one main-thread user turn position, with colors splitting the total by version.",
			),
			metricDetailLine("Scope", groupedTurnMetricScope(m)),
		)
	}
	mode := m.sessionsTurnCostMode
	metrics := m.sessionTurnMetricsForMode(mode)
	return renderStatsMetricDetail(lane.title, width, append([]chip{
		{Label: "stat", Value: mode.ShortLabel()},
	}, turnCostMetricDetailChips(metrics, mode)...),
		metricDetailLine(
			"Question",
			"How expensive does each main-thread user turn become once the full assistant-side cost is counted?",
		),
		metricDetailLine(
			"Reading",
			fmt.Sprintf(
				"The X-axis is main-thread user turn number and the Y-axis is %s total assistant tokens per turn.",
				mode.TextLabel(),
			),
		),
		metricDetailLine(
			"Scope",
			"Excludes subagents, sidechains, system records, and assistant steps before the first real user prompt.",
		),
	)
}

func (m statsModel) renderToolsMetricDetail(width int) string {
	lane, _, ok := m.selectedStatsLane()
	if !ok {
		return renderStatsMetricDetail("Tools", width, nil, noDataLabel)
	}
	if m.toolsGrouped {
		return m.renderGroupedToolsMetricDetail(width, lane)
	}

	tools := m.snapshot.Tools
	if lane.id == statsLaneToolsCalls {
		bucket, count := dominantHistogramBucket(tools.CallsPerSession)
		return renderStatsMetricDetail(lane.title, width, []chip{
			{Label: "dominant bucket", Value: bucket},
			{Label: "sessions", Value: statspkg.FormatNumber(count)},
		},
			metricDetailLine("Question", "Is tool use light and frequent or concentrated in a few heavy sessions?"),
			metricDetailLine("Reading", "The X-axis is call-count bucket and the Y-axis is session count."),
		)
	}
	if lane.id == statsLaneToolsTop {
		leader, count := leadingTool(tools.TopTools)
		return renderStatsMetricDetail(lane.title, width, []chip{
			{Label: "top tool", Value: leader},
			{Label: "calls", Value: statspkg.FormatNumber(count)},
		},
			metricDetailLine("Question", "Which tools dominate the workflow?"),
			metricDetailLine("Reading", "Longer bars mean more total calls in the active slice."),
		)
	}
	if lane.id == statsLaneToolsErrors {
		name, rate := topToolRate(tools.ToolErrorRates)
		return renderStatsMetricDetail(lane.title, width, []chip{
			{Label: "top rate", Value: name},
			{Label: "error rate", Value: rate},
		},
			metricDetailLine("Question", "Which tools are failing often enough to inspect?"),
			metricDetailLine(
				"Reading",
				"Rates exclude user-declined suggestions and show absolute failures alongside percentage.",
			),
		)
	}
	name, rate := topToolRate(tools.ToolRejectRates)
	return renderStatsMetricDetail(lane.title, width, []chip{
		{Label: "top rate", Value: name},
		{Label: "rejected", Value: rate},
	},
		metricDetailLine("Question", "Which proposed tools are users pushing back on before execution?"),
		metricDetailLine(
			"Reading",
			"Higher rates mean stronger user resistance to the suggested tool choice.",
		),
	)
}

func (m statsModel) renderPerformanceMetricDetail(width int) string {
	if !m.performanceScopeAllowsScorecard() {
		return renderPerformancePreviewMetricDetail(m, width)
	}

	metric, lane, _, ok := m.selectedPerformanceMetric()
	if !ok {
		return renderStatsMetricDetail("Performance", width, nil, "No performance metrics are available.")
	}
	return renderPerformanceMetricInspector(metric, lane, width)
}

func renderPerformancePreviewMetricDetail(m statsModel, width int) string {
	cards := performanceScopePreviewCards()
	if len(cards) == 0 {
		return renderStatsMetricDetail("Performance preview", width, nil, "No preview lanes are available.")
	}

	cursor := clampCursor(m.performanceLaneCursor, len(cards))
	card := cards[cursor]
	return renderStatsMetricDetail(card.Title, width, []chip{
		{Label: "need", Value: "1 provider + 1 model"},
		{Label: "lane", Value: card.Title},
	},
		metricDetailLine("Reading", "These are the metrics this lane will score once the scope is narrowed."),
		metricDetailLine("Includes", strings.Join(card.Metrics, ", ")),
	)
}
