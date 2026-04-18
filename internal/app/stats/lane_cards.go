package stats

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	el "github.com/rkuska/carn/internal/app/elements"
)

type statsLaneContentBuilder func(bodyWidth int) string
type statsLaneRenderer func(index, width int, selected bool) string

func renderStatsLaneBox(theme *el.Theme, title string, selected bool, width int, content string) string {
	if width <= 0 {
		return ""
	}
	return renderFramedBox(
		theme,
		selectedStatsTitle(title, selected),
		width,
		statsLaneBorderColor(theme, selected),
		"\n"+content,
	)
}

func renderStatsLanePane(theme *el.Theme, title string, selected bool, width, bodyHeight int, content string) string {
	if width <= 0 {
		return ""
	}
	return renderFramedPane(
		theme,
		selectedStatsTitle(title, selected),
		width,
		max(bodyHeight, 1)+1,
		statsLaneBorderColor(theme, selected),
		"\n"+content,
	)
}

func renderStatsLanePair(
	theme *el.Theme,
	width, minColumnWidth int,
	leftTitle string,
	leftSelected bool,
	leftContent statsLaneContentBuilder,
	rightTitle string,
	rightSelected bool,
	rightContent statsLaneContentBuilder,
) string {
	leftWidth, rightWidth, stacked := statsColumnWidths(width, 1, 1, minColumnWidth)
	if stacked {
		bodyWidth := statsLaneBodyWidth(width)
		return strings.Join([]string{
			renderStatsLaneBox(theme, leftTitle, leftSelected, width, leftContent(bodyWidth)),
			renderStatsLaneBox(theme, rightTitle, rightSelected, width, rightContent(bodyWidth)),
		}, "\n\n")
	}

	leftBody := leftContent(statsLaneBodyWidth(leftWidth))
	rightBody := rightContent(statsLaneBodyWidth(rightWidth))
	bodyHeight := max(lipgloss.Height(leftBody), lipgloss.Height(rightBody))

	return renderPreformattedColumns(
		theme,
		renderStatsLanePane(theme, leftTitle, leftSelected, leftWidth, bodyHeight, leftBody),
		renderStatsLanePane(theme, rightTitle, rightSelected, rightWidth, bodyHeight, rightBody),
		leftWidth,
		rightWidth,
		false,
	)
}

func renderStatsLaneGrid(
	theme *el.Theme,
	width, minColumnWidth int,
	selectedIndex int,
	render statsLaneRenderer,
) string {
	const laneCount = 4

	leftWidth, rightWidth, stacked := statsColumnWidths(width, 1, 1, minColumnWidth)
	if stacked {
		parts := make([]string, 0, laneCount)
		for index := range laneCount {
			parts = append(parts, render(index, width, index == selectedIndex))
		}
		return strings.Join(parts, "\n\n")
	}

	top := renderPreformattedColumns(
		theme,
		render(0, leftWidth, selectedIndex == 0),
		render(1, rightWidth, selectedIndex == 1),
		leftWidth,
		rightWidth,
		false,
	)
	bottom := renderPreformattedColumns(
		theme,
		render(2, leftWidth, selectedIndex == 2),
		render(3, rightWidth, selectedIndex == 3),
		leftWidth,
		rightWidth,
		false,
	)
	return top + "\n\n" + bottom
}

func statsLaneBodyWidth(width int) int {
	return max(width-2, 1)
}

func statsLaneBorderColor(theme *el.Theme, selected bool) color.Color {
	if selected {
		return theme.ColorAccent
	}
	return theme.ColorPrimary
}
