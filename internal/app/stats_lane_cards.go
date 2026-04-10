package app

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

type statsLaneContentBuilder func(bodyWidth int) string

func renderStatsLaneBox(title string, selected bool, width int, content string) string {
	if width <= 0 {
		return ""
	}
	return renderFramedBox(
		selectedStatsTitle(title, selected),
		width,
		statsLaneBorderColor(selected),
		content,
	)
}

func renderStatsLanePane(title string, selected bool, width, bodyHeight int, content string) string {
	if width <= 0 {
		return ""
	}
	return renderFramedPane(
		selectedStatsTitle(title, selected),
		width,
		max(bodyHeight, 1),
		statsLaneBorderColor(selected),
		content,
	)
}

func renderStatsLanePair(
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
			renderStatsLaneBox(leftTitle, leftSelected, width, leftContent(bodyWidth)),
			renderStatsLaneBox(rightTitle, rightSelected, width, rightContent(bodyWidth)),
		}, "\n\n")
	}

	leftBody := leftContent(statsLaneBodyWidth(leftWidth))
	rightBody := rightContent(statsLaneBodyWidth(rightWidth))
	bodyHeight := max(lipgloss.Height(leftBody), lipgloss.Height(rightBody))

	return renderColumns(
		renderStatsLanePane(leftTitle, leftSelected, leftWidth, bodyHeight, leftBody),
		renderStatsLanePane(rightTitle, rightSelected, rightWidth, bodyHeight, rightBody),
		leftWidth,
		rightWidth,
		false,
	)
}

func statsLaneBodyWidth(width int) int {
	return max(width-2, 1)
}

func statsLaneBorderColor(selected bool) color.Color {
	if selected {
		return colorAccent
	}
	return colorPrimary
}
