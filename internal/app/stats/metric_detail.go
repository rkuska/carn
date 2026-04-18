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

func (m statsModel) renderStatsMetricDetailBody(title string, width int, chips []chip, lines ...string) string {
	innerWidth := max(width-4, 1)
	parts := []string{renderStatsTitle(m.theme, title)}
	if len(chips) > 0 {
		parts = append(parts, renderSummaryChips(m.theme, chips, innerWidth))
	}
	parts = append(parts, wrapStatsMetricDetailLines(lines, innerWidth)...)
	return strings.Join(parts, "\n")
}

func (m statsModel) metricDetailLine(label, value string) string {
	return m.theme.StyleMetaLabel.Render(label) + " " + m.theme.StyleMetaValue.Render(value)
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
		return m.renderStatsMetricDetailBody("Overview", width, nil, noDataLabel)
	}

	overview := m.snapshot.Overview
	if lane.id == statsLaneOverviewModel {
		leader, share := leadingModelDetail(overview)
		return m.renderStatsMetricDetailBody(lane.title, width, []chip{
			{Label: "leading model", Value: leader},
			{Label: "share", Value: share},
			{Label: "tokens", Value: statspkg.FormatNumber(overview.Tokens.Total)},
		},
			m.metricDetailLine("Question", "Which models are driving token use?"),
			m.metricDetailLine("Reading", "Longer bars mean more total tokens in the active slice."),
		)
	}
	if lane.id == statsLaneOverviewProject {
		leader, share := leadingProjectDetail(overview)
		return m.renderStatsMetricDetailBody(lane.title, width, []chip{
			{Label: "leading project", Value: leader},
			{Label: "share", Value: share},
			{Label: "tokens", Value: statspkg.FormatNumber(overview.Tokens.Total)},
		},
			m.metricDetailLine("Question", "Which projects are driving token use?"),
			m.metricDetailLine("Reading", "Longer bars mean more total tokens in the active slice."),
		)
	}
	if lane.id == statsLaneOverviewProviderVersion {
		items := m.snapshot.Overview.ByProviderVersion
		if len(items) == 0 {
			return m.renderStatsMetricDetailBody(lane.title, width, nil, noDataLabel)
		}
		total := 0
		for _, item := range items {
			total += item.Tokens
		}
		return m.renderStatsMetricDetailBody(lane.title, width, []chip{
			{Label: "providers", Value: statspkg.FormatNumber(providerVersionProviderCount(items))},
			{Label: "entries", Value: statspkg.FormatNumber(len(items))},
			{Label: "tokens", Value: statspkg.FormatNumber(total)},
		},
			m.metricDetailLine("Question", "Which provider/version pairs are driving tokens in the active range?"),
			m.metricDetailLine(
				"Reading",
				"Each row shows provider, version, a proportional bar, total tokens, and share within the filtered dataset.",
			),
			m.metricDetailLine("Scope", "Use filters to narrow provider, model, and version."),
		)
	}

	session, index, selected := m.selectedOverviewSession()
	if !selected {
		return m.renderStatsMetricDetailBody(
			lane.title,
			width,
			nil,
			"No token-heavy sessions are available.",
		)
	}
	return m.renderStatsMetricDetailBody(lane.title, width, []chip{
		{Label: "session", Value: formatFractionInt(index+1, len(overview.TopSessions))},
		{Label: "project", Value: session.Project},
		{Label: "tokens", Value: statspkg.FormatNumber(session.Tokens)},
	},
		m.metricDetailLine("Slug", session.Slug),
		m.metricDetailLine("Date", session.Timestamp.Format("2006-01-02")),
		m.metricDetailLine("Messages", statspkg.FormatNumber(session.MessageCount)),
		m.metricDetailLine("Duration", session.Duration.String()),
		"Enter opens the selected session.",
	)
}

