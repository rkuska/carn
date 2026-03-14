package app

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	helpOverlayIndent    = "  "
	helpOverlayColumnGap = "  "
)

func renderHelpOverlay(width, height int, title string, sections []helpSection) string {
	boxWidth := min(max(width-8, 40), 96)
	bodyHeight := max(height-framedFooterRows, 1)
	contentWidth := max(boxWidth-2, 1)

	lines := []string{""}
	for i, section := range sections {
		lines = append(lines, renderHelpOverlaySection(section, contentWidth)...)
		if i != len(sections)-1 {
			lines = append(lines, "")
		}
	}
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := renderFramedBox(title, boxWidth, colorPrimary, content)
	return lipgloss.Place(width, bodyHeight, lipgloss.Center, lipgloss.Center, box)
}

func renderHelpOverlaySection(section helpSection, contentWidth int) []string {
	lines := []string{}
	if section.title != "" {
		title := ansi.Truncate(helpOverlayIndent+section.title, contentWidth, "…")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(title))
	}
	if len(section.items) == 0 {
		return lines
	}

	rowWidth := max(contentWidth-lipgloss.Width(helpOverlayIndent), 1)
	keyWidth, descWidth := helpOverlayColumnWidths(section.items, rowWidth)
	for _, item := range section.items {
		lines = append(lines, helpOverlayIndent+renderHelpOverlayItemRow(item, keyWidth, descWidth, rowWidth))
	}
	return lines
}

func helpOverlayColumnWidths(items []helpItem, rowWidth int) (int, int) {
	keyWidth := 1
	descWidth := 1
	for _, item := range items {
		keyWidth = max(keyWidth, lipgloss.Width(helpItemKeyText(item)))
		descWidth = max(descWidth, lipgloss.Width(item.desc))
	}

	maxDescWidth := max(rowWidth-keyWidth-(2*lipgloss.Width(helpOverlayColumnGap))-1, 1)
	if descWidth > maxDescWidth {
		descWidth = maxDescWidth
	}
	return keyWidth, descWidth
}

func renderHelpOverlayItemRow(item helpItem, keyWidth, descWidth, width int) string {
	if width <= 0 {
		return ""
	}

	keyWidth = max(keyWidth, 1)
	descWidth = max(descWidth, 1)
	gapWidth := lipgloss.Width(helpOverlayColumnGap)
	detailWidth := max(width-keyWidth-descWidth-(2*gapWidth), 1)

	keyText := ansi.Truncate(helpItemKeyText(item), keyWidth, "")
	descText := ansi.Truncate(item.desc, descWidth, "…")
	detailText := item.detail
	if detailText == "" {
		detailText = item.desc
	}
	detailText = ansi.Truncate(detailText, detailWidth, "…")

	keyPart := fitToWidth(helpItemKeyStyle(item).Render(keyText), keyWidth)
	descPart := fitToWidth(lipgloss.NewStyle().Foreground(colorNormalTitle).Render(descText), descWidth)
	detailPart := fitToWidth(lipgloss.NewStyle().Foreground(colorNormalDesc).Render(detailText), detailWidth)
	return keyPart + helpOverlayColumnGap + descPart + helpOverlayColumnGap + detailPart
}
