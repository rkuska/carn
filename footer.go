package main

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	framedFooterRows   = 2
	framedChromeHeight = 4 // top border + bottom border + footer rows
)

func framedBodyHeight(totalHeight int) int {
	return max(totalHeight-framedChromeHeight, 1)
}

func framedFooterContentWidth(width int) int {
	return max(width-2, 1)
}

func renderFramedFooter(width int, topRow, statusRow string) string {
	rowStyle := lipgloss.NewStyle().Padding(0, 1).Width(width)
	contentWidth := framedFooterContentWidth(width)

	rows := []string{
		rowStyle.Render(trimFooterRow(topRow, contentWidth)),
		rowStyle.Render(trimFooterRow(statusRow, contentWidth)),
	}
	return strings.Join(rows, "\n")
}

func composeFooterRow(width int, left, right string) string {
	contentWidth := framedFooterContentWidth(width)
	if right == "" {
		return trimFooterRow(left, contentWidth)
	}

	right = truncateFooterText(right, contentWidth)
	rightWidth := lipgloss.Width(right)
	if rightWidth >= contentWidth {
		return fitToWidth(right, contentWidth)
	}

	leftWidth := max(contentWidth-rightWidth-1, 0)
	left = truncateFooterText(left, leftWidth)

	gap := max(contentWidth-lipgloss.Width(left)-rightWidth, 1)
	return fitToWidth(left+strings.Repeat(" ", gap)+right, contentWidth)
}

func trimFooterRow(row string, width int) string {
	return fitToWidth(truncateFooterText(row, width), width)
}

func truncateFooterText(row string, width int) string {
	if width <= 0 {
		return ""
	}

	tail := ""
	if width > 1 {
		tail = "…"
	}

	return ansi.Truncate(row, width, tail)
}
