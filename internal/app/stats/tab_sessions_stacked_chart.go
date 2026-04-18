package stats

import (
	"image/color"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	el "github.com/rkuska/carn/internal/app/elements"
	statspkg "github.com/rkuska/carn/internal/stats"
)

type stackedTurnBarSegment struct {
	Version string
	Value   float64
	Height  int
	Color   color.Color
}

type stackedTurnBarColumn struct {
	turnBarColumn
	Segments []stackedTurnBarSegment
}

func buildStackedTurnBars(
	series []statspkg.VersionTurnSeries,
	plotHeight int,
	colorByVersion map[string]color.Color,
	value func(statspkg.PositionTokenMetrics) float64,
) []stackedTurnBarColumn {
	positions := collectTurnPositions(series)
	if len(positions) == 0 {
		return nil
	}

	barTotals := stackedTurnBarTotals(series, value)
	maxTotal := 0.0
	for _, position := range positions {
		maxTotal = max(maxTotal, barTotals[position])
	}
	if maxTotal <= 0 {
		return nil
	}

	columns := make([]stackedTurnBarColumn, 0, len(positions))
	for _, position := range positions {
		segments := stackedTurnBarSegmentsForPosition(series, position, colorByVersion, value)
		if len(segments) == 0 {
			continue
		}
		totalHeight := turnBarScaledHeight(barTotals[position], maxTotal, max(plotHeight, 1))
		assignStackedTurnSegmentHeights(segments, totalHeight)
		columns = append(columns, stackedTurnBarColumn{
			turnBarColumn: turnBarColumn{
				Position: position,
				Height:   totalHeight,
			},
			Segments: segments,
		})
	}
	return columns
}

func collectTurnPositions(series []statspkg.VersionTurnSeries) []int {
	positionSet := make(map[int]bool)
	for _, item := range series {
		for _, metric := range item.Metrics {
			positionSet[metric.Position] = true
		}
	}

	positions := make([]int, 0, len(positionSet))
	for position := range positionSet {
		positions = append(positions, position)
	}
	slices.Sort(positions)
	return positions
}

func stackedTurnBarTotals(
	series []statspkg.VersionTurnSeries,
	value func(statspkg.PositionTokenMetrics) float64,
) map[int]float64 {
	totals := make(map[int]float64)
	for _, item := range series {
		for _, metric := range item.Metrics {
			totals[metric.Position] += value(metric)
		}
	}
	return totals
}

func stackedTurnBarSegmentsForPosition(
	series []statspkg.VersionTurnSeries,
	position int,
	colorByVersion map[string]color.Color,
	value func(statspkg.PositionTokenMetrics) float64,
) []stackedTurnBarSegment {
	segments := make([]stackedTurnBarSegment, 0, len(series))
	for _, item := range series {
		metricValue, ok := stackedTurnMetricValue(item.Metrics, position, value)
		if !ok || metricValue <= 0 {
			continue
		}
		segments = append(segments, stackedTurnBarSegment{
			Version: item.Version,
			Value:   metricValue,
			Color:   colorByVersion[item.Version],
		})
	}
	return segments
}

func stackedTurnMetricValue(
	metrics []statspkg.PositionTokenMetrics,
	position int,
	value func(statspkg.PositionTokenMetrics) float64,
) (float64, bool) {
	for _, metric := range metrics {
		if metric.Position == position {
			return value(metric), true
		}
	}
	return 0, false
}

func assignStackedTurnSegmentHeights(segments []stackedTurnBarSegment, totalHeight int) {
	values := make([]float64, 0, len(segments))
	for _, segment := range segments {
		values = append(values, segment.Value)
	}
	heights := resolveFloatSegmentHeights(totalHeight, values)
	for i := range segments {
		segments[i].Height = heights[i]
	}
}