func (m statsModel) renderActivityMetricDetail(width int) string {
	lane, _, ok := m.selectedStatsLane()
	if !ok {
		return m.renderStatsMetricDetailBody("Activity", width, nil, noDataLabel)
	}

	if lane.id == statsLaneActivityDaily {
		label := activityMetricName(m.activityMetric)
		_, counts := m.activitySeries()
		peakDay, peakCount := peakDailyCount(counts)
		return m.renderStatsMetricDetailBody("Daily Activity", width, []chip{
			{Label: "metric", Value: label},
			{Label: "peak day", Value: peakDay},
			{Label: "total", Value: statspkg.FormatNumber(totalDailyCount(counts))},
		},
			m.metricDetailLine("Question", "How does work volume change across the active range?"),
			m.metricDetailLine(
				"Reading",
				"The X-axis is calendar day and the Y-axis is the selected daily metric.",
			),
			"Press m to cycle sessions, messages, and tokens.",
			fmt.Sprintf("Peak day reached %s %s.", statspkg.FormatNumber(peakCount), strings.ToLower(label)),
		)
	}

	slot, count := busiestHeatmapSlot(m.snapshot.Activity.Heatmap)
	return m.renderStatsMetricDetailBody(lane.title, width, []chip{
		{Label: "busiest slot", Value: slot},
		{Label: "sessions", Value: statspkg.FormatNumber(count)},
	},
		m.metricDetailLine("Question", "When does work tend to happen?"),
		m.metricDetailLine(
			"Reading",
			"Rows are weekdays, columns are hours, and darker cells mean more sessions.",
		),
	)
}

func (m statsModel) renderSessionsMetricDetail(width int) string {
	lane, _, ok := m.selectedStatsLane()
	if !ok {
		return m.renderStatsMetricDetailBody("Sessions", width, nil, noDataLabel)
	}

	sessionStats := m.snapshot.Sessions
	if lane.id == statsLaneSessionsDuration {
		bucket, count := dominantHistogramBucket(sessionStats.DurationHistogram)
		return m.renderStatsMetricDetailBody(lane.title, width, []chip{
			{Label: "dominant bucket", Value: bucket},
			{Label: "sessions", Value: statspkg.FormatNumber(count)},
		},
			m.metricDetailLine("Question", "Are sessions mostly quick checks or long runs?"),
			m.metricDetailLine("Reading", "The X-axis is duration bucket and the Y-axis is session count."),
		)
	}
	if lane.id == statsLaneSessionsMessages {
		bucket, count := dominantHistogramBucket(sessionStats.MessageHistogram)
		return m.renderStatsMetricDetailBody(lane.title, width, []chip{
			{Label: "dominant bucket", Value: bucket},
			{Label: "sessions", Value: statspkg.FormatNumber(count)},
		},
			m.metricDetailLine("Question", "Do sessions stay short or turn into long exchanges?"),
			m.metricDetailLine("Reading", "The X-axis is message-count bucket and the Y-axis is session count."),
		)
	}
	if lane.id == statsLaneSessionsContext {
		return m.renderSessionTurnLaneDetail(
			lane,
			width,
			m.sessionsPromptMode,
			"How does prompt size grow as main-thread sessions go deeper?",
			"prompt-side tokens",
			"",
			promptMetricDetailChips,
		)
	}
	return m.renderSessionTurnLaneDetail(
		lane,
		width,
		m.sessionsTurnCostMode,
		"How expensive does each main-thread user turn become once the full assistant-side cost is counted?",
		"total assistant tokens per turn",
		"Sum across every assistant API call in the turn; cached prompts are counted in full,"+
			" so long tool loops can exceed the model's context window.",
		turnCostMetricDetailChips,
	)
}

