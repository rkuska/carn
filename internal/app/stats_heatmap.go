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

	for _, hour := range heatmapDisplayRows(cells) {
		if hour < 0 {
			lines = append(lines, lipgloss.NewStyle().Foreground(colorSecondary).Render("···"))
			continue
		}

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

func heatmapDisplayRows(cells [7][24]int) []int {
	activeHours := heatmapActiveHours(cells)
	if len(activeHours) == 0 {
		rows := make([]int, 0, 24)
		for hour := range 24 {
			rows = append(rows, hour)
		}
		return rows
	}

	rows := make([]int, 0, len(activeHours)+2)
	if activeHours[0] > 0 {
		rows = append(rows, -1)
	}
	rows = append(rows, activeHours[0])
	for i := 1; i < len(activeHours); i++ {
		if activeHours[i]-activeHours[i-1] > 1 {
			rows = append(rows, -1)
		}
		rows = append(rows, activeHours[i])
	}
	if activeHours[len(activeHours)-1] < 23 {
		rows = append(rows, -1)
	}
	return rows
}

func heatmapActiveHours(cells [7][24]int) []int {
	hours := make([]int, 0, 24)
	for hour := range 24 {
		for day := range 7 {
			if cells[day][hour] > 0 {
				hours = append(hours, hour)
				break
			}
		}
	}
	return hours
}
