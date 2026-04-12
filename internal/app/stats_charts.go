package app

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	statspkg "github.com/rkuska/carn/internal/stats"
)

type barItem struct {
	Label string
	Value int
}

type histBucket struct {
	Label   string
	Count   int
	Display string
}

type tableRow struct {
	Columns []string
}

type chip struct {
	Label string
	Value string
}

func renderHorizontalBars(title string, items []barItem, width int, barColor color.Color) string {
	body := renderHorizontalBarsBody(items, width, barColor)
	if body == "" {
		return ""
	}
	return renderStatsTitle(title) + "\n" + body
}

func renderHorizontalBarsBody(items []barItem, width int, barColor color.Color) string {
	if width <= 0 {
		return ""
	}

	lines := make([]string, 0, len(items)+1)
	if len(items) == 0 {
		lines = append(lines, "No data")
		return strings.Join(lines, "\n")
	}

	labelWidth := 16
	valueWidth := 1
	maxValue := 0
	values := make([]string, len(items))
	for i, item := range items {
		values[i] = statspkg.FormatNumber(item.Value)
		valueWidth = max(valueWidth, lipgloss.Width(values[i]))
		maxValue = max(maxValue, item.Value)
	}
	barWidth := max(width-labelWidth-valueWidth-2, 1)
	barStyle := lipgloss.NewStyle().Foreground(barColor)

	for i, item := range items {
		fillWidth := scaledWidth(item.Value, maxValue, barWidth)
		bar := barStyle.Render(strings.Repeat("█", fillWidth)) +
			strings.Repeat(" ", max(barWidth-fillWidth, 0))
		label := fitToWidth(ansi.Truncate(item.Label, labelWidth, "…"), labelWidth)
		value := fitToWidth(values[i], valueWidth)
		lines = append(lines, ansi.Truncate(label+" "+bar+" "+value, width, "…"))
	}

	return strings.Join(lines, "\n")
}

func renderRankedTable(title string, rows []tableRow, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	width := min(maxWidth, 72)
	titleLine := renderStatsTitle(title)
	if maxWidth > width {
		titleLine = lipgloss.PlaceHorizontal(maxWidth, lipgloss.Center, titleLine)
	}

	body := renderRankedTableBody(rows, maxWidth)
	if body == "" {
		return titleLine
	}
	return titleLine + "\n" + body
}