func (m statsModel) renderSessionTurnLaneDetail(
	lane statsLane,
	width int,
	mode statspkg.StatisticMode,
	question, yAxisDescription, note string,
	metricChips func([]statspkg.PositionTokenMetrics, statspkg.StatisticMode) []chip,
) string {
	metrics := m.sessionTurnMetricsForMode(mode)
	chips := append([]chip{{Label: "stat", Value: mode.ShortLabel()}}, metricChips(metrics, mode)...)
	if m.splitActive() && m.splitBy.SupportsTurnMetrics() {
		chips = append(chips, splitTurnMetricDetailChips(m)...)
	}

	reading := fmt.Sprintf(
		"The X-axis is main-thread user turn number and the Y-axis is %s %s.",
		mode.TextLabel(),
		yAxisDescription,
	)
	if m.splitActive() && m.splitBy.SupportsTurnMetrics() {
		reading += " " + m.colorsStackSuffix()
	}

	scope := "Excludes subagents, sidechains, system records, and assistant steps before the first real user prompt."
	if m.splitActive() && m.splitBy.SupportsTurnMetrics() {
		scope += " " + splitTurnMetricScope(m)
	}

	lines := []string{
		m.metricDetailLine("Question", question),
		m.metricDetailLine("Reading", reading),
		m.metricDetailLine("Scope", scope),
	}
	if note != "" {
		lines = append(lines, m.metricDetailLine("Note", note))
	}
	return m.renderStatsMetricDetailBody(lane.title, width, chips, lines...)
}

func (m statsModel) renderToolsMetricDetail(width int) string {
	lane, _, ok := m.selectedStatsLane()
	if !ok {
		return m.renderStatsMetricDetailBody("Tools", width, nil, noDataLabel)
	}
	if m.splitActive() {
		return m.renderSplitToolsMetricDetail(width, lane)
	}

	tools := m.snapshot.Tools
	if lane.id == statsLaneToolsCalls {
		bucket, count := dominantHistogramBucket(tools.CallsPerSession)
		return m.renderStatsMetricDetailBody(lane.title, width, []chip{
			{Label: "dominant bucket", Value: bucket},
			{Label: "sessions", Value: statspkg.FormatNumber(count)},
		},
			m.metricDetailLine("Question", "Is tool use light and frequent or concentrated in a few heavy sessions?"),
			m.metricDetailLine("Reading", "The X-axis is call-count bucket and the Y-axis is session count."),
		)
	}
	if lane.id == statsLaneToolsTop {
		leader, count := leadingTool(tools.TopTools)
		return m.renderStatsMetricDetailBody(lane.title, width, []chip{
			{Label: "top tool", Value: leader},
			{Label: "calls", Value: statspkg.FormatNumber(count)},
		},
			m.metricDetailLine("Question", "Which tools dominate the workflow?"),
			m.metricDetailLine("Reading", "Longer bars mean more total calls in the active slice."),
		)
	}
	if lane.id == statsLaneToolsErrors {
		name, rate := topToolRate(tools.ToolErrorRates)
		return m.renderStatsMetricDetailBody(lane.title, width, []chip{
			{Label: "top rate", Value: name},
			{Label: "error rate", Value: rate},
		},
			m.metricDetailLine("Question", "Which tools are failing often enough to inspect?"),
			m.metricDetailLine(
				"Reading",
				"Rates exclude user-declined suggestions and show absolute failures alongside percentage.",
			),
		)
	}
	name, rate := topToolRate(tools.ToolRejectRates)
	return m.renderStatsMetricDetailBody(lane.title, width, []chip{
		{Label: "top rate", Value: name},
		{Label: "rejected", Value: rate},
	},
		m.metricDetailLine("Question", "Which proposed tools are users pushing back on before execution?"),
		m.metricDetailLine(
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
		return m.renderStatsMetricDetailBody("Performance", width, nil, "No performance metrics are available.")
	}
	return renderPerformanceMetricInspector(m.theme, metric, lane, width)
}

func renderPerformancePreviewMetricDetail(m statsModel, width int) string {
	cards := performanceScopePreviewCards()
	if len(cards) == 0 {
		return m.renderStatsMetricDetailBody("Performance preview", width, nil, "No preview lanes are available.")
	}

	cursor := clampCursor(m.performanceLaneCursor, len(cards))
	card := cards[cursor]
	return m.renderStatsMetricDetailBody(card.Title, width, []chip{
		{Label: "need", Value: "1 provider + 1 model"},
		{Label: "lane", Value: card.Title},
	},
		m.metricDetailLine("Reading", "These are the metrics this lane will score once the scope is narrowed."),
		m.metricDetailLine("Includes", strings.Join(card.Metrics, ", ")),
	)
}
