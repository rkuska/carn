package elements

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	HelpOverlayIndent    = "  "
	HelpOverlayColumnGap = "  "
)

func RenderHelpOverlay(width, height int, title string, sections []HelpSection) string {
	boxWidth := min(max(width-8, 40), 96)
	bodyHeight := max(height-FramedFooterRows, 1)
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
	box := RenderFramedBox(title, boxWidth, ColorPrimary, content)
	return lipgloss.Place(width, bodyHeight, lipgloss.Center, lipgloss.Center, box)
}

func renderHelpOverlaySection(section HelpSection, contentWidth int) []string {
	lines := []string{}
	if section.Title != "" {
		title := ansi.Truncate(HelpOverlayIndent+section.Title, contentWidth, "…")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(title))
	}
	if len(section.Items) == 0 {
		return lines
	}

	rowWidth := max(contentWidth-lipgloss.Width(HelpOverlayIndent), 1)
	keyWidth, descWidth := helpOverlayColumnWidths(section.Items, rowWidth)
	for _, item := range section.Items {
		for _, row := range renderHelpOverlayItemRows(item, keyWidth, descWidth, rowWidth) {
			lines = append(lines, HelpOverlayIndent+row)
		}
	}
	return lines
}

func helpOverlayColumnWidths(items []HelpItem, rowWidth int) (int, int) {
	keyWidth := 1
	descWidth := 1
	for _, item := range items {
		keyWidth = max(keyWidth, lipgloss.Width(HelpItemKeyText(item)))
		descWidth = max(descWidth, lipgloss.Width(item.Desc))
	}

	maxDescWidth := max(rowWidth-keyWidth-(2*lipgloss.Width(HelpOverlayColumnGap))-1, 1)
	if descWidth > maxDescWidth {
		descWidth = maxDescWidth
	}
	return keyWidth, descWidth
}

func renderHelpOverlayItemRows(item HelpItem, keyWidth, descWidth, width int) []string {
	if width <= 0 {
		return nil
	}

	keyWidth = max(keyWidth, 1)
	descWidth = max(descWidth, 1)
	gapWidth := lipgloss.Width(HelpOverlayColumnGap)
	detailWidth := max(width-keyWidth-descWidth-(2*gapWidth), 1)

	detailText := item.Detail
	if detailText == "" {
		detailText = item.Desc
	}
	detailLines := strings.Split(ansi.Wordwrap(detailText, detailWidth, ""), "\n")
	if len(detailLines) == 0 {
		detailLines = []string{""}
	}

	rows := make([]string, 0, len(detailLines))
	for i, detailLine := range detailLines {
		detailLine = strings.TrimSpace(detailLine)
		keyText := ""
		descText := ""
		if i == 0 {
			keyText = ansi.Truncate(HelpItemKeyText(item), keyWidth, "")
			descText = ansi.Truncate(item.Desc, descWidth, "…")
		}

		keyPart := FitToWidth(helpItemKeyStyle(item).Render(keyText), keyWidth)
		descPart := FitToWidth(lipgloss.NewStyle().Foreground(ColorNormalTitle).Render(descText), descWidth)
		detailPart := FitToWidth(lipgloss.NewStyle().Foreground(ColorNormalDesc).Render(detailLine), detailWidth)
		rows = append(rows, keyPart+HelpOverlayColumnGap+descPart+HelpOverlayColumnGap+detailPart)
	}
	return rows
}
