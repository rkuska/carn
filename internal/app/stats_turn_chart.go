package app

import (
	"image/color"
	"math"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/charmbracelet/x/ansi"

	statspkg "github.com/rkuska/carn/internal/stats"
)

type turnBarColumn struct {
	Start    int
	End      int
	Anchor   int
	Position int
	Height   int
}

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
	if width <= 0 {
		return ""
	}
	if len(metrics) == 0 {
		return statsNoClaudeTurnMetricsData
	}

	maxY := 1.0
	for _, metric := range metrics {
		maxY = max(maxY, value(metric))
	}

	axisLabelWidth := turnBarAxisLabelWidth(maxY)
	graphWidth := max(width-axisLabelWidth-3, 1)
	plotHeight := max(height, 1)
	columns := turnBarColumns(metrics, graphWidth, plotHeight, maxY, value)
	if len(columns) == 0 {
		return statsNoClaudeTurnMetricsData
	}

	barStyle := lipgloss.NewStyle().Foreground(barColor)
	lines := make([]string, 0, plotHeight+3)
	for level := plotHeight; level >= 1; level-- {
		lines = append(lines, renderTurnBarLevel(
			columns,
			level,
			plotHeight,
			maxY,
			axisLabelWidth,
			graphWidth,
			barStyle,
			width,
		))
	}
	lines = append(lines, renderTurnBarAxis(axisLabelWidth, graphWidth, width))
	if showXAxis {
		lines = append(lines, renderTurnBarXAxisRows(columns, axisLabelWidth, graphWidth)...)
	}
	return strings.Join(lines, "\n")
}

func turnBarAxisLabelWidth(maxY float64) int {
	topLabel := statspkg.FormatNumber(int(math.Round(maxY)))
	midLabel := statspkg.FormatNumber(int(math.Round(maxY / 2)))
	return max(lipgloss.Width(topLabel), lipgloss.Width(midLabel), 1)
}

func turnBarColumns(
	metrics []statspkg.PositionTokenMetrics,
	graphWidth, plotHeight int,
	maxY float64,
	value func(statspkg.PositionTokenMetrics) float64,
) []turnBarColumn {
	if len(metrics) == 0 || graphWidth <= 0 || plotHeight <= 0 {
		return nil
	}
	return turnBarEvenColumns(metrics, graphWidth, plotHeight, maxY, value)
}

func turnBarEvenColumns(
	metrics []statspkg.PositionTokenMetrics,
	graphWidth, plotHeight int,
	maxY float64,
	value func(statspkg.PositionTokenMetrics) float64,
) []turnBarColumn {
	layout, ok := resolveUniformTurnBarLayout(graphWidth, len(metrics))
	if !ok {
		return turnBarColumnsFromHistogramLayout(metrics, graphWidth, plotHeight, maxY, value)
	}

	columns := make([]turnBarColumn, 0, len(metrics))
	start := layout.leftPad
	for _, metric := range metrics {
		end := start + layout.barWidth - 1
		columns = append(columns, turnBarColumn{
			Start:    start,
			End:      end,
			Anchor:   start + layout.barWidth/2,
			Position: metric.Position,
			Height:   turnBarScaledHeight(value(metric), maxY, plotHeight),
		})
		start = end + 1 + layout.gapWidth
	}
	return columns
}

func turnBarColumnsFromHistogramLayout(
	metrics []statspkg.PositionTokenMetrics,
	graphWidth, plotHeight int,
	maxY float64,
	value func(statspkg.PositionTokenMetrics) float64,
) []turnBarColumn {
	layout := resolveHistogramLayout(graphWidth, len(metrics))
	columns := make([]turnBarColumn, 0, len(metrics))
	start := 0
	for i, metric := range metrics {
		width := layout.bucketWidths[i]
		if width <= 0 {
			continue
		}
		end := start + width - 1
		columns = append(columns, turnBarColumn{
			Start:    start,
			End:      end,
			Anchor:   start + width/2,
			Position: metric.Position,
			Height:   turnBarScaledHeight(value(metric), maxY, plotHeight),
		})
		start = end + 1 + layout.gapWidth
	}
	return columns
}

