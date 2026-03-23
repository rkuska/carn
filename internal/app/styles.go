package app

import (
	"image/color"
	"strings"
	"sync"

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
	colorChartBar    color.Color
	colorChartToken  color.Color
	colorChartTime   color.Color
	colorChartError  color.Color
	colorHeatmap0    color.Color
	colorHeatmap1    color.Color
	colorHeatmap2    color.Color
	colorHeatmap3    color.Color
	colorHeatmap4    color.Color

	// Styles — rebuilt by initPalette.
	styleSubtitle             lipgloss.Style
	styleToolCall             lipgloss.Style
	styleToolCallItalic       lipgloss.Style
	styleMetaLabel            lipgloss.Style
	styleMetaValue            lipgloss.Style
	styleSearchMatch          lipgloss.Style
	styleCurrentMatch         lipgloss.Style
	styleRuleHR               lipgloss.Style
	styleBadgeUser            lipgloss.Style
	styleBadgeAssistant       lipgloss.Style
	styleBadgeSystem          lipgloss.Style
	styleThinkLabel           lipgloss.Style
	styleThinkBorder          lipgloss.Style
	styleThinkLine            lipgloss.Style
	styleSelectedPreview      lipgloss.Style
	styleNormalPreview        lipgloss.Style
	styleDimmedPreview        lipgloss.Style
	styleDiffBg               lipgloss.Style
	styleDiffAdd              lipgloss.Style
	styleDiffRemoveLine       lipgloss.Style
	styleDiffHunkLine         lipgloss.Style
	styleToolResultBadge      lipgloss.Style
	styleToolResultErrorBadge lipgloss.Style
	stylePaneTitle            lipgloss.Style

	paletteOnce = &sync.Once{}
)

// initPalette sets the colour palette and derived styles based on
// whether the terminal has a dark background.
func initPalette(hasDarkBG bool) {
	paletteOnce.Do(func() {
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
		colorChartBar = lipgloss.Color("#39d353")
		colorChartToken = lipgloss.Color("#58a6ff")
		colorChartTime = lipgloss.Color("#d2a8ff")
		colorChartError = lipgloss.Color("#f85149")
		colorHeatmap0 = lipgloss.Color("#161b22")
		colorHeatmap1 = lipgloss.Color("#0e4429")
		colorHeatmap2 = lipgloss.Color("#006d32")
		colorHeatmap3 = lipgloss.Color("#26a641")
		colorHeatmap4 = lipgloss.Color("#39d353")

		styleSubtitle = lipgloss.NewStyle().
			Foreground(colorSecondary)

		styleToolCall = lipgloss.NewStyle().
			Foreground(colorAccent)

		styleToolCallItalic = lipgloss.NewStyle().
			Foreground(colorAccent).
			Italic(true)

		styleMetaLabel = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

		styleMetaValue = lipgloss.NewStyle().
			Foreground(colorNormalTitle)

		styleSearchMatch = lipgloss.NewStyle().
			Background(colorHighlight)

		colorCurrentMatch := ld(lipgloss.Color("201"), lipgloss.Color("129"))
		styleCurrentMatch = lipgloss.NewStyle().
			Background(colorCurrentMatch).
			Bold(true)

		styleRuleHR = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

		styleBadgeUser = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

		styleBadgeAssistant = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

		styleBadgeSystem = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary)

		styleThinkLabel = lipgloss.NewStyle().
			Italic(true).
			Foreground(colorSecondary)

		styleThinkBorder = lipgloss.NewStyle().
			Foreground(colorSecondary)

		styleThinkLine = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Italic(true)

		styleSelectedPreview = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(colorAccent).
			Foreground(colorSelectedFg).
			Padding(0, 0, 0, 1)

		styleNormalPreview = lipgloss.NewStyle().
			Foreground(colorNormalTitle).
			Bold(true).
			Padding(0, 0, 0, 2)

		styleDimmedPreview = lipgloss.NewStyle().
			Foreground(colorNormalDesc).
			Padding(0, 0, 0, 2)

		styleDiffBg = lipgloss.NewStyle().
			Background(colorToolBg).
			Foreground(colorFgOnBg)

		styleDiffAdd = lipgloss.NewStyle().
			Background(colorToolBg).
			Foreground(colorAccent)

		styleDiffRemoveLine = lipgloss.NewStyle().
			Background(colorToolBg).
			Foreground(colorDiffRemove)

		styleDiffHunkLine = lipgloss.NewStyle().
			Background(colorToolBg).
			Foreground(colorDiffHunk)

		styleToolResultBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorStatusFg).
			Background(colorPrimary).
			Padding(0, 1)

		styleToolResultErrorBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorStatusFg).
			Background(colorDiffRemove).
			Padding(0, 1)

		stylePaneTitle = lipgloss.NewStyle().
			Foreground(colorTitleFg).
			Bold(true).
			Padding(0, 1)
	})
}

func initPaletteForTest(hasDarkBG bool) {
	paletteOnce = &sync.Once{}
	initPalette(hasDarkBG)
}

// renderBorderTop builds a custom top border line with an embedded title badge:
// ╭─ Title ──────────────────────╮
func renderBorderTop(title string, width int, fg, bg color.Color) string {
	border := lipgloss.RoundedBorder()
	bs := lipgloss.NewStyle().Foreground(fg)
	ts := stylePaneTitle.Background(bg)

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
	innerWidth := max(width-2, 1)
	bodyContent := lipgloss.NewStyle().
		Width(innerWidth).
		Height(bodyHeight).
		MaxHeight(bodyHeight).
		Render(content)
	body := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		BorderForeground(borderColor).
		Width(width).
		Render(bodyContent)
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
