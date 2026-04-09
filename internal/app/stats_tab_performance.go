package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderPerformanceTab(width int) string {
	performance := m.snapshot.Performance
	loadingSequence := m.performanceSequenceLoading()
	summary := renderSummaryChips([]chip{
		{Label: "overall", Value: formatPerformanceScore(performance.Overall)},
		{Label: "outcome", Value: formatPerformanceLaneScore(performance.Outcome)},
		{Label: "discipline", Value: formatPerformanceLaneScore(performance.Discipline)},
		{Label: "efficiency", Value: formatPerformanceLaneScore(performance.Efficiency)},
		{Label: "robustness", Value: formatPerformanceLaneScore(performance.Robustness)},
	}, width)
	scope := renderSummaryChips(performanceScopeChips(performance), width)

	leftWidth, rightWidth, stacked := statsColumnWidths(width, 1, 1, 32)
	top := renderColumns(
		renderPerformanceLane(performance.Outcome, leftWidth),
		renderPerformanceLane(performance.Discipline, rightWidth),
		leftWidth,
		rightWidth,
		stacked,
	)
	bottom := renderColumns(
		renderPerformanceLane(performance.Efficiency, leftWidth),
		renderPerformanceLane(performance.Robustness, rightWidth),
		leftWidth,
		rightWidth,
		stacked,
	)
	trends := renderColumns(
		renderPerformanceTrendChart(primaryPerformanceMetric(performance.Outcome, "verification pass"), leftWidth),
		renderPerformanceTrendChart(primaryPerformanceMetric(performance.Discipline, "read before write"), rightWidth),
		leftWidth,
		rightWidth,
		stacked,
	) + "\n\n" + renderColumns(
		renderPerformanceTrendChart(primaryPerformanceMetric(performance.Efficiency, "tokens / user turn"), leftWidth),
		renderPerformanceTrendChart(primaryPerformanceMetric(performance.Robustness, "tool error rate"), rightWidth),
		leftWidth,
		rightWidth,
		stacked,
	)

	diagnostics := renderPerformanceDiagnostics(performance.Diagnostics, width)
	loading := ""
	if loadingSequence {
		loading = m.spinner.View() + " Loading transcript sequence metrics..."
	}

	sections := []string{summary, scope}
	if loading != "" {
		sections = append(sections, loading)
	}
	sections = append(sections, top, bottom, trends, diagnostics)
	return strings.Join(sections, "\n\n")
}

func performanceScopeChips(performance statspkg.Performance) []chip {
	chips := []chip{
		{Label: "sessions", Value: statspkg.FormatNumber(performance.Scope.SessionCount)},
		{Label: "baseline", Value: statspkg.FormatNumber(performance.Scope.BaselineSessionCount)},
	}
	if len(performance.Scope.Providers) > 0 {
		chips = append(chips, chip{Label: "providers", Value: strings.Join(performance.Scope.Providers, ",")})
	}
	if len(performance.Scope.Models) > 0 {
		chips = append(chips, chip{Label: "models", Value: strings.Join(performance.Scope.Models, ",")})
	}
	if performance.Scope.SequenceLoaded {
		chips = append(chips, chip{Label: "sequence", Value: statspkg.FormatNumber(performance.Scope.SequenceSampleCount)})
	}
	return chips
}

func renderPerformanceLane(lane statspkg.PerformanceLane, width int) string {
	if width <= 0 {
		return ""
	}
	lines := []string{
		renderStatsTitle(fmt.Sprintf("%s %s", lane.Label, formatPerformanceLaneScore(lane))),
		lane.Detail,
	}
	for _, metric := range lane.Metrics {
		lines = append(lines, renderPerformanceMetric(metric, width))
	}
	return strings.Join(lines, "\n")
}

func renderPerformanceMetric(metric statspkg.PerformanceMetric, width int) string {
	labelWidth := min(max(width/3, 14), 20)
	valueWidth := min(max(width/5, 8), 14)
	trendWidth := 2
	sparkWidth := max(width-labelWidth-valueWidth-trendWidth-3, 6)

	label := fitToWidth(ansi.Truncate(metric.Label, labelWidth, "…"), labelWidth)
	value := fitToWidth(metric.Value, valueWidth)
	return ansi.Truncate(
		label+" "+renderSparkline(metric.Series, sparkWidth)+" "+fitToWidth(trendGlyph(metric.Trend), trendWidth)+" "+value,
		width,
		"…",
	)
}

func renderPerformanceTrendChart(metric statspkg.PerformanceMetric, width int) string {
	if width <= 0 {
		return ""
	}
	lines := []string{
		renderStatsTitle(metric.Label),
		metric.Detail,
		renderSparkline(metric.Series, max(width-1, 8)),
		metric.Value + " " + trendGlyph(metric.Trend),
	}
	return strings.Join(lines, "\n")
}

func renderPerformanceDiagnostics(diagnostics []statspkg.PerformanceDiagnostic, width int) string {
	lines := []string{renderStatsTitle("Diagnostics")}
	if len(diagnostics) == 0 {
		lines = append(lines, "No data")
		return strings.Join(lines, "\n")
	}

	for _, diagnostic := range diagnostics {
		line := fmt.Sprintf("%s %s %s", diagnostic.Label, trendGlyph(diagnostic.Trend), diagnostic.Value)
		lines = append(lines, ansi.Truncate(line, width, "…"))
	}
	return strings.Join(lines, "\n")
}

func formatPerformanceScore(score statspkg.PerformanceScore) string {
	if !score.HasScore {
		return "n/a"
	}
	return fmt.Sprintf("%d %s", score.Score, trendGlyph(score.Trend))
}

func formatPerformanceLaneScore(lane statspkg.PerformanceLane) string {
	return formatPerformanceScore(statspkg.PerformanceScore{
		Score:    lane.Score,
		HasScore: lane.HasScore,
		Trend:    lane.Trend,
	})
}

func trendGlyph(direction statspkg.TrendDirection) string {
	switch direction {
	case statspkg.TrendDirectionNone:
		return "·"
	case statspkg.TrendDirectionUp:
		return "↑"
	case statspkg.TrendDirectionDown:
		return "↓"
	case statspkg.TrendDirectionFlat:
		return "→"
	}
	return "·"
}

func primaryPerformanceMetric(lane statspkg.PerformanceLane, fallback string) statspkg.PerformanceMetric {
	if len(lane.Metrics) == 0 {
		return statspkg.PerformanceMetric{Label: fallback}
	}
	return lane.Metrics[0]
}
