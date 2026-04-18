package elements

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	ColorPrimary     color.Color
	ColorSecondary   color.Color
	ColorAccent      color.Color
	ColorHighlight   color.Color
	ColorSelectedFg  color.Color
	ColorDiffRemove  color.Color
	ColorDiffHunk    color.Color
	ColorToolBg      color.Color
	ColorFgOnBg      color.Color
	ColorStatusFg    color.Color
	ColorNormalTitle color.Color
	ColorNormalDesc  color.Color
	ColorTitleFg     color.Color
	ColorChartBar    color.Color
	ColorChartToken  color.Color
	ColorChartTime   color.Color
	ColorChartError  color.Color
	ColorHeatmap0    color.Color
	ColorHeatmap1    color.Color
	ColorHeatmap2    color.Color
	ColorHeatmap3    color.Color
	ColorHeatmap4    color.Color

	StyleSubtitle             lipgloss.Style
	StyleToolCall             lipgloss.Style
	StyleToolCallItalic       lipgloss.Style
	StyleMetaLabel            lipgloss.Style
	StyleMetaValue            lipgloss.Style
	StyleSearchMatch          lipgloss.Style
	StyleCurrentMatch         lipgloss.Style
	StyleRuleHR               lipgloss.Style
	StyleHistogramAxisLabel   lipgloss.Style
	StyleHistogramAxisLine    lipgloss.Style
	StyleHistogramValueLabel  lipgloss.Style
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
	StyleHeatmapCells         [5]lipgloss.Style
}

func NewTheme(hasDarkBG bool) *Theme {
	ld := lipgloss.LightDark(hasDarkBG)
	theme := &Theme{
		ColorPrimary:     ld(lipgloss.Color("55"), lipgloss.Color("99")),
		ColorSecondary:   ld(lipgloss.Color("245"), lipgloss.Color("243")),
		ColorAccent:      ld(lipgloss.Color("28"), lipgloss.Color("114")),
		ColorHighlight:   ld(lipgloss.Color("225"), lipgloss.Color("53")),
		ColorSelectedFg:  ld(lipgloss.Color("22"), lipgloss.Color("156")),
		ColorDiffRemove:  ld(lipgloss.Color("124"), lipgloss.Color("203")),
		ColorDiffHunk:    ld(lipgloss.Color("30"), lipgloss.Color("37")),
		ColorToolBg:      ld(lipgloss.Color("254"), lipgloss.Color("236")),
		ColorFgOnBg:      ld(lipgloss.Color("238"), lipgloss.Color("252")),
		ColorStatusFg:    ld(lipgloss.Color("232"), lipgloss.Color("255")),
		ColorNormalTitle: ld(lipgloss.Color("240"), lipgloss.Color("249")),
		ColorNormalDesc:  ld(lipgloss.Color("245"), lipgloss.Color("243")),
		ColorTitleFg:     ld(lipgloss.Color("255"), lipgloss.Color("230")),
		ColorChartBar:    lipgloss.Color("#39d353"),
		ColorChartToken:  lipgloss.Color("#a371f7"),
		ColorChartTime:   lipgloss.Color("#d2a8ff"),
		ColorChartError:  lipgloss.Color("#f85149"),
		ColorHeatmap0:    lipgloss.Color("#161b22"),
		ColorHeatmap1:    lipgloss.Color("#0e4429"),
		ColorHeatmap2:    lipgloss.Color("#006d32"),
		ColorHeatmap3:    lipgloss.Color("#26a641"),
		ColorHeatmap4:    lipgloss.Color("#39d353"),
	}

	theme.StyleSubtitle = lipgloss.NewStyle().
		Foreground(theme.ColorSecondary)

	theme.StyleToolCall = lipgloss.NewStyle().
		Foreground(theme.ColorAccent)

	theme.StyleToolCallItalic = lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Italic(true)

	theme.StyleMetaLabel = lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Bold(true)

	theme.StyleMetaValue = lipgloss.NewStyle().
		Foreground(theme.ColorNormalTitle)

	theme.StyleSearchMatch = lipgloss.NewStyle().
		Background(theme.ColorHighlight)

	colorCurrentMatch := ld(lipgloss.Color("201"), lipgloss.Color("129"))
	theme.StyleCurrentMatch = lipgloss.NewStyle().
		Background(colorCurrentMatch).
		Bold(true)

	theme.StyleRuleHR = lipgloss.NewStyle().
		Foreground(lipgloss.Color("238"))

	theme.StyleHistogramAxisLabel = lipgloss.NewStyle().
		Foreground(theme.ColorNormalDesc)

	theme.StyleHistogramAxisLine = lipgloss.NewStyle().
		Foreground(theme.ColorSecondary)

	theme.StyleHistogramValueLabel = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff"))

	theme.StyleBadgeUser = lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorPrimary)

	theme.StyleBadgeAssistant = lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	theme.StyleBadgeSystem = lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorSecondary)

	theme.StyleThinkLabel = lipgloss.NewStyle().
		Italic(true).
		Foreground(theme.ColorSecondary)

	theme.StyleThinkBorder = lipgloss.NewStyle().
		Foreground(theme.ColorSecondary)

	theme.StyleThinkLine = lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Italic(true)

	theme.StyleSelectedPreview = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(theme.ColorAccent).
		Foreground(theme.ColorSelectedFg).
		Padding(0, 0, 0, 1)

	theme.StyleNormalPreview = lipgloss.NewStyle().
		Foreground(theme.ColorNormalTitle).
		Bold(true).
		Padding(0, 0, 0, 2)

	theme.StyleDimmedPreview = lipgloss.NewStyle().
		Foreground(theme.ColorNormalDesc).
		Padding(0, 0, 0, 2)

	theme.StyleDiffBg = lipgloss.NewStyle().
		Background(theme.ColorToolBg).
		Foreground(theme.ColorFgOnBg)

	theme.StyleDiffAdd = lipgloss.NewStyle().
		Background(theme.ColorToolBg).
		Foreground(theme.ColorAccent)

	theme.StyleDiffRemoveLine = lipgloss.NewStyle().
		Background(theme.ColorToolBg).
		Foreground(theme.ColorDiffRemove)

	theme.StyleDiffHunkLine = lipgloss.NewStyle().
		Background(theme.ColorToolBg).
		Foreground(theme.ColorDiffHunk)

	theme.StyleToolResultBadge = lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorStatusFg).
		Background(theme.ColorPrimary).
		Padding(0, 1)

	theme.StyleToolResultErrorBadge = lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorStatusFg).
		Background(theme.ColorDiffRemove).
		Padding(0, 1)

	theme.StylePaneTitle = lipgloss.NewStyle().
		Foreground(theme.ColorTitleFg).
		Bold(true).
		Padding(0, 1)

	theme.StyleHeatmapCells = [5]lipgloss.Style{
		lipgloss.NewStyle().Foreground(theme.ColorHeatmap0),
		lipgloss.NewStyle().Foreground(theme.ColorHeatmap1),
		lipgloss.NewStyle().Foreground(theme.ColorHeatmap2),
		lipgloss.NewStyle().Foreground(theme.ColorHeatmap3),
		lipgloss.NewStyle().Foreground(theme.ColorHeatmap4),
	}

	return theme
}

