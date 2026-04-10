package app

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderPerformanceTab(width int) string {
	m = m.normalizePerformanceSelection()
	performance := m.snapshot.Performance
	if !m.performanceScopeAllowsScorecard() {
		return renderPerformanceScopeGate(performance.Scope, width)
	}

	loadingSequence := m.performanceSequenceLoading()
	sections := []string{renderPerformanceHeadline(performance, width)}

	sections = append(sections,
		renderSummaryChips(performanceScoreChips(performance), width),
		renderSummaryChips(performanceScopeChips(performance), width),
	)

	if loadingSequence {
		sections = append(sections, m.spinner.View()+" Loading transcript sequence metrics...")
	}

	sections = append(sections, renderPerformanceCards(m, width))

	if metric, lane, _, ok := m.selectedPerformanceMetric(); ok {
		sections = append(sections, renderPerformanceMetricInspector(metric, lane, width))
	}
	sections = append(sections, renderPerformanceDiagnostics(performance, width))

	return strings.Join(sections, "\n\n")
}

func performanceScoreChips(performance statspkg.Performance) []chip {
	return []chip{
		{Label: "overall", Value: formatPerformanceScore(performance.Overall)},
		{Label: "outcome", Value: formatPerformanceLaneScore(performance.Outcome)},
		{Label: "discipline", Value: formatPerformanceLaneScore(performance.Discipline)},
		{Label: "efficiency", Value: formatPerformanceLaneScore(performance.Efficiency)},
		{Label: "robustness", Value: formatPerformanceLaneScore(performance.Robustness)},
	}
}

func performanceScopeChips(performance statspkg.Performance) []chip {
	chips := []chip{
		{Label: "sessions", Value: statspkg.FormatNumber(performance.Scope.SessionCount)},
		{Label: "baseline", Value: statspkg.FormatNumber(performance.Scope.BaselineSessionCount)},
		{Label: "current", Value: formatPerformanceTimeRange(performance.Scope.CurrentRange)},
		{Label: "baseline window", Value: formatPerformanceTimeRange(performance.Scope.BaselineRange)},
	}
	if performance.Scope.SequenceLoaded {
		chips = append(chips, chip{Label: "sequence", Value: statspkg.FormatNumber(performance.Scope.SequenceSampleCount)})
	}
	return chips
}

func renderPerformanceHeadline(performance statspkg.Performance, width int) string {
	scope := performance.Scope
	innerWidth := max(width-4, 1)
	lines := []string{
		renderStatsTitle(performanceVerdictText(performance.Overall) + " relative to baseline"),
		fmt.Sprintf("scope %s", performanceScopeSummary(scope)),
		renderSummaryChips([]chip{
			{Label: "current", Value: formatPerformanceTimeRange(scope.CurrentRange)},
			{Label: "baseline", Value: formatPerformanceTimeRange(scope.BaselineRange)},
		}, innerWidth),
	}
	return renderFramedBox("Performance verdict", width, colorPrimary, strings.Join(lines, "\n"))
}

func renderPerformanceCards(m statsModel, width int) string {
	leftWidth, rightWidth, stacked := statsColumnWidths(width, 1, 1, 36)
	cards := m.performanceLanes()
	selectedLane := m.performanceLaneCursor
	bodyHeight := performanceLaneCardsBodyHeight(cards)

	top := renderColumns(
		renderPerformanceLaneCard(
			cards[0],
			selectedLane == 0,
			metricCursorForLane(selectedLane, 0, m.performanceMetricCursor),
			leftWidth,
			bodyHeight,
		),
		renderPerformanceLaneCard(
			cards[1],
			selectedLane == 1,
			metricCursorForLane(selectedLane, 1, m.performanceMetricCursor),
			rightWidth,
			bodyHeight,
		),
		leftWidth,
		rightWidth,
		stacked,
	)
	bottom := renderColumns(
		renderPerformanceLaneCard(
			cards[2],
			selectedLane == 2,
			metricCursorForLane(selectedLane, 2, m.performanceMetricCursor),
			leftWidth,
			bodyHeight,
		),
		renderPerformanceLaneCard(
			cards[3],
			selectedLane == 3,
			metricCursorForLane(selectedLane, 3, m.performanceMetricCursor),
			rightWidth,
			bodyHeight,
		),
		leftWidth,
		rightWidth,
		stacked,
	)
	return top + "\n\n" + bottom
}

