package app

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	wlc "github.com/NimbleMarkets/ntcharts/v2/linechart/wavelinechart"

	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

const (
	statsLoadingText               = "Loading..."
	statsClaudeContextGrowthTitle  = "Context Growth"
	statsClaudeTurnCostTitle       = "Turn Cost"
	statsNoClaudeTurnMetricsData   = "No message usage data"
	statsClaudeMetricsNoDataLabel  = "turn metrics"
	statsClaudeContextEarlyLabel   = "context 1-5 avg"
	statsClaudeContextLateLabel    = "context 20+ avg"
	statsClaudeContextFactorLabel  = "context multiplier"
	statsClaudeTurnCostEarlyLabel  = "turn cost 1-5 avg"
	statsClaudeTurnCostLateLabel   = "turn cost 20+ avg"
	statsClaudeTurnCostFactorLabel = "turn cost multiplier"
)

func (m statsModel) renderSessionsTab(width int) string {
	sessionStats := m.snapshot.Sessions
	loadingClaudeTurnMetrics := m.claudeTurnMetricsLoading()
	chips := renderSummaryChips([]chip{
		{Label: "avg duration", Value: conv.FormatDuration(sessionStats.AverageDuration)},
		{Label: "avg messages", Value: formatFloat(sessionStats.AverageMessages)},
		{Label: "user:assistant", Value: formatRatio(sessionStats.UserAssistantRatio)},
		{Label: "abandoned", Value: fmt.Sprintf("%d (%.1f%%)", sessionStats.AbandonedCount, sessionStats.AbandonedRate)},
	}, width)

	durationBuckets := make([]histBucket, 0, len(sessionStats.DurationHistogram))
	for _, bucket := range sessionStats.DurationHistogram {
		durationBuckets = append(durationBuckets, histBucket{Label: bucket.Label, Count: bucket.Count})
	}
	messageBuckets := make([]histBucket, 0, len(sessionStats.MessageHistogram))
	for _, bucket := range sessionStats.MessageHistogram {
		messageBuckets = append(messageBuckets, histBucket{Label: bucket.Label, Count: bucket.Count})
	}

	histograms := renderSideBySide(
		renderVerticalHistogramWithColor("Session Duration", durationBuckets, max((width-3)/2, 30), 8, colorChartTime),
		renderVerticalHistogram("Messages per Session", messageBuckets, max((width-3)/2, 30), 8),
		width,
	)

	growthChips := renderSummaryChips(claudeTurnMetricChips(m.claudeTurnMetrics, loadingClaudeTurnMetrics), width)
	growthCharts := statsLoadingText
	switch {
	case loadingClaudeTurnMetrics:
		growthCharts = fmt.Sprintf("%s Computing turn charts...", m.spinner.View())
	case m.claudeTurnMetrics != nil:
		growthChartWidth := max(width-2, 12)
		if (width-3)/2 >= 30 {
			growthChartWidth = max((width-3)/2, 30)
		}
		growthCharts = renderSideBySide(
			renderClaudeTurnChart(
				statsClaudeContextGrowthTitle,
				m.claudeTurnMetrics,
				growthChartWidth,
				10,
				colorChartToken,
				func(metric statspkg.PositionTokenMetrics) float64 {
					return metric.AverageInputTokens
				},
			),
			renderClaudeTurnChart(
				statsClaudeTurnCostTitle,
				m.claudeTurnMetrics,
				growthChartWidth,
				10,
				colorChartToken,
				func(metric statspkg.PositionTokenMetrics) float64 {
					return metric.AverageTurnTokens
				},
			),
			width,
		)
	}
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", chips, histograms, growthChips, growthCharts)
}

