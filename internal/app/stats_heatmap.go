package app

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const heatmapIntervalCount = 6

type heatmapInterval struct {
	label string
	start int
	end   int
}

var heatmapIntervals = [heatmapIntervalCount]heatmapInterval{
	{label: "00-03", start: 0, end: 3},
	{label: "04-07", start: 4, end: 7},
	{label: "08-11", start: 8, end: 11},
	{label: "12-15", start: 12, end: 15},
	{label: "16-19", start: 16, end: 19},
	{label: "20-23", start: 20, end: 23},
}

func renderActivityHeatmap(title string, cells [7][24]int, width int) string {
	body := renderActivityHeatmapBody(cells, width)
	if body == "" {
		return ""
	}
	return renderStatsTitle(title) + "\n" + body
}

func renderActivityHeatmapBody(cells [7][24]int, width int) string {
	if width <= 0 {
		return ""
	}

	const prefixWidth = 6

	cellWidth := heatmapCellWidth(width)
	lines := make([]string, 0, len(heatmapIntervals)+1)

	headers := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	var headerRow strings.Builder
	headerRow.WriteString(strings.Repeat(" ", prefixWidth))
	for _, header := range headers {
		headerRow.WriteString(lipgloss.PlaceHorizontal(cellWidth, lipgloss.Center, header))
	}
	lines = append(lines, headerRow.String())

	intervalCells := heatmapIntervalCells(cells)
	maxValue := 0
	for day := range 7 {
		for interval := range heatmapIntervalCount {
			maxValue = max(maxValue, intervalCells[day][interval])
		}
	}

	for intervalIndex, interval := range heatmapIntervals {
		var row strings.Builder
		row.WriteString(interval.label)
		row.WriteByte(' ')
		for day := range 7 {
			level := heatmapLevel(intervalCells[day][intervalIndex], maxValue)
			char, style := heatmapCellStyle(level)
			row.WriteString(style.Render(strings.Repeat(char, cellWidth)))
		}
		lines = append(lines, row.String())
	}

	return strings.Join(lines, "\n")
}

func heatmapCellWidth(width int) int {
	const prefixWidth = 6

	return max((width-prefixWidth)/7, 3)
}

func heatmapIntervalCells(cells [7][24]int) [7][heatmapIntervalCount]int {
	var intervals [7][heatmapIntervalCount]int
	for day := range 7 {
		for intervalIndex, interval := range heatmapIntervals {
			total := 0
			for hour := interval.start; hour <= interval.end; hour++ {
				total += cells[day][hour]
			}
			intervals[day][intervalIndex] = total
		}
	}
	return intervals
}

func heatmapLevel(value, maxValue int) int {
	if value <= 0 || maxValue <= 0 {
		return 0
	}

	ratio := float64(value) / float64(maxValue)
	switch {
	case ratio >= 0.75:
		return 4
	case ratio >= 0.5:
		return 3
	case ratio >= 0.25:
		return 2
	default:
		return 1
	}
}

func heatmapCellStyle(level int) (string, lipgloss.Style) {
	switch level {
	case 1:
		return "░", lipgloss.NewStyle().Foreground(colorHeatmap1)
	case 2:
		return "▒", lipgloss.NewStyle().Foreground(colorHeatmap2)
	case 3:
		return "▓", lipgloss.NewStyle().Foreground(colorHeatmap3)
	case 4:
		return "█", lipgloss.NewStyle().Foreground(colorHeatmap4)
	default:
		return " ", lipgloss.NewStyle().Foreground(colorHeatmap0)
	}
}