func metricCursorForLane(selectedLane, laneIndex, metricCursor int) int {
	if selectedLane == laneIndex {
		return metricCursor
	}
	return 0
}

func renderPerformanceLaneCard(
	lane statspkg.PerformanceLane,
	selected bool,
	selectedMetricIndex int,
	width int,
	bodyHeight int,
) string {
	if width <= 0 {
		return ""
	}

	title := lane.Label + " " + formatPerformanceLaneScore(lane)
	if selected {
		title = "▸ " + title
	}

	lines := []string{
		lane.Detail,
		styleMetaLabel.Render("verdict") + " " + styleMetaValue.Render(performanceVerdictText(statspkg.PerformanceScore{
			Score:    lane.Score,
			HasScore: lane.HasScore,
			Trend:    lane.Trend,
		})),
	}

	selectedMetricID := ""
	if selectedMetricIndex >= 0 && selectedMetricIndex < len(lane.Metrics) {
		selectedMetricID = lane.Metrics[selectedMetricIndex].ID
	}
	for _, metric := range performanceVisibleMetrics(lane, selectedMetricIndex) {
		lines = append(lines, renderPerformanceMetricRow(metric, metric.ID == selectedMetricID, width-4))
	}
	return renderFramedPane(title, width, bodyHeight, laneBorderColor(selected), strings.Join(lines, "\n"))
}

func performanceLaneCardsBodyHeight(lanes []statspkg.PerformanceLane) int {
	height := 0
	for _, lane := range lanes {
		height = max(height, performanceLaneCardBodyHeight(lane))
	}
	return height
}

func performanceLaneCardBodyHeight(lane statspkg.PerformanceLane) int {
	return 2 + len(performanceVisibleMetrics(lane, 0))
}

func laneBorderColor(selected bool) color.Color {
	if selected {
		return colorAccent
	}
	return colorPrimary
}

func renderPerformanceMetricRow(metric statspkg.PerformanceMetric, selected bool, width int) string {
	marker := " "
	if selected {
		marker = ">"
	}

	labelWidth := min(max(width/4, 12), 20)
	valueWidth := min(max(width/8, 8), 10)
	deltaWidth := min(max(width/7, 8), 11)
	statusWidth := min(max(width/8, 9), 11)
	sparkWidth := max(width-labelWidth-valueWidth-deltaWidth-statusWidth-8, 6)

	label := fitToWidth(ansi.Truncate(metric.Label, labelWidth, "…"), labelWidth)
	value := fitToWidth(metric.Value, valueWidth)
	delta := fitToWidth(performanceDelta(metric), deltaWidth)
	status := fitToWidth(performanceMetricStatusText(metric.Status), statusWidth)
	line := fmt.Sprintf(
		"%s %s %s %s %s %s",
		marker,
		label,
		renderSparkline(metric.Series, sparkWidth),
		value,
		delta,
		status,
	)
	return ansi.Truncate(line, width, "…")
}

