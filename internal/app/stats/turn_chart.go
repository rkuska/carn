package stats

import (
	"image/color"

	el "github.com/rkuska/carn/internal/app/elements"
	statspkg "github.com/rkuska/carn/internal/stats"
)

type turnBarColumn = el.TurnBarColumn

func claudeTurnMetricChips(metrics []statspkg.PositionTokenMetrics) []chip {
	if len(metrics) == 0 {
		return []chip{{Label: statsClaudeMetricsNoDataLabel, Value: noDataLabel}}
	}

	contextFirstFive := averageTurnMetricRange(metrics, 1, 5, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AveragePromptTokens
	})
	contextTwentyPlus := averageTurnMetricRange(metrics, 20, 999, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AveragePromptTokens
	})
	turnCostFirstFive := averageTurnMetricRange(metrics, 1, 5, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageTurnTokens
	})
	turnCostTwentyPlus := averageTurnMetricRange(metrics, 20, 999, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageTurnTokens
	})
	return []chip{
		{Label: statsClaudePromptEarlyLabel, Value: formatFloat(contextFirstFive)},
		{Label: statsClaudePromptLateLabel, Value: formatFloat(contextTwentyPlus)},
		{Label: statsClaudePromptFactorLabel, Value: formatTurnMetricMultiplier(contextFirstFive, contextTwentyPlus)},
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
	barColor color.Color,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	body := renderClaudeTurnChartBody(metrics, width, height, barColor, value)
	if body == "" {
		return ""
	}
	return renderStatsTitle(title) + "\n" + body
}

func renderClaudeTurnChartBody(
	metrics []statspkg.PositionTokenMetrics,
	width, height int,
	barColor color.Color,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	return renderTurnBarChartBody(metrics, width, height, barColor, value, true)
}

func renderTurnBarChartBody(
	metrics []statspkg.PositionTokenMetrics,
	width, height int,
	barColor color.Color,
	value func(statspkg.PositionTokenMetrics) float64,
	showXAxis bool,
) string {
	return el.RenderTurnBarChartBody(
		metrics,
		width,
		height,
		barColor,
		value,
		showXAxis,
		statsNoClaudeTurnMetricsData,
	)
}

var (
	turnBarAxisLabelWidth  = el.TurnBarAxisLabelWidth
	turnBarColumns         = el.TurnBarColumns
	turnBarScaledHeight    = el.TurnBarScaledHeight
	turnBarLevelLabel      = el.TurnBarLevelLabel
	renderTurnBarAxis      = el.RenderTurnBarAxis
	renderTurnBarXAxisRows = el.RenderTurnBarXAxisRows
	claudeTurnChartPoints  = el.ClaudeTurnChartPoints
	claudeTurnChartRange   = el.ClaudeTurnChartRange
)

func (m statsModel) renderClaudeTurnMetricLaneBody(
	width, height int,
	metrics []statspkg.PositionTokenMetrics,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	if len(metrics) == 0 {
		return statsNoClaudeTurnMetricsData
	}
	return renderClaudeTurnChartBody(metrics, width, height, colorChartToken, value)
}