func claudeTurnMetricChips(metrics []statspkg.PositionTokenMetrics, loading bool) []chip {
	if loading {
		return []chip{{Label: statsClaudeMetricsNoDataLabel, Value: statsLoadingText}}
	}
	if len(metrics) == 0 {
		return []chip{{Label: statsClaudeMetricsNoDataLabel, Value: "No data"}}
	}

	contextFirstFive := averageTurnMetricRange(metrics, 1, 5, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageInputTokens
	})
	contextTwentyPlus := averageTurnMetricRange(metrics, 20, 999, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageInputTokens
	})
	turnCostFirstFive := averageTurnMetricRange(metrics, 1, 5, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageTurnTokens
	})
	turnCostTwentyPlus := averageTurnMetricRange(metrics, 20, 999, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageTurnTokens
	})
	return []chip{
		{Label: statsClaudeContextEarlyLabel, Value: formatFloat(contextFirstFive)},
		{Label: statsClaudeContextLateLabel, Value: formatFloat(contextTwentyPlus)},
		{Label: statsClaudeContextFactorLabel, Value: formatTurnMetricMultiplier(contextFirstFive, contextTwentyPlus)},
		{Label: statsClaudeTurnCostEarlyLabel, Value: formatFloat(turnCostFirstFive)},
		{Label: statsClaudeTurnCostLateLabel, Value: formatFloat(turnCostTwentyPlus)},
		{Label: statsClaudeTurnCostFactorLabel, Value: formatTurnMetricMultiplier(turnCostFirstFive, turnCostTwentyPlus)},
	}
}

func averageTurnMetricRange(
	metrics []statspkg.PositionTokenMetrics,
	minPos, maxPos int,
	value func(statspkg.PositionTokenMetrics) float64,
) float64 {
	total := 0.0
	count := 0
	for _, metric := range metrics {
		if metric.Position < minPos || metric.Position > maxPos {
			continue
		}
		total += value(metric)
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func formatTurnMetricMultiplier(early, late float64) string {
	if early <= 0 || late <= 0 {
		return "0x"
	}
	return formatFloat(late/early) + "x"
}

func renderClaudeTurnChart(
	title string,
	metrics []statspkg.PositionTokenMetrics,
	width, height int,
	lineColor color.Color,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	lines := []string{renderStatsTitle(title)}
	if len(metrics) == 0 {
		lines = append(lines, statsNoClaudeTurnMetricsData)
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	// Leave one column of slack so the last X-axis label stays visible
	// when the chart is composed into framed and side-by-side layouts.
	chartWidth := max(width-1, 1)

	minX, maxX := claudeTurnChartRange(metrics)
	maxY := 1.0
	// Keep the true turn positions so sparse samples render
	// with their real horizontal gaps instead of equal spacing.
	points := claudeTurnChartPoints(metrics, value)
	for _, metric := range metrics {
		maxY = max(maxY, value(metric))
	}

	baseChart := linechart.New(
		chartWidth,
		height,
		minX,
		maxX,
		0,
		maxY,
		linechart.WithXYSteps(1, 2),
	)
	baseChart.SetXStep(claudeTurnAxisStep(baseChart.GraphWidth(), 6))
	chart := wlc.New(
		chartWidth,
		height,
		wlc.WithLineChart(&baseChart),
		wlc.WithStyles(
			runes.ArcLineStyle,
			lipgloss.NewStyle().Foreground(lineColor),
		),
		wlc.WithAxesStyles(
			lipgloss.NewStyle().Foreground(colorSecondary),
			lipgloss.NewStyle().Foreground(colorNormalDesc),
		),
	)
	for _, point := range points {
		chart.Plot(point)
	}
	chart.Draw()
	lines = append(lines, strings.Join(splitAndFitLines(chart.View(), width), "\n"))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func claudeTurnChartPoints(
	metrics []statspkg.PositionTokenMetrics,
	value func(statspkg.PositionTokenMetrics) float64,
) []canvas.Float64Point {
	points := make([]canvas.Float64Point, 0, len(metrics))
	for _, metric := range metrics {
		points = append(points, canvas.Float64Point{
			X: float64(metric.Position),
			Y: value(metric),
		})
	}
	return points
}

func claudeTurnChartRange(metrics []statspkg.PositionTokenMetrics) (float64, float64) {
	if len(metrics) == 0 {
		return 0, 1
	}

	minX := float64(metrics[0].Position)
	maxX := float64(metrics[len(metrics)-1].Position)
	return minX, maxX + 1
}

func claudeTurnAxisStep(graphWidth, maxLabels int) int {
	if graphWidth <= 0 || maxLabels <= 1 {
		return 1
	}
	if graphWidth <= maxLabels {
		return 1
	}
	return max((graphWidth-1)/(maxLabels-1), 1)
}