func renderStackedTurnBarsChartBody(theme *el.Theme, columns []stackedTurnBarColumn, width int) string {
	if len(columns) == 0 || width <= 0 {
		return statsNoClaudeTurnMetricsData
	}

	maxTotal := 0.0
	plotHeight := 1
	for _, column := range columns {
		total := 0.0
		for _, segment := range column.Segments {
			total += segment.Value
		}
		maxTotal = max(maxTotal, total)
		plotHeight = max(plotHeight, column.Height)
	}

	axisLabelWidth := turnBarAxisLabelWidth(maxTotal)
	graphWidth := max(width-axisLabelWidth-3, len(columns))
	renderColumns := layoutStackedTurnColumns(columns, graphWidth)

	lines := make([]string, 0, plotHeight+3)
	for level := plotHeight; level >= 1; level-- {
		lines = append(lines, renderStackedTurnBarLevel(
			theme,
			renderColumns,
			level,
			plotHeight,
			maxTotal,
			axisLabelWidth,
			graphWidth,
			width,
		))
	}
	lines = append(lines, renderTurnBarAxis(theme, axisLabelWidth, graphWidth, width))

	axisColumns := make([]turnBarColumn, 0, len(renderColumns))
	for _, column := range renderColumns {
		axisColumns = append(axisColumns, column.turnBarColumn)
	}
	lines = append(lines, renderTurnBarXAxisRows(axisColumns, axisLabelWidth, graphWidth)...)
	return strings.Join(lines, "\n")
}

func layoutStackedTurnColumns(
	columns []stackedTurnBarColumn,
	graphWidth int,
) []stackedTurnBarColumn {
	if len(columns) == 0 {
		return nil
	}

	layout, ok := resolveUniformTurnBarLayout(max(graphWidth, len(columns)), len(columns))
	if !ok {
		return layoutStackedTurnColumnsFromHistogram(columns, graphWidth)
	}

	resized := make([]stackedTurnBarColumn, 0, len(columns))
	start := layout.leftPad
	for _, column := range columns {
		column.Start = start
		column.End = start + layout.barWidth - 1
		column.Anchor = start + layout.barWidth/2
		resized = append(resized, column)
		start = column.End + 1 + layout.gapWidth
	}
	return resized
}

func layoutStackedTurnColumnsFromHistogram(
	columns []stackedTurnBarColumn,
	graphWidth int,
) []stackedTurnBarColumn {
	layout := resolveHistogramLayout(max(graphWidth, len(columns)), len(columns))
	resized := make([]stackedTurnBarColumn, 0, len(columns))
	start := 0
	for i, column := range columns {
		columnWidth := layout.bucketWidths[i]
		if columnWidth <= 0 {
			continue
		}
		column.Start = start
		column.End = start + columnWidth - 1
		column.Anchor = start + columnWidth/2
		resized = append(resized, column)
		start = column.End + 1 + layout.gapWidth
	}
	return resized
}

func renderStackedTurnBarLevel(
	theme *el.Theme,
	columns []stackedTurnBarColumn,
	level, plotHeight int,
	maxTotal float64,
	axisLabelWidth, graphWidth, width int,
) string {
	var graph strings.Builder
	cursor := 0
	for _, column := range columns {
		if column.Start > cursor {
			graph.WriteString(strings.Repeat(" ", column.Start-cursor))
			cursor = column.Start
		}

		cellWidth := max(column.End-column.Start+1, 0)
		if cellWidth == 0 {
			continue
		}

		segment, ok := stackedTurnSegmentAtLevel(column, level)
		if ok {
			graph.WriteString(
				lipgloss.NewStyle().Foreground(segment.Color).Render(strings.Repeat("█", cellWidth)),
			)
		} else {
			graph.WriteString(strings.Repeat(" ", cellWidth))
		}
		cursor = column.End + 1
	}
	if cursor < graphWidth {
		graph.WriteString(strings.Repeat(" ", graphWidth-cursor))
	}

	label := turnBarLevelLabel(level, plotHeight, maxTotal)
	prefix := fitToWidth(histogramAxisLabel(theme, label), axisLabelWidth) + " " + histogramAxisLine(theme, "│") + " "
	return ansi.Truncate(prefix+graph.String(), width, "…")
}

func stackedTurnSegmentAtLevel(
	column stackedTurnBarColumn,
	level int,
) (stackedTurnBarSegment, bool) {
	if column.Height < level {
		return stackedTurnBarSegment{}, false
	}

	accumulated := 0
	for _, segment := range column.Segments {
		accumulated += segment.Height
		if level <= accumulated {
			return segment, true
		}
	}
	return stackedTurnBarSegment{}, false
}
