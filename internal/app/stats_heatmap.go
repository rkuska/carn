package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	heatmapGapStyle = lipgloss.NewStyle().Foreground(colorSecondary)
	heatmapStyles   = [...]lipgloss.Style{
		lipgloss.NewStyle().Foreground(colorHeatmap0),
		lipgloss.NewStyle().Foreground(colorHeatmap1),
		lipgloss.NewStyle().Foreground(colorHeatmap2),
		lipgloss.NewStyle().Foreground(colorHeatmap3),
		lipgloss.NewStyle().Foreground(colorHeatmap4),
	}
)

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

	const prefixWidth = 3

	cellWidth := max((width-prefixWidth)/7, 3)
	lines := make([]string, 0, 26)

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
			lines = append(lines, heatmapGapStyle.Render("···"))
			continue
		}

		var row strings.Builder
		_, _ = fmt.Fprintf(&row, "%02d ", hour)
		for day := range 7 {
			level := heatmapLevel(cells[day][hour], maxValue)
			char, style := heatmapCellStyle(level)
			cell := strings.Repeat(char, cellWidth)
			row.WriteString(style.Render(cell))
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

func heatmapCellStyle(level int) (string, lipgloss.Style) {
	switch level {
	case 1:
		return "░", heatmapStyles[1]
	case 2:
		return "▒", heatmapStyles[2]
	case 3:
		return "▓", heatmapStyles[3]
	case 4:
		return "█", heatmapStyles[4]
	default:
		return " ", heatmapStyles[0]
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
