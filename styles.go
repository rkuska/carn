package main

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	// Colors — set by initPalette based on terminal background.
	colorPrimary     color.Color
	colorSecondary   color.Color
	colorAccent      color.Color
	colorHighlight   color.Color
	colorSelectedFg  color.Color
	colorDiffRemove  color.Color
	colorDiffHunk    color.Color
	colorToolBg      color.Color
	colorFgOnBg      color.Color // foreground on toolBg / status surfaces
	colorStatusFg    color.Color
	colorNormalTitle color.Color
	colorNormalDesc  color.Color
	colorTitleFg     color.Color // list title text on primary bg

	// Styles — rebuilt by initPalette.
	styleSubtitle  lipgloss.Style
	styleToolCall  lipgloss.Style
	styleMetaLabel lipgloss.Style
	styleMetaValue lipgloss.Style
)

// initPalette sets the colour palette and derived styles based on
// whether the terminal has a dark background.
func initPalette(hasDarkBG bool) {
	ld := lipgloss.LightDark(hasDarkBG)

	// (light-bg variant, dark-bg variant)
	colorPrimary = ld(lipgloss.Color("55"), lipgloss.Color("99"))
	colorSecondary = ld(lipgloss.Color("245"), lipgloss.Color("243"))
	colorAccent = ld(lipgloss.Color("28"), lipgloss.Color("114"))
	colorHighlight = ld(lipgloss.Color("225"), lipgloss.Color("53"))
	colorSelectedFg = ld(lipgloss.Color("22"), lipgloss.Color("156"))
	colorDiffRemove = ld(lipgloss.Color("124"), lipgloss.Color("203"))
	colorDiffHunk = ld(lipgloss.Color("30"), lipgloss.Color("37"))
	colorToolBg = ld(lipgloss.Color("254"), lipgloss.Color("236"))
	colorFgOnBg = ld(lipgloss.Color("238"), lipgloss.Color("252"))
	colorStatusFg = ld(lipgloss.Color("232"), lipgloss.Color("255"))
	colorNormalTitle = ld(lipgloss.Color("240"), lipgloss.Color("249"))
	colorNormalDesc = ld(lipgloss.Color("245"), lipgloss.Color("243"))
	colorTitleFg = ld(lipgloss.Color("255"), lipgloss.Color("230"))

	styleSubtitle = lipgloss.NewStyle().
		Foreground(colorSecondary)

	styleToolCall = lipgloss.NewStyle().
		Foreground(colorAccent)

	styleMetaLabel = lipgloss.NewStyle().
		Foreground(colorSecondary).
		Bold(true)

	styleMetaValue = lipgloss.NewStyle().
		Foreground(colorNormalTitle)
}

// renderBorderTop builds a custom top border line with an embedded title badge:
// ╭─ Title ──────────────────────╮
func renderBorderTop(title string, width int, fg, bg color.Color) string {
	border := lipgloss.RoundedBorder()
	bs := lipgloss.NewStyle().Foreground(fg)
	ts := lipgloss.NewStyle().
		Background(bg).
		Foreground(colorTitleFg).
		Bold(true).
		Padding(0, 1)

	titleRendered := ts.Render(title)
	titleW := lipgloss.Width(titleRendered)
	innerWidth := width - 2 // minus two corners
	padLen := max(innerWidth-titleW-3, 0)

	return bs.Render(string(border.TopLeft)+"─") +
		" " + titleRendered + " " +
		bs.Render(strings.Repeat("─", padLen)+string(border.TopRight))
}

func renderFramedPane(title string, width, bodyHeight int, borderColor color.Color, content string) string {
	topBorder := renderBorderTop(title, width, borderColor, borderColor)
	body := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		BorderForeground(borderColor).
		Width(width).
		Height(bodyHeight).
		MaxHeight(bodyHeight + 1).
		Render(content)
	return topBorder + "\n" + body
}

func renderFramedBox(title string, width int, borderColor color.Color, content string) string {
	topBorder := renderBorderTop(title, width, borderColor, borderColor)
	body := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		BorderForeground(borderColor).
		Width(width).
		Render(content)
	return topBorder + "\n" + body
}

func renderInsetBox(width int, borderColor color.Color, content string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width).
		Render(content)
}