func renderRankedTableBody(rows []tableRow, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	width := min(maxWidth, 72)
	lines := make([]string, 0, len(rows))
	if len(rows) == 0 {
		lines = append(lines, "No data")
		return strings.Join(lines, "\n")
	}

	colCount := 0
	for _, row := range rows {
		colCount = max(colCount, len(row.Columns))
	}
	if colCount == 0 {
		return strings.Join(lines, "\n")
	}

	colWidths := rankedTableColumnWidths(rows, colCount, width)

	for _, row := range rows {
		line := ansi.Truncate(formatRankedTableRow(row, colWidths), width, "…")
		if maxWidth > width {
			line = lipgloss.PlaceHorizontal(maxWidth, lipgloss.Center, line)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func renderSideBySide(left, right string, width int) string {
	return renderWeightedColumns(left, right, width, 1, 1)
}

func renderWeightedColumns(left, right string, width, leftWeight, rightWeight int) string {
	if width <= 0 {
		return ""
	}

	leftWidth, rightWidth, stacked := statsColumnWidths(width, leftWeight, rightWeight, 30)
	return renderColumns(left, right, leftWidth, rightWidth, stacked)
}

func statsColumnWidths(
	width, leftWeight, rightWeight, minColumnWidth int,
) (int, int, bool) {
	if width <= 0 {
		return 0, 0, true
	}
	if leftWeight <= 0 || rightWeight <= 0 {
		leftWeight, rightWeight = 1, 1
	}
	if minColumnWidth <= 0 {
		minColumnWidth = 1
	}

	available := width - 3
	if available < minColumnWidth*2 {
		return 0, 0, true
	}
	totalWeight := leftWeight + rightWeight
	leftWidth := available * leftWeight / totalWeight
	rightWidth := available - leftWidth
	if leftWidth < minColumnWidth || rightWidth < minColumnWidth {
		return 0, 0, true
	}
	return leftWidth, rightWidth, false
}

func renderColumns(left, right string, leftWidth, rightWidth int, stacked bool) string {
	if stacked || leftWidth <= 0 || rightWidth <= 0 {
		return strings.TrimSpace(left) + "\n\n" + strings.TrimSpace(right)
	}

	leftLines := splitAndFitLines(left, leftWidth)
	rightLines := splitAndFitLines(right, rightWidth)
	lineCount := max(len(leftLines), len(rightLines))

	rows := make([]string, 0, lineCount)
	for i := range lineCount {
		leftLine := ""
		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		rightLine := ""
		if i < len(rightLines) {
			rightLine = rightLines[i]
		}
		rows = append(rows,
			fitToWidth(leftLine, leftWidth)+" "+
				styleRuleHR.Render("│")+" "+
				fitToWidth(rightLine, rightWidth),
		)
	}
	return strings.Join(rows, "\n")
}

func renderStatsTitle(title string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(title)
}

func renderTokenValue(text string) string {
	return lipgloss.NewStyle().Foreground(colorChartToken).Render(text)
}

func renderSummaryChips(chips []chip, width int) string {
	if len(chips) == 0 {
		return ""
	}

	tokens := make([]string, 0, len(chips))
	for _, item := range chips {
		tokens = append(tokens, renderSingleChip(item.Label, item.Value))
	}
	return renderWrappedTokens(tokens, width)
}

func scaledWidth(value, maxValue, width int) int {
	if value <= 0 || maxValue <= 0 || width <= 0 {
		return 0
	}

	scaled := int(math.Round(float64(value) / float64(maxValue) * float64(width)))
	if scaled == 0 {
		return 1
	}
	return min(scaled, width)
}

func splitAndFitLines(content string, width int) []string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		result = append(result, fitToWidth(ansi.Truncate(line, width, "…"), width))
	}
	return result
}

func formatFloat(value float64) string {
	return fmt.Sprintf("%.1f", value)
}

func renderSparkline(points []statspkg.PerformancePoint, width int) string {
	if width <= 0 {
		return ""
	}
	if len(points) == 0 {
		return strings.Repeat("·", width)
	}

	const blocks = "▁▂▃▄▅▆▇█"
	values := make([]float64, 0, len(points))
	minValue := points[0].Value
	maxValue := points[0].Value
	for _, point := range points {
		values = append(values, point.Value)
		if point.Value < minValue {
			minValue = point.Value
		}
		if point.Value > maxValue {
			maxValue = point.Value
		}
	}

	scaled := make([]rune, 0, width)
	for i := range width {
		index := i * len(values) / width
		value := values[index]
		level := 0
		if maxValue > minValue {
			level = int(((value - minValue) / (maxValue - minValue)) * 7)
		}
		scaled = append(scaled, []rune(blocks)[level])
	}
	return string(scaled)
}

func rankedTableColumnWidths(rows []tableRow, colCount, width int) []int {
	colWidths := make([]int, colCount)
	for _, row := range rows {
		for i, col := range row.Columns {
			colWidths[i] = max(colWidths[i], lipgloss.Width(col))
		}
	}

	totalWidth := 2 * (colCount - 1)
	for _, colWidth := range colWidths {
		totalWidth += colWidth
	}
	if totalWidth <= width {
		return colWidths
	}

	shared := max((width-2*(colCount-1))/colCount, 6)
	for i := range colWidths {
		colWidths[i] = min(colWidths[i], shared)
	}
	return colWidths
}

func formatRankedTableRow(row tableRow, colWidths []int) string {
	parts := make([]string, 0, len(colWidths))
	for i := range len(colWidths) {
		col := ""
		if i < len(row.Columns) {
			col = row.Columns[i]
		}
		col = ansi.Truncate(col, colWidths[i], "…")
		parts = append(parts, fitToWidth(col, colWidths[i]))
	}
	return strings.Join(parts, "  ")
}
