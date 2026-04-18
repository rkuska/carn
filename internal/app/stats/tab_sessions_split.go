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
	return renderSplitTurnChartBody(m.theme, series, width, height, m.splitColors, value)
}

func splitTurnMetricsUnsupportedMessage(dim statspkg.SplitDimension) string {
	return "Split by " + dim.Label() + " is not available for turn metrics."
}

func renderSplitTurnChartBody(
	theme *el.Theme,
	series []statspkg.SplitTurnSeries,
	width, height int,
	colorByKey map[string]color.Color,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	if len(series) == 0 {
		return statsNoClaudeTurnMetricsData
	}

	keys := make([]string, 0, len(series))
	for _, item := range series {
		keys = append(keys, item.Key)
	}
	chartWidth, legendWidth, sideLegend := splitLegendLayout(width, keys, splitTurnChartMinWidth)
	columns := buildStackedTurnBars(series, height, colorByKey, value)
	chartBody := renderStackedTurnBarsChartBody(theme, columns, chartWidth)
	legendBody := renderSplitLegendForSeries(series, legendWidth, colorByKey)
	if !sideLegend {
		return chartBody + "\n" + legendBody
	}
	return renderColumns(theme, chartBody, legendBody, chartWidth, legendWidth, false)
}
