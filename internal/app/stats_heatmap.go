package app

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

func renderActivityHeatmap(title string, cells [7][24]int, width int) string {
	if width <= 0 {
		return ""
	}

	const prefixWidth = 3

	cellWidth := max((width-prefixWidth)/7, 3)
	lines := []string{renderStatsTitle(title)}

	headers := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	var headerRow strings.Builder
	headerRow.WriteString(strings.Repeat(" ", prefixWidth))
	for _, header := range headers {
		headerRow.WriteString(lipgloss.PlaceHorizontal(cellWidth, lipgloss.Center, header))
	}
	lines = append(lines, headerRow.String())

	maxValue := 0
	for day := range 7 {
		for hour := range 24 {
			maxValue = max(maxValue, cells[day][hour])
		}
	}

	for hour := range 24 {
		var row strings.Builder
		_, _ = fmt.Fprintf(&row, "%02d ", hour)
		for day := range 7 {
			level := heatmapLevel(cells[day][hour], maxValue)
			char, color := heatmapCellStyle(level)
			cell := strings.Repeat(char, cellWidth)
			row.WriteString(lipgloss.NewStyle().Foreground(color).Render(cell))
		}
		lines = append(lines, row.String())
	}

	return strings.Join(lines, "\n")
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

func heatmapCellStyle(level int) (string, color.Color) {
	switch level {
	case 1:
		return "░", colorHeatmap1
	case 2:
		return "▒", colorHeatmap2
	case 3:
		return "▓", colorHeatmap3
	case 4:
		return "█", colorHeatmap4
	default:
		return " ", colorHeatmap0
	}
}
