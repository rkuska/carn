package elements

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

type TurnBarColumn struct {
	Start         int
	End           int
	Anchor        int
	Position      int
	StartPosition int
	EndPosition   int
	Height        int
}

func (t *Theme) RenderTurnBarChartBody(
	metrics []statspkg.PositionTokenMetrics,
	width, height int,
	barColor color.Color,
	value func(statspkg.PositionTokenMetrics) float64,
	showXAxis bool,
	emptyState string,
) string {
	if width <= 0 {
		return ""
	}
	if len(metrics) == 0 {
		return emptyState
	}

	maxY := 1.0
	for _, metric := range metrics {
		maxY = max(maxY, value(metric))
	}

	axisLabelWidth := TurnBarAxisLabelWidth(maxY)
	graphWidth := max(width-axisLabelWidth-3, 1)
	plotHeight := max(height, 1)
	columns := TurnBarColumns(metrics, graphWidth, plotHeight, maxY, value)
	if len(columns) == 0 {
		return emptyState
	}

	barStyle := lipgloss.NewStyle().Foreground(barColor)
	lines := make([]string, 0, plotHeight+3)
	for level := plotHeight; level >= 1; level-- {
		lines = append(lines, t.renderTurnBarLevel(
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
	lines = append(lines, t.RenderTurnBarAxis(axisLabelWidth, graphWidth, width))
	if showXAxis {
		lines = append(lines, RenderTurnBarXAxisRows(columns, axisLabelWidth, graphWidth)...)
	}
	return strings.Join(lines, "\n")
}

func TurnBarAxisLabelWidth(maxY float64) int {
	topLabel := statspkg.FormatNumber(int(math.Round(maxY)))
	midLabel := statspkg.FormatNumber(int(math.Round(maxY / 2)))
	return max(lipgloss.Width(topLabel), lipgloss.Width(midLabel), 1)
}

func TurnBarColumns(
	metrics []statspkg.PositionTokenMetrics,
	graphWidth, plotHeight int,
	maxY float64,
	value func(statspkg.PositionTokenMetrics) float64,
) []TurnBarColumn {
	if len(metrics) == 0 || graphWidth <= 0 || plotHeight <= 0 {
		return nil
	}

	bucketCount := turnMetricBucketCount(metrics, graphWidth)
	buckets := bucketTurnMetrics(metrics, bucketCount, value)
	slots := resolveVerticalBarGroupSlots(turnMetricActiveCounts(len(buckets)), graphWidth)
	columns := make([]TurnBarColumn, 0, min(len(buckets), len(slots)))
	for i := range min(len(buckets), len(slots)) {
		slot := slots[i]
		if len(slot.Bars) == 0 {
			continue
		}
		bar := slot.Bars[0]
		columns = append(columns, TurnBarColumn{
			Start:         bar.Start,
			End:           bar.End - 1,
			Anchor:        bar.Anchor,
			Position:      buckets[i].StartPosition,
			StartPosition: buckets[i].StartPosition,
			EndPosition:   buckets[i].EndPosition,
			Height:        TurnBarScaledHeight(buckets[i].Value, maxY, plotHeight),
		})
	}
	return columns
}

func TurnBarScaledHeight(current, maxY float64, plotHeight int) int {
	if current <= 0 || maxY <= 0 || plotHeight <= 0 {
		return 0
	}
	scaled := int(math.Round(current / maxY * float64(plotHeight)))
	if scaled == 0 {
		return 1
	}
	return min(scaled, plotHeight)
}

func (t *Theme) renderTurnBarLevel(
	columns []TurnBarColumn,
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
	label := TurnBarLevelLabel(level, plotHeight, maxY)
	prefix := FitToWidth(t.HistogramAxisLabel(label), axisLabelWidth) + " " + t.HistogramAxisLine("│") + " "
	return ansi.Truncate(prefix+graph.String(), width, "…")
}

func TurnBarLevelLabel(level, plotHeight int, maxY float64) string {
	switch level {
	case plotHeight:
		return statspkg.FormatNumber(int(math.Round(maxY)))
	case max((plotHeight+1)/2, 1):
		return statspkg.FormatNumber(int(math.Round(maxY / 2)))
	default:
		return ""
	}
}

func (t *Theme) RenderTurnBarAxis(axisLabelWidth, graphWidth, width int) string {
	prefix := FitToWidth(t.HistogramAxisLabel("0"), axisLabelWidth) + " " + t.HistogramAxisLine("└")
	return ansi.Truncate(prefix+t.HistogramAxisLine(strings.Repeat("─", graphWidth)), width, "…")
}

func RenderTurnBarXAxisRows(
	columns []TurnBarColumn,
	axisLabelWidth, graphWidth int,
) []string {
	placements := make([]claudeTurnAxisLabelPlacement, 0, len(columns))
	for _, column := range columns {
		placements = append(placements, claudeTurnAxisLabelPlacement{
			Anchor: column.Anchor,
			Label:  turnBarColumnLabel(column),
		})
	}
	rows := claudeTurnAxisLabelGrid(graphWidth, placements)
	return renderClaudeTurnAxisRows(strings.Repeat(" ", axisLabelWidth+3), rows)
}

type turnMetricBucket struct {
	StartPosition int
	EndPosition   int
	Value         float64
}

func turnMetricBucketCount(
	metrics []statspkg.PositionTokenMetrics,
	graphWidth int,
) int {
	if len(metrics) == 0 || graphWidth <= 0 {
		return 0
	}
	return groupedVerticalBarBucketCount(len(metrics), graphWidth, turnMetricActiveCounts)
}

func turnMetricActiveCounts(bucketCount int) []int {
	counts := make([]int, bucketCount)
	for i := range counts {
		counts[i] = 1
	}
	return counts
}

func bucketTurnMetrics(
	metrics []statspkg.PositionTokenMetrics,
	bucketCount int,
	value func(statspkg.PositionTokenMetrics) float64,
) []turnMetricBucket {
	if len(metrics) == 0 || bucketCount <= 0 {
		return nil
	}

	buckets := make([]turnMetricBucket, 0, bucketCount)
	for i := range bucketCount {
		start := i * len(metrics) / bucketCount
		end := (i + 1) * len(metrics) / bucketCount
		if end <= start {
			end = start + 1
		}

		bucket := turnMetricBucket{
			StartPosition: metrics[start].Position,
			EndPosition:   metrics[end-1].Position,
		}
		for _, metric := range metrics[start:end] {
			bucket.Value += value(metric)
		}
		bucket.Value /= float64(end - start)
		buckets = append(buckets, bucket)
	}
	return buckets
}

func turnBarColumnLabel(column TurnBarColumn) string {
	start := max(column.StartPosition, column.Position)
	end := max(column.EndPosition, start)
	if start == end {
		return strconv.Itoa(start)
	}
	return strconv.Itoa(start) + "-" + strconv.Itoa(end)
}

func ClaudeTurnChartPoints(
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

func ClaudeTurnChartRange(metrics []statspkg.PositionTokenMetrics) (float64, float64) {
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
		if start+len(labelRunes) == graphWidth && graphWidth > len(labelRunes) {
			start--
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
