package app

import (
	"fmt"
	"strings"

	statspkg "github.com/rkuska/carn/internal/stats"
)

const noDataLabel = "No data"

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
	parts = append(parts, lines...)
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
	if lane.id == statsLaneOverviewProj {
		leader, share := leadingProjectDetail(overview)
		return renderStatsMetricDetail(lane.title, width, []chip{
			{Label: "leading project", Value: leader},
			{Label: "share", Value: share},
			{Label: "tokens", Value: statspkg.FormatNumber(overview.Tokens.Total)},
		},
			metricDetailLine("Question", "Which projects are consuming the most tokens?"),
			metricDetailLine("Reading", "Longer bars mean more total tokens in the active slice."),
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
		{Label: "session", Value: fmt.Sprintf("%d/%d", index+1, len(overview.TopSessions))},
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
		return renderStatsMetricDetail(lane.title, width, contextMetricDetailChips(m.claudeTurnMetrics),
			metricDetailLine("Question", "How quickly does prompt context accumulate as sessions go deeper?"),
			metricDetailLine(
				"Reading",
				"The X-axis is user turn number and the Y-axis is average maximum input tokens.",
			),
		)
	}
	return renderStatsMetricDetail(lane.title, width, turnCostMetricDetailChips(m.claudeTurnMetrics),
		metricDetailLine(
			"Question",
			"How expensive does each turn become once prompt and response are counted together?",
		),
		metricDetailLine(
			"Reading",
			"The X-axis is user turn number and the Y-axis is average input plus output tokens.",
		),
	)
}

func (m statsModel) renderToolsMetricDetail(width int) string {
	lane, _, ok := m.selectedStatsLane()
	if !ok {
		return renderStatsMetricDetail("Tools", width, nil, noDataLabel)
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

func leadingModelDetail(overview statspkg.Overview) (string, string) {
	if len(overview.ByModel) == 0 {
		return noDataLabel, "0%"
	}
	item := overview.ByModel[0]
	return item.Model, formatPercent(item.Tokens, overview.Tokens.Total)
}

func leadingProjectDetail(overview statspkg.Overview) (string, string) {
	if len(overview.ByProject) == 0 {
		return noDataLabel, "0%"
	}
	item := overview.ByProject[0]
	return item.Project, formatPercent(item.Tokens, overview.Tokens.Total)
}

func formatPercent(part, total int) string {
	if total <= 0 || part <= 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", float64(part)/float64(total)*100)
}

func activityMetricName(metric activityMetric) string {
	switch metric {
	case metricSessions:
		return "Sessions"
	case metricMessages:
		return "Messages"
	case metricTokens:
		return "Tokens"
	}
	return "Sessions"
}

func peakDailyCount(counts []statspkg.DailyCount) (string, int) {
	if len(counts) == 0 {
		return noDataLabel, 0
	}

	peak := counts[0]
	for _, count := range counts[1:] {
		if count.Count > peak.Count {
			peak = count
		}
	}
	return peak.Date.Format("2006-01-02"), peak.Count
}

func totalDailyCount(counts []statspkg.DailyCount) int {
	total := 0
	for _, count := range counts {
		total += count.Count
	}
	return total
}

func busiestHeatmapSlot(cells [7][24]int) (string, int) {
	dayNames := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	bestDay, bestHour, bestCount := 0, 0, 0
	for day := range 7 {
		for hour := range 24 {
			if cells[day][hour] > bestCount {
				bestDay = day
				bestHour = hour
				bestCount = cells[day][hour]
			}
		}
	}
	if bestCount == 0 {
		return noDataLabel, 0
	}
	return fmt.Sprintf("%s %02d:00", dayNames[bestDay], bestHour), bestCount
}

func dominantHistogramBucket(buckets []statspkg.HistogramBucket) (string, int) {
	if len(buckets) == 0 {
		return noDataLabel, 0
	}
	best := buckets[0]
	for _, bucket := range buckets[1:] {
		if bucket.Count > best.Count {
			best = bucket
		}
	}
	return best.Label, best.Count
}

func leadingTool(tools []statspkg.ToolStat) (string, int) {
	if len(tools) == 0 {
		return noDataLabel, 0
	}
	return tools[0].Name, tools[0].Count
}

func topToolRate(rates []statspkg.ToolRateStat) (string, string) {
	if len(rates) == 0 {
		return noDataLabel, "0.0%"
	}
	return rates[0].Name, formatToolRatePercent(rates[0].Rate)
}

func formatRatio(value float64) string {
	return fmt.Sprintf("%.1f:1", value)
}

func contextMetricDetailChips(metrics []statspkg.PositionTokenMetrics) []chip {
	early := averageTurnMetricRange(metrics, 1, 5, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageInputTokens
	})
	late := averageTurnMetricRange(metrics, 20, 999, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageInputTokens
	})
	return []chip{
		{Label: statsClaudeContextEarlyLabel, Value: formatFloat(early)},
		{Label: statsClaudeContextLateLabel, Value: formatFloat(late)},
		{Label: statsClaudeContextFactorLabel, Value: formatTurnMetricMultiplier(early, late)},
	}
}

func turnCostMetricDetailChips(metrics []statspkg.PositionTokenMetrics) []chip {
	early := averageTurnMetricRange(metrics, 1, 5, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageTurnTokens
	})
	late := averageTurnMetricRange(metrics, 20, 999, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageTurnTokens
	})
	return []chip{
		{Label: statsClaudeTurnCostEarlyLabel, Value: formatFloat(early)},
		{Label: statsClaudeTurnCostLateLabel, Value: formatFloat(late)},
		{Label: statsClaudeTurnCostFactorLabel, Value: formatTurnMetricMultiplier(early, late)},
	}
}
