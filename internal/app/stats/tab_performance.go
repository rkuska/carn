package stats

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	el "github.com/rkuska/carn/internal/app/elements"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderPerformanceTab(width int) string {
	performance := m.snapshot.Performance
	sections := make([]string, 0, 8)
	if !m.performanceScopeAllowsScorecard() {
		sections = append(sections, renderPerformanceScopeGate(m, width))
		return strings.Join(sections, "\n\n")
	}

	sections = append(sections, renderPerformanceHeadline(m.theme, performance, width))

	sections = append(sections,
		renderSummaryChips(m.theme, performanceScoreChips(performance), width),
		renderSummaryChips(m.theme, performanceScopeChips(performance), width),
	)

	sections = append(sections, renderPerformanceCards(m, width))
	sections = append(sections, renderPerformanceDiagnostics(m.theme, performance, width))

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

func renderPerformanceHeadline(theme *el.Theme, performance statspkg.Performance, width int) string {
	scope := performance.Scope
	innerWidth := max(width-4, 1)
	lines := []string{
		renderStatsTitle(theme, performanceVerdictText(performance.Overall)+" relative to baseline"),
		fmt.Sprintf("scope %s", performanceScopeSummary(scope)),
		renderSummaryChips(theme, []chip{
			{Label: "current", Value: formatPerformanceTimeRange(scope.CurrentRange)},
			{Label: "baseline", Value: formatPerformanceTimeRange(scope.BaselineRange)},
		}, innerWidth),
	}
	return renderFramedBox(theme, "Performance verdict", width, theme.ColorPrimary, strings.Join(lines, "\n"))
}

func renderPerformanceCards(m statsModel, width int) string {
	cards := m.performanceLanes()
	selectedLane := m.performanceLaneCursor
	bodyHeight := performanceLaneCardsBodyHeight(cards[:])
	return renderStatsLaneGrid(m.theme, width, 36, selectedLane, func(index, width int, selected bool) string {
		return renderPerformanceLaneCard(
			m.theme,
			cards[index],
			selected,
			metricCursorForLane(selectedLane, index, m.performanceMetricCursor),
			width,
			bodyHeight,
		)
	})
}

func metricCursorForLane(selectedLane, laneIndex, metricCursor int) int {
	if selectedLane == laneIndex {
		return metricCursor
	}
	return 0
}

func renderPerformanceLaneCard(
	theme *el.Theme,
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

	lines := []string{
		lane.Detail,
		theme.StyleMetaLabel.Render("verdict") + " " + theme.StyleMetaValue.Render(
			performanceVerdictText(statspkg.PerformanceScore{
				Score:    lane.Score,
				HasScore: lane.HasScore,
				Trend:    lane.Trend,
			}),
		),
	}

	selectedMetricID := ""
	if selectedMetricIndex >= 0 && selectedMetricIndex < len(lane.Metrics) {
		selectedMetricID = lane.Metrics[selectedMetricIndex].ID
	}
	for _, metric := range lane.Metrics {
		lines = append(lines, renderPerformanceMetricRow(metric, metric.ID == selectedMetricID, width-4))
	}
	return renderStatsLanePane(theme, title, selected, width, bodyHeight, strings.Join(lines, "\n"))
}

func performanceLaneCardsBodyHeight(lanes []statspkg.PerformanceLane) int {
	height := 0
	for _, lane := range lanes {
		height = max(height, performanceLaneCardBodyHeight(lane))
	}
	return height
}

func performanceLaneCardBodyHeight(lane statspkg.PerformanceLane) int {
	return 2 + len(lane.Metrics)
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
	theme *el.Theme,
	metric statspkg.PerformanceMetric,
	lane statspkg.PerformanceLane,
	width int,
) string {
	innerWidth := max(width-4, 1)
	lines := []string{
		renderStatsTitle(theme, metric.Label+"  "+metric.Value),
		theme.StyleMetaLabel.Render("Question") + " " + theme.StyleMetaValue.Render(metric.Question),
		theme.StyleMetaLabel.Render("Formula") + " " + theme.StyleMetaValue.Render(metric.Formula),
		renderSummaryChips(theme, []chip{
			{Label: "lane", Value: lane.Label},
			{Label: "baseline", Value: performanceBaselineValue(metric)},
			{Label: "delta", Value: performanceDelta(metric)},
			{Label: "status", Value: performanceMetricStatusText(metric.Status)},
			{Label: "better when", Value: performanceDirection(metric.HigherIsBetter)},
			{Label: "samples", Value: statspkg.FormatNumber(metric.SampleCount)},
		}, innerWidth),
		renderStatsTitle(theme, "Trend"),
		renderSparkline(metric.Series, max(innerWidth, 8)),
	}
	return strings.Join(lines, "\n")
}

func renderPerformanceDiagnostics(theme *el.Theme, performance statspkg.Performance, width int) string {
	leftWidth, rightWidth, stacked := statsColumnWidths(width, 1, 1, 36)
	return renderColumns(
		theme,
		renderPerformanceLikelyCauses(theme, performance, leftWidth),
		renderPerformanceProviderSignals(theme, performance.Diagnostics, rightWidth),
		leftWidth,
		rightWidth,
		stacked,
	)
}

func renderPerformanceLikelyCauses(theme *el.Theme, performance statspkg.Performance, width int) string {
	lines := []string{renderStatsTitle(theme, "Likely causes")}
	for _, cause := range performanceLikelyCauses(performance) {
		lines = append(lines, ansi.Truncate(cause, width, "…"))
	}
	return strings.Join(lines, "\n")
}

func renderPerformanceProviderSignals(
	theme *el.Theme,
	diagnostics []statspkg.PerformanceDiagnostic,
	width int,
) string {
	lines := []string{renderStatsTitle(theme, "Provider signals")}
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
