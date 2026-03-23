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
	Label string
	Count int
}

type tableRow struct {
	Columns []string
}

type chip struct {
	Label string
	Value string
}

func renderHorizontalBars(title string, items []barItem, width int, barColor color.Color) string {
	if width <= 0 {
		return ""
	}

	lines := []string{renderStatsTitle(title)}
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

func renderVerticalHistogram(title string, buckets []histBucket, width, maxHeight int) string {
	return renderVerticalHistogramWithColor(title, buckets, width, maxHeight, colorChartBar)
}

func renderVerticalHistogramWithColor(
	title string,
	buckets []histBucket,
	width, maxHeight int,
	barColor color.Color,
) string {
	if width <= 0 {
		return ""
	}

	lines := []string{renderStatsTitle(title)}
	if len(buckets) == 0 {
		lines = append(lines, "No data")
		return strings.Join(lines, "\n")
	}

	maxHeight = max(maxHeight, 1)
	maxCount := histogramMaxCount(buckets)
	axisLabelWidth := max(lipgloss.Width(statspkg.FormatNumber(maxCount)), 1)
	gapWidth := 1
	graphWidth := max(width-axisLabelWidth-3, 1)
	bucketWidth := max((graphWidth-gapWidth*(len(buckets)-1))/len(buckets), 3)
	graphWidth = bucketWidth*len(buckets) + gapWidth*(len(buckets)-1)
	barStyle := lipgloss.NewStyle().Foreground(barColor)

	for level := maxHeight; level >= 1; level-- {
		lines = append(lines, renderHistogramLevel(
			buckets,
			level,
			maxCount,
			maxHeight,
			axisLabelWidth,
			bucketWidth,
			gapWidth,
			width,
			barStyle,
		))
	}

	lines = append(lines, renderHistogramAxis(axisLabelWidth, graphWidth, width))
	lines = append(lines, renderHistogramLabels(buckets, axisLabelWidth, bucketWidth, gapWidth, width))

	return strings.Join(lines, "\n")
}

func histogramMaxCount(buckets []histBucket) int {
	maxCount := 0
	for _, bucket := range buckets {
		maxCount = max(maxCount, bucket.Count)
	}
	if maxCount == 0 {
		return 1
	}
	return maxCount
}

func renderHistogramLevel(
	buckets []histBucket,
	level, maxCount, maxHeight, axisLabelWidth, bucketWidth, gapWidth, width int,
	barStyle lipgloss.Style,
) string {
	parts := make([]string, 0, len(buckets))
	for _, bucket := range buckets {
		parts = append(parts, renderHistogramCell(bucket.Count, maxCount, maxHeight, level, bucketWidth, barStyle))
	}
	prefix := fitToWidth(histogramAxisLabel(histogramLevelLabel(level, maxHeight, maxCount)), axisLabelWidth) +
		" " + histogramAxisLine("│") + " "
	return ansi.Truncate(prefix+strings.Join(parts, strings.Repeat(" ", gapWidth)), width, "…")
}

func renderHistogramCell(
	count, maxCount, maxHeight, level, bucketWidth int,
	barStyle lipgloss.Style,
) string {
	scaled := scaledWidth(count, maxCount, maxHeight)
	if scaled < level {
		return strings.Repeat(" ", bucketWidth)
	}
	return barStyle.Render(strings.Repeat("█", bucketWidth))
}

func histogramLevelLabel(level, maxHeight, maxCount int) string {
	switch level {
	case maxHeight:
		return statspkg.FormatNumber(maxCount)
	case max((maxHeight+1)/2, 1):
		return statspkg.FormatNumber(int(math.Round(float64(maxCount) / 2)))
	default:
		return ""
	}
}

func renderHistogramAxis(axisLabelWidth, graphWidth, width int) string {
	prefix := fitToWidth(histogramAxisLabel("0"), axisLabelWidth) + " " + histogramAxisLine("└")
	return ansi.Truncate(prefix+histogramAxisLine(strings.Repeat("─", graphWidth)), width, "…")
}

func renderHistogramLabels(
	buckets []histBucket,
	axisLabelWidth, bucketWidth, gapWidth, width int,
) string {
	labels := make([]string, 0, len(buckets))
	for _, bucket := range buckets {
		label := ansi.Truncate(bucket.Label, bucketWidth, "…")
		labels = append(labels, lipgloss.PlaceHorizontal(bucketWidth, lipgloss.Center, histogramAxisLabel(label)))
	}
	return ansi.Truncate(
		strings.Repeat(" ", axisLabelWidth+3)+strings.Join(labels, strings.Repeat(" ", gapWidth)),
		width,
		"…",
	)
}

func histogramAxisLabel(text string) string {
	return lipgloss.NewStyle().Foreground(colorNormalDesc).Render(text)
}

func histogramAxisLine(text string) string {
	return lipgloss.NewStyle().Foreground(colorSecondary).Render(text)
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
	lines := []string{titleLine}
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
	if width <= 0 {
		return ""
	}

	halfWidth := (width - 3) / 2
	if halfWidth < 30 {
		return strings.TrimSpace(left) + "\n\n" + strings.TrimSpace(right)
	}

	leftLines := splitAndFitLines(left, halfWidth)
	rightLines := splitAndFitLines(right, halfWidth)
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
			fitToWidth(leftLine, halfWidth)+" "+
				styleRuleHR.Render("│")+" "+
				fitToWidth(rightLine, halfWidth),
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

func centerBlock(content string, outerWidth int) string {
	if outerWidth <= 0 {
		return ""
	}

	lines := strings.Split(content, "\n")
	centered := make([]string, 0, len(lines))
	for _, line := range lines {
		centered = append(centered, lipgloss.PlaceHorizontal(outerWidth, lipgloss.Center, line))
	}
	return strings.Join(centered, "\n")
}

func formatFloat(value float64) string {
	return fmt.Sprintf("%.1f", value)
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