// RenderBorderTop builds a custom top border line with an embedded title badge:
// ╭─ Title ──────────────────────╮
func (t *Theme) RenderBorderTop(title string, width int, fg, bg color.Color) string {
	border := lipgloss.RoundedBorder()
	bs := lipgloss.NewStyle().Foreground(fg)
	ts := t.StylePaneTitle.Background(bg)

	titleRendered := ts.Render(title)
	titleW := lipgloss.Width(titleRendered)
	innerWidth := width - 2
	padLen := max(innerWidth-titleW-3, 0)

	return bs.Render(string(border.TopLeft)+"─") +
		" " + titleRendered + " " +
		bs.Render(strings.Repeat("─", padLen)+string(border.TopRight))
}

func (t *Theme) RenderFramedPane(
	title string,
	width, bodyHeight int,
	borderColor color.Color,
	content string,
) string {
	topBorder := t.RenderBorderTop(title, width, borderColor, borderColor)
	return topBorder + "\n" + RenderFramedBody(width, bodyHeight, borderColor, content)
}

func (t *Theme) RenderFramedBox(title string, width int, borderColor color.Color, content string) string {
	topBorder := t.RenderBorderTop(title, width, borderColor, borderColor)
	return topBorder + "\n" + RenderFramedBody(width, 0, borderColor, content)
}

// RenderInlineTitledRule renders a horizontal rule with an inline title, like
// "─── Title ────────────". When rightMeta is non-empty it is embedded on the
// right side, like "─── Title ─── rightMeta ───"; the caller owns its
// styling. If the width cannot fit the full layout with rightMeta, the meta
// segment is dropped and the rule falls back to a title-only render. The
// title is foregrounded in titleColor, and the dashes use the theme's
// StyleRuleHR styling.
func (t *Theme) RenderInlineTitledRule(title, rightMeta string, width int, titleColor color.Color) string {
	if width <= 0 {
		return ""
	}
	if title == "" {
		return t.StyleRuleHR.Render(strings.Repeat("─", width))
	}

	titleStyle := lipgloss.NewStyle().Foreground(titleColor).Bold(true)
	titleRendered := titleStyle.Render(title)
	titleW := lipgloss.Width(titleRendered)

	const leadDashes = 3
	if rightMeta != "" {
		rightMetaW := lipgloss.Width(rightMeta)
		// Full layout: lead + " " + title + " " + middle(≥1) + " " + meta + " " + trail
		//            = 2*leadDashes + titleW + rightMetaW + 4 + middleDashes
		if middle := width - 2*leadDashes - titleW - rightMetaW - 4; middle >= 1 {
			lead := t.StyleRuleHR.Render(strings.Repeat("─", leadDashes))
			mid := t.StyleRuleHR.Render(strings.Repeat("─", middle))
			trail := t.StyleRuleHR.Render(strings.Repeat("─", leadDashes))
			return lead + " " + titleRendered + " " + mid + " " + rightMeta + " " + trail
		}
	}

	if width <= leadDashes+2+titleW {
		if width <= titleW {
			return FitToWidth(titleStyle.Render(title), width)
		}
		padding := width - titleW - 1
		return t.StyleRuleHR.Render(strings.Repeat("─", padding)) + " " + titleRendered
	}
	lead := t.StyleRuleHR.Render(strings.Repeat("─", leadDashes))
	trail := t.StyleRuleHR.Render(strings.Repeat("─", width-leadDashes-2-titleW))
	return lead + " " + titleRendered + " " + trail
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
