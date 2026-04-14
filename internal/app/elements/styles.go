package elements

import (
	"image/color"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
)

var (
	// Colors — set by InitPalette based on terminal background.
	ColorPrimary     color.Color
	ColorSecondary   color.Color
	ColorAccent      color.Color
	ColorHighlight   color.Color
	ColorSelectedFg  color.Color
	ColorDiffRemove  color.Color
	ColorDiffHunk    color.Color
	ColorToolBg      color.Color
	ColorFgOnBg      color.Color // foreground on toolBg / status surfaces
	ColorStatusFg    color.Color
	ColorNormalTitle color.Color
	ColorNormalDesc  color.Color
	ColorTitleFg     color.Color // list title text on primary bg
	ColorChartBar    color.Color
	ColorChartToken  color.Color
	ColorChartTime   color.Color
	ColorChartError  color.Color
	ColorHeatmap0    color.Color
	ColorHeatmap1    color.Color
	ColorHeatmap2    color.Color
	ColorHeatmap3    color.Color
	ColorHeatmap4    color.Color

	// Styles — rebuilt by InitPalette.
	StyleSubtitle             lipgloss.Style
	StyleToolCall             lipgloss.Style
	StyleToolCallItalic       lipgloss.Style
	StyleMetaLabel            lipgloss.Style
	StyleMetaValue            lipgloss.Style
	StyleSearchMatch          lipgloss.Style
	StyleCurrentMatch         lipgloss.Style
	StyleRuleHR               lipgloss.Style
	StyleBadgeUser            lipgloss.Style
	StyleBadgeAssistant       lipgloss.Style
	StyleBadgeSystem          lipgloss.Style
	StyleThinkLabel           lipgloss.Style
	StyleThinkBorder          lipgloss.Style
	StyleThinkLine            lipgloss.Style
	StyleSelectedPreview      lipgloss.Style
	StyleNormalPreview        lipgloss.Style
	StyleDimmedPreview        lipgloss.Style
	StyleDiffBg               lipgloss.Style
	StyleDiffAdd              lipgloss.Style
	StyleDiffRemoveLine       lipgloss.Style
	StyleDiffHunkLine         lipgloss.Style
	StyleToolResultBadge      lipgloss.Style
	StyleToolResultErrorBadge lipgloss.Style
	StylePaneTitle            lipgloss.Style

	paletteOnce = &sync.Once{}
)

// InitPalette sets the colour palette and derived styles based on
// whether the terminal has a dark background.
func InitPalette(hasDarkBG bool) {
	paletteOnce.Do(func() {
		ld := lipgloss.LightDark(hasDarkBG)

		// (light-bg variant, dark-bg variant)
		ColorPrimary = ld(lipgloss.Color("55"), lipgloss.Color("99"))
		ColorSecondary = ld(lipgloss.Color("245"), lipgloss.Color("243"))
		ColorAccent = ld(lipgloss.Color("28"), lipgloss.Color("114"))
		ColorHighlight = ld(lipgloss.Color("225"), lipgloss.Color("53"))
		ColorSelectedFg = ld(lipgloss.Color("22"), lipgloss.Color("156"))
		ColorDiffRemove = ld(lipgloss.Color("124"), lipgloss.Color("203"))
		ColorDiffHunk = ld(lipgloss.Color("30"), lipgloss.Color("37"))
		ColorToolBg = ld(lipgloss.Color("254"), lipgloss.Color("236"))
		ColorFgOnBg = ld(lipgloss.Color("238"), lipgloss.Color("252"))
		ColorStatusFg = ld(lipgloss.Color("232"), lipgloss.Color("255"))
		ColorNormalTitle = ld(lipgloss.Color("240"), lipgloss.Color("249"))
		ColorNormalDesc = ld(lipgloss.Color("245"), lipgloss.Color("243"))
		ColorTitleFg = ld(lipgloss.Color("255"), lipgloss.Color("230"))
		ColorChartBar = lipgloss.Color("#39d353")
		ColorChartToken = lipgloss.Color("#a371f7")
		ColorChartTime = lipgloss.Color("#d2a8ff")
		ColorChartError = lipgloss.Color("#f85149")
		ColorHeatmap0 = lipgloss.Color("#161b22")
		ColorHeatmap1 = lipgloss.Color("#0e4429")
		ColorHeatmap2 = lipgloss.Color("#006d32")
		ColorHeatmap3 = lipgloss.Color("#26a641")
		ColorHeatmap4 = lipgloss.Color("#39d353")

		StyleSubtitle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

		StyleToolCall = lipgloss.NewStyle().
			Foreground(ColorAccent)

		StyleToolCallItalic = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Italic(true)

		StyleMetaLabel = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

		StyleMetaValue = lipgloss.NewStyle().
			Foreground(ColorNormalTitle)

		StyleSearchMatch = lipgloss.NewStyle().
			Background(ColorHighlight)

		colorCurrentMatch := ld(lipgloss.Color("201"), lipgloss.Color("129"))
		StyleCurrentMatch = lipgloss.NewStyle().
			Background(colorCurrentMatch).
			Bold(true)

		StyleRuleHR = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

		StyleBadgeUser = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

		StyleBadgeAssistant = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

		StyleBadgeSystem = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

		StyleThinkLabel = lipgloss.NewStyle().
			Italic(true).
			Foreground(ColorSecondary)

		StyleThinkBorder = lipgloss.NewStyle().
			Foreground(ColorSecondary)

		StyleThinkLine = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true)

		StyleSelectedPreview = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(ColorAccent).
			Foreground(ColorSelectedFg).
			Padding(0, 0, 0, 1)

		StyleNormalPreview = lipgloss.NewStyle().
			Foreground(ColorNormalTitle).
			Bold(true).
			Padding(0, 0, 0, 2)

		StyleDimmedPreview = lipgloss.NewStyle().
			Foreground(ColorNormalDesc).
			Padding(0, 0, 0, 2)

		StyleDiffBg = lipgloss.NewStyle().
			Background(ColorToolBg).
			Foreground(ColorFgOnBg)

		StyleDiffAdd = lipgloss.NewStyle().
			Background(ColorToolBg).
			Foreground(ColorAccent)

		StyleDiffRemoveLine = lipgloss.NewStyle().
			Background(ColorToolBg).
			Foreground(ColorDiffRemove)

		StyleDiffHunkLine = lipgloss.NewStyle().
			Background(ColorToolBg).
			Foreground(ColorDiffHunk)

		StyleToolResultBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorStatusFg).
			Background(ColorPrimary).
			Padding(0, 1)

		StyleToolResultErrorBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorStatusFg).
			Background(ColorDiffRemove).
			Padding(0, 1)

		StylePaneTitle = lipgloss.NewStyle().
			Foreground(ColorTitleFg).
			Bold(true).
			Padding(0, 1)
	})
}

