package stats

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	el "github.com/rkuska/carn/internal/app/elements"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderVersionedTurnMetricLaneBody(
	width, height int,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	if !m.groupScope.hasProvider() {
		return selectProviderWithVersionsPrompt
	}

	series := m.groupedTurnSeries()
	if len(series) == 0 {
		return "No turn metrics for the selected provider/version scope."
	}
	return renderVersionedTurnChartBody(m.theme, series, width, height, m.groupScopeColorMap(), value)
}

func renderVersionedTurnChartBody(
	theme *el.Theme,
	series []statspkg.VersionTurnSeries,
	width, height int,
	colorByVersion map[string]color.Color,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	if len(series) == 0 {
		return statsNoClaudeTurnMetricsData
	}

	chartWidth, legendWidth, sideLegend := versionedTurnChartWidths(width, series)
	columns := buildStackedTurnBars(series, height, colorByVersion, value)
	chartBody := renderStackedTurnBarsChartBody(theme, columns, chartWidth)
	legendBody := renderVersionLegend(series, legendWidth, colorByVersion)
	if !sideLegend {
		return chartBody + "\n" + legendBody
	}
	return renderColumns(theme, chartBody, legendBody, chartWidth, legendWidth, false)
}

func versionedTurnChartWidths(
	width int,
	series []statspkg.VersionTurnSeries,
) (chartWidth int, legendWidth int, sideLegend bool) {
	longestLabel := 0
	for _, item := range series {
		longestLabel = max(longestLabel, lipgloss.Width(item.Version))
	}

	legendWidth = min(max(longestLabel+4, 18), max(width/4, 18))
	chartWidth = width - legendWidth - 3
	if chartWidth < 40 {
		return width, width, false
	}
	return chartWidth, legendWidth, true
}

func renderVersionLegend(
	series []statspkg.VersionTurnSeries,
	width int,
	colorByVersion map[string]color.Color,
) string {
	versions := make([]string, 0, len(series))
	for _, item := range series {
		versions = append(versions, item.Version)
	}
	return renderVersionLegendLabels(versions, width, colorByVersion)
}

func renderVersionLegendLabels(
	versions []string,
	width int,
	colorByVersion map[string]color.Color,
) string {
	lines := make([]string, 0, len(versions))
	for _, version := range versions {
		line := lipgloss.NewStyle().Foreground(colorByVersion[version]).Render("██") +
			" " + version
		lines = append(lines, fitToWidth(line, width))
	}
	return strings.Join(lines, "\n")
}
