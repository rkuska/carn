package app

import (
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type stackedRowSegment struct {
	Value int
	Color color.Color
}

type stackedRowItem struct {
	Label    string
	Scale    float64
	Value    string
	Segments []stackedRowSegment
}

func renderHorizontalStackedBarsBody(items []stackedRowItem, width int) string {
	if width <= 0 {
		return ""
	}
	if len(items) == 0 {
		return "No data"
	}

	labelWidth := 16
	valueWidth := 1
	maxScale := 0.0
	for _, item := range items {
		labelWidth = max(labelWidth, lipgloss.Width(item.Label))
		valueWidth = max(valueWidth, lipgloss.Width(item.Value))
		maxScale = max(maxScale, item.Scale)
	}
	labelWidth = min(labelWidth, max(width/2, 12))
	barWidth := max(width-labelWidth-valueWidth-2, 1)

	lines := make([]string, 0, len(items))
	for _, item := range items {
		label := fitToWidth(ansi.Truncate(item.Label, labelWidth, "…"), labelWidth)
		value := fitToWidth(item.Value, valueWidth)
		fillWidth := scaledFloatWidth(item.Scale, maxScale, barWidth)
		line := label + " " + renderHorizontalStackedBar(item.Segments, fillWidth, barWidth) + " " + value
		lines = append(lines, ansi.Truncate(line, width, "…"))
	}
	return strings.Join(lines, "\n")
}

func renderHorizontalStackedBar(
	segments []stackedRowSegment,
	fillWidth int,
	barWidth int,
) string {
	if barWidth <= 0 {
		return ""
	}
	if fillWidth <= 0 || len(segments) == 0 {
		return strings.Repeat(" ", barWidth)
	}
	values := make([]int, 0, len(segments))
	for _, segment := range segments {
		values = append(values, max(segment.Value, 0))
	}
	segmentWidths := resolveStackedBarWidths(fillWidth, values)
	var bar strings.Builder
	for i, segment := range segments {
		if segmentWidths[i] <= 0 {
			continue
		}
		bar.WriteString(
			lipgloss.NewStyle().Foreground(segment.Color).Render(strings.Repeat("█", segmentWidths[i])),
		)
	}
	if remainder := barWidth - lipgloss.Width(bar.String()); remainder > 0 {
		bar.WriteString(strings.Repeat(" ", remainder))
	}
	return bar.String()
}

func scaledFloatWidth(value, maxValue float64, width int) int {
	if value <= 0 || maxValue <= 0 || width <= 0 {
		return 0
	}
	scaled := int(math.Round(value / maxValue * float64(width)))
	if scaled == 0 {
		return 1
	}
	return min(scaled, width)
}
