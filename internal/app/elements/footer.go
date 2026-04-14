package elements

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	FramedFooterRows   = 2
	FramedChromeHeight = 4 // top border + bottom border + footer rows
)

func FramedBodyHeight(totalHeight int) int {
	return max(totalHeight-FramedChromeHeight, 1)
}

func FramedFooterContentWidth(width int) int {
	return max(width-2, 1)
}

func RenderFramedFooter(width int, topRow, statusRow string) string {
	rowStyle := lipgloss.NewStyle().Padding(0, 1).Width(width)
	contentWidth := FramedFooterContentWidth(width)

	rows := []string{
		rowStyle.Render(TrimFooterRow(topRow, contentWidth)),
		rowStyle.Render(TrimFooterRow(statusRow, contentWidth)),
	}
	return strings.Join(rows, "\n")
}

func ComposeFooterRow(width int, left, right string) string {
	contentWidth := FramedFooterContentWidth(width)
	if right == "" {
		return TrimFooterRow(left, contentWidth)
	}

	right = TruncateFooterText(right, contentWidth)
	rightWidth := lipgloss.Width(right)
	if rightWidth >= contentWidth {
		return FitToWidth(right, contentWidth)
	}

	leftWidth := max(contentWidth-rightWidth-1, 0)
	left = TruncateFooterText(left, leftWidth)

	gap := max(contentWidth-lipgloss.Width(left)-rightWidth, 1)
	return FitToWidth(left+strings.Repeat(" ", gap)+right, contentWidth)
}

func TrimFooterRow(row string, width int) string {
	return FitToWidth(TruncateFooterText(row, width), width)
}

func TruncateFooterText(row string, width int) string {
	if width <= 0 {
		return ""
	}

	tail := ""
	if width > 1 {
		tail = "…"
	}

	return ansi.Truncate(row, width, tail)
}