func InitPaletteForTest(hasDarkBG bool) {
	paletteOnce = &sync.Once{}
	InitPalette(hasDarkBG)
}

// RenderBorderTop builds a custom top border line with an embedded title badge:
// ╭─ Title ──────────────────────╮
func RenderBorderTop(title string, width int, fg, bg color.Color) string {
	border := lipgloss.RoundedBorder()
	bs := lipgloss.NewStyle().Foreground(fg)
	ts := StylePaneTitle.Background(bg)

	titleRendered := ts.Render(title)
	titleW := lipgloss.Width(titleRendered)
	innerWidth := width - 2 // minus two corners
	padLen := max(innerWidth-titleW-3, 0)

	return bs.Render(string(border.TopLeft)+"─") +
		" " + titleRendered + " " +
		bs.Render(strings.Repeat("─", padLen)+string(border.TopRight))
}

func RenderFramedPane(title string, width, bodyHeight int, borderColor color.Color, content string) string {
	topBorder := RenderBorderTop(title, width, borderColor, borderColor)
	return topBorder + "\n" + RenderFramedBody(width, bodyHeight, borderColor, content)
}

func RenderFramedBox(title string, width int, borderColor color.Color, content string) string {
	topBorder := RenderBorderTop(title, width, borderColor, borderColor)
	return topBorder + "\n" + RenderFramedBody(width, 0, borderColor, content)
}

func RenderInsetBox(width int, borderColor color.Color, content string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width).
		Render(content)
}

func RenderFramedBody(width, bodyHeight int, borderColor color.Color, content string) string {
	innerWidth := max(width-2, 1)
	blankLine := strings.Repeat(" ", innerWidth)
	lines := SplitAndFitLines(content, innerWidth)
	if len(lines) == 0 {
		lines = []string{blankLine}
	}
	if bodyHeight > 0 {
		switch {
		case len(lines) < bodyHeight:
			for len(lines) < bodyHeight {
				lines = append(lines, blankLine)
			}
		case len(lines) > bodyHeight:
			lines = lines[:bodyHeight]
		}
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	leftBorder := borderStyle.Render("│")
	rightBorder := leftBorder
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", innerWidth) + "╯")

	var body strings.Builder
	body.Grow((len(lines)+1)*(innerWidth+8) + len(bottomBorder))
	for i, line := range lines {
		if i > 0 {
			body.WriteByte('\n')
		}
		body.WriteString(leftBorder)
		body.WriteString(line)
		body.WriteString(rightBorder)
	}
	body.WriteByte('\n')
	body.WriteString(bottomBorder)
	return body.String()
}
