package stats

import (
	"image/color"

	el "github.com/rkuska/carn/internal/app/elements"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderSplitTurnMetricLaneBody(
	width, height int,
	mode statspkg.StatisticMode,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	if !m.splitBy.SupportsTurnMetrics() {
		return splitTurnMetricsUnsupportedMessage(m.splitBy)
	}

	series := m.splitTurnSeries(mode)
	if len(series) == 0 {
		return statsNoClaudeTurnMetricsData
	}
	return renderSplitTurnChartBody(m.theme, series, width, height, m.splitColors, mode, value)
}

func splitTurnMetricsUnsupportedMessage(dim statspkg.SplitDimension) string {
	return splitMetricUnavailableMessage(dim, "turn metrics")
}

func renderSplitTurnChartBody(
	theme *el.Theme,
	series []statspkg.SplitTurnSeries,
	width, height int,
	colorByKey map[string]color.Color,
	mode statspkg.StatisticMode,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	if len(series) == 0 {
		return statsNoClaudeTurnMetricsData
	}

	return renderChartWithSplitLegend(
		theme,
		width,
		el.SplitTurnSeriesKeys(series),
		colorByKey,
		splitTurnChartMinWidth,
		func(chartWidth int) string {
			return theme.RenderSplitTurnGroupedChartBody(
				series,
				chartWidth,
				height,
				colorByKey,
				mode,
				value,
				statsNoClaudeTurnMetricsData,
			)
		},
	)
}

func splitMetricUnavailableMessage(dim statspkg.SplitDimension, metric string) string {
	return "Split by " + dim.Label() + " is not available for " + metric + "."
}