func turnBarScaledHeight(current, maxY float64, plotHeight int) int {
	if current <= 0 || maxY <= 0 || plotHeight <= 0 {
		return 0
	}
	scaled := int(math.Round(current / maxY * float64(plotHeight)))
	if scaled == 0 {
		return 1
	}
	return min(scaled, plotHeight)
}

func renderTurnBarLevel(
	columns []turnBarColumn,
	level, plotHeight int,
	maxY float64,
	axisLabelWidth, graphWidth int,
	barStyle lipgloss.Style,
	width int,
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
		if column.Height >= level {
			graph.WriteString(barStyle.Render(strings.Repeat("█", cellWidth)))
		} else {
			graph.WriteString(strings.Repeat(" ", cellWidth))
		}
		cursor = column.End + 1
	}
	if cursor < graphWidth {
		graph.WriteString(strings.Repeat(" ", graphWidth-cursor))
	}
	label := turnBarLevelLabel(level, plotHeight, maxY)
	prefix := fitToWidth(histogramAxisLabel(label), axisLabelWidth) + " " + histogramAxisLine("│") + " "
	return ansi.Truncate(prefix+graph.String(), width, "…")
}

func turnBarLevelLabel(level, plotHeight int, maxY float64) string {
	switch level {
	case plotHeight:
		return statspkg.FormatNumber(int(math.Round(maxY)))
	case max((plotHeight+1)/2, 1):
		return statspkg.FormatNumber(int(math.Round(maxY / 2)))
	default:
		return ""
	}
}

func renderTurnBarAxis(axisLabelWidth, graphWidth, width int) string {
	prefix := fitToWidth(histogramAxisLabel("0"), axisLabelWidth) + " " + histogramAxisLine("└")
	return ansi.Truncate(prefix+histogramAxisLine(strings.Repeat("─", graphWidth)), width, "…")
}

func renderTurnBarXAxisRows(
	columns []turnBarColumn,
	axisLabelWidth, graphWidth int,
) []string {
	placements := make([]claudeTurnAxisLabelPlacement, 0, len(columns))
	for _, column := range columns {
		placements = append(placements, claudeTurnAxisLabelPlacement{
			Anchor: column.Anchor,
			Label:  strconv.Itoa(column.Position),
		})
	}
	rows := claudeTurnAxisLabelGrid(graphWidth, placements)
	return renderClaudeTurnAxisRows(strings.Repeat(" ", axisLabelWidth+3), rows)
}

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

type claudeTurnAxisLabelPlacement struct {
	Anchor int
	Label  string
}

func claudeTurnAxisLabelGrid(
	graphWidth int,
	placements []claudeTurnAxisLabelPlacement,
) [][]rune {
	rows := make([][]rune, 0, 2)
	for _, placement := range placements {
		labelRunes := []rune(placement.Label)
		start := placement.Anchor - len(labelRunes)/2
		start = max(start, 0)
		if start+len(labelRunes) > graphWidth {
			start = max(graphWidth-len(labelRunes), 0)
		}

		rowIndex := 0
		for {
			if rowIndex == len(rows) {
				rows = append(rows, []rune(strings.Repeat(" ", graphWidth)))
			}
			if claudeTurnAxisLabelFits(rows[rowIndex], start, labelRunes) {
				copy(rows[rowIndex][start:start+len(labelRunes)], labelRunes)
				break
			}
			rowIndex++
		}
	}
	return rows
}

func claudeTurnAxisLabelFits(row []rune, start int, label []rune) bool {
	if start < 0 || start+len(label) > len(row) {
		return false
	}
	if start > 0 && row[start-1] != ' ' {
		return false
	}
	if end := start + len(label); end < len(row) && row[end] != ' ' {
		return false
	}
	for i := range label {
		if row[start+i] != ' ' {
			return false
		}
	}
	return true
}

func renderClaudeTurnAxisRows(prefix string, rows [][]rune) []string {
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, prefix+string(row))
	}
	return lines
}