func renderPerformanceMetricInspector(
	metric statspkg.PerformanceMetric,
	lane statspkg.PerformanceLane,
	width int,
) string {
	innerWidth := max(width-4, 1)
	lines := []string{
		renderStatsTitle(metric.Label + "  " + metric.Value),
		styleMetaLabel.Render("Question") + " " + styleMetaValue.Render(metric.Question),
		styleMetaLabel.Render("Formula") + " " + styleMetaValue.Render(metric.Formula),
		renderSummaryChips([]chip{
			{Label: "lane", Value: lane.Label},
			{Label: "baseline", Value: performanceBaselineValue(metric)},
			{Label: "delta", Value: performanceDelta(metric)},
			{Label: "status", Value: performanceMetricStatusText(metric.Status)},
			{Label: "better when", Value: performanceDirection(metric.HigherIsBetter)},
			{Label: "samples", Value: statspkg.FormatNumber(metric.SampleCount)},
		}, innerWidth),
		renderStatsTitle("Trend"),
		renderSparkline(metric.Series, max(innerWidth, 8)),
	}
	return renderFramedBox("Metric detail", width, colorPrimary, strings.Join(lines, "\n"))
}

func renderPerformanceDiagnostics(performance statspkg.Performance, width int) string {
	leftWidth, rightWidth, stacked := statsColumnWidths(width, 1, 1, 36)
	return renderColumns(
		renderPerformanceLikelyCauses(performance, leftWidth),
		renderPerformanceProviderSignals(performance.Diagnostics, rightWidth),
		leftWidth,
		rightWidth,
		stacked,
	)
}

func renderPerformanceLikelyCauses(performance statspkg.Performance, width int) string {
	lines := []string{renderStatsTitle("Likely causes")}
	for _, cause := range performanceLikelyCauses(performance) {
		lines = append(lines, ansi.Truncate(cause, width, "…"))
	}
	return strings.Join(lines, "\n")
}

func renderPerformanceProviderSignals(
	diagnostics []statspkg.PerformanceDiagnostic,
	width int,
) string {
	lines := []string{renderStatsTitle("Provider signals")}
	if len(diagnostics) == 0 {
		lines = append(lines, "No diagnostic signals")
		return strings.Join(lines, "\n")
	}
	valueWidth := performanceProviderSignalValueWidth(diagnostics, width)
	arrowWidth := lipgloss.Width(trendGlyph(statspkg.TrendDirectionFlat))
	labelWidth := max(width-valueWidth-arrowWidth-2, 1)
	for _, diagnostic := range diagnostics {
		label := fitToWidth(ansi.Truncate(diagnostic.Label, labelWidth, "…"), labelWidth)
		value := alignRightToWidth(ansi.Truncate(diagnostic.Value, valueWidth, "…"), valueWidth)
		lines = append(lines, label+" "+value+" "+trendGlyph(diagnostic.Trend))
	}
	return strings.Join(lines, "\n")
}

func performanceProviderSignalValueWidth(
	diagnostics []statspkg.PerformanceDiagnostic,
	width int,
) int {
	maxWidth := min(max(width/4, 10), 16)
	for _, diagnostic := range diagnostics {
		maxWidth = max(maxWidth, lipgloss.Width(diagnostic.Value))
	}
	return min(maxWidth, max(width-4, 1))
}

func alignRightToWidth(s string, width int) string {
	stringWidth := lipgloss.Width(s)
	if stringWidth >= width {
		return s
	}
	return strings.Repeat(" ", width-stringWidth) + s
}

func performanceLikelyCauses(performance statspkg.Performance) []string {
	lines := make([]string, 0, 3)
	for _, lane := range []statspkg.PerformanceLane{
		performance.Outcome,
		performance.Discipline,
		performance.Efficiency,
		performance.Robustness,
	} {
		for _, metric := range lane.Metrics {
			if metric.Status != statspkg.PerformanceMetricStatusWorse {
				continue
			}
			lines = append(lines, fmt.Sprintf(
				"%s is worse than baseline (%s, %s).",
				metric.Label,
				performanceDelta(metric),
				metric.Value,
			))
			if len(lines) == 3 {
				return lines
			}
		}
	}
	if len(lines) == 0 {
		return []string{"No clear score-driving regressions in the current window."}
	}
	return lines
}
