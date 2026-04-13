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

	histograms := renderStatsLanePair(
		width,
		30,
		"Session Duration",
		m.sessionsLaneCursor == 0,
		func(bodyWidth int) string {
			return renderVerticalHistogramBody(durationBuckets, bodyWidth, 8, colorChartTime)
		},
		"Messages per Session",
		m.sessionsLaneCursor == 1,
		func(bodyWidth int) string {
			return renderVerticalHistogramBody(messageBuckets, bodyWidth, 8, colorChartBar)
		},
	)

	growthChips := renderSummaryChips(claudeTurnMetricChips(sessionStats.ClaudeTurnMetrics), width)
	growthCharts := renderStatsLanePair(
		width,
		30,
		statsClaudeContextGrowthTitle,
		m.sessionsLaneCursor == 2,
		func(bodyWidth int) string {
			return m.renderClaudeTurnMetricLaneBody(
				bodyWidth,
				10,
				sessionStats.ClaudeTurnMetrics,
				func(metric statspkg.PositionTokenMetrics) float64 {
					return metric.AverageInputTokens
				},
			)
		},
		statsClaudeTurnCostTitle,
		m.sessionsLaneCursor == 3,
		func(bodyWidth int) string {
			return m.renderClaudeTurnMetricLaneBody(
				bodyWidth,
				10,
				sessionStats.ClaudeTurnMetrics,
				func(metric statspkg.PositionTokenMetrics) float64 {
					return metric.AverageTurnTokens
				},
			)
		},
	)
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n\n%s",
		chips,
		histograms,
		growthChips,
		growthCharts,
		m.renderActiveMetricDetail(width),
	)
}

func claudeTurnMetricChips(metrics []statspkg.PositionTokenMetrics) []chip {
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
	body := renderClaudeTurnChartBody(metrics, width, height, lineColor, value)
	if body == "" {
		return ""
	}
	return renderStatsTitle(title) + "\n" + body
}

func renderClaudeTurnChartBody(
	metrics []statspkg.PositionTokenMetrics,
	width, height int,
	lineColor color.Color,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	lines := make([]string, 0, 2)
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

func (m statsModel) renderClaudeTurnMetricLaneBody(
	width, height int,
	metrics []statspkg.PositionTokenMetrics,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	if len(metrics) == 0 {
		return statsNoClaudeTurnMetricsData
	}
	return renderClaudeTurnChartBody(
		metrics,
		width,
		height,
		colorChartToken,
		value,
	)
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
