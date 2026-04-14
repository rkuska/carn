package app

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

func versionLegendLayout(width int, versions []string, minChartWidth int) (int, int, bool) {
	longestLabel := 0
	for _, version := range versions {
		longestLabel = max(longestLabel, lipgloss.Width(version))
	}
	legendWidth := min(max(longestLabel+4, 18), max(width/4, 18))
	chartWidth := width - legendWidth - 3
	if chartWidth < minChartWidth {
		return width, width, false
	}
	return chartWidth, legendWidth, true
}

func renderChartWithVersionLegend(
	width int,
	versions []string,
	colorByVersion map[string]color.Color,
	minChartWidth int,
	buildChart func(chartWidth int) string,
) string {
	chartWidth, legendWidth, sideLegend := versionLegendLayout(width, versions, minChartWidth)
	chartBody := buildChart(chartWidth)
	if !sideLegend {
		return chartBody + "\n" + renderVersionLegendLabels(versions, width, colorByVersion)
	}
	return renderColumns(
		chartBody,
		renderVersionLegendLabels(versions, legendWidth, colorByVersion),
		chartWidth,
		legendWidth,
		false,
	)
}
