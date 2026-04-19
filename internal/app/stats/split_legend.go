package stats

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	el "github.com/rkuska/carn/internal/app/elements"
)

const (
	splitChartMinWidth     = 24
	splitTurnChartMinWidth = 40
)

func splitLegendLayout(width int, keys []string, minChartWidth int) (int, int, bool) {
	longestLabel := 0
	for _, key := range keys {
		longestLabel = max(longestLabel, lipgloss.Width(key))
	}
	legendWidth := min(max(longestLabel+4, 18), max(width/4, 18))
	chartWidth := width - legendWidth - 3
	if chartWidth < minChartWidth {
		return width, width, false
	}
	return chartWidth, legendWidth, true
}

func renderChartWithSplitLegend(
	theme *el.Theme,
	width int,
	keys []string,
	colorByKey map[string]color.Color,
	minChartWidth int,
	buildChart func(chartWidth int) string,
) string {
	if len(keys) == 0 {
		return buildChart(width)
	}
	chartWidth, legendWidth, sideLegend := splitLegendLayout(width, keys, minChartWidth)
	chartBody := buildChart(chartWidth)
	if !sideLegend {
		return chartBody + "\n" + renderSplitLegendLabels(keys, width, colorByKey)
	}
	return renderColumns(
		theme,
		chartBody,
		renderSplitLegendLabels(keys, legendWidth, colorByKey),
		chartWidth,
		legendWidth,
		false,
	)
}

func renderSplitLegendLabels(
	keys []string,
	width int,
	colorByKey map[string]color.Color,
) string {
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		line := lipgloss.NewStyle().Foreground(colorByKey[key]).Render("██") +
			" " + key
		lines = append(lines, fitToWidth(line, width))
	}
	return strings.Join(lines, "\n")
}
