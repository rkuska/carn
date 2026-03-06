package main

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type helpItem struct {
	key    string
	desc   string
	toggle bool
	on     bool
	glow   bool
}

type helpSection struct {
	title string
	items []helpItem
}

func renderHelpFooter(width int, items []helpItem, rightParts []string, n notification) string {
	return renderFramedFooter(
		width,
		composeFooterRow(width, renderHelpItems(items), joinNonEmpty(rightParts, "  ")),
		renderNotification(n),
	)
}

func renderSearchFooter(width int, prompt string, n notification) string {
	return renderFramedFooter(width, prompt, renderNotification(n))
}

func renderHelpItems(items []helpItem) string {
	helpStyle := lipgloss.NewStyle().Foreground(colorSecondary)
	keyNormal := lipgloss.NewStyle().Foreground(colorAccent)
	keyGlow := lipgloss.NewStyle().Foreground(colorPrimary)

	parts := make([]string, 0, len(items))
	for _, item := range items {
		if item.key == "" || item.desc == "" {
			continue
		}

		keyStyle := keyNormal
		if item.glow {
			keyStyle = keyGlow
		}

		keyText := item.key
		if item.toggle {
			if item.on {
				keyText = "+" + keyText
			} else {
				keyText = "-" + keyText
			}
		}

		parts = append(parts, keyStyle.Render(keyText)+helpStyle.Render(" "+item.desc))
	}

	return strings.Join(parts, "  ")
}

func renderHelpItem(item helpItem) string {
	return renderHelpItems([]helpItem{item})
}

func renderHelpOverlay(width, height int, title string, sections []helpSection) string {
	boxWidth := min(max(width-8, 40), 96)
	bodyHeight := max(height-framedFooterRows, 1)

	var lines []string
	lines = append(lines, "")
	for i, section := range sections {
		if section.title != "" {
			header := lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				Render("  " + section.title)
			lines = append(lines, header)
		}

		for _, item := range section.items {
			lines = append(lines, "  "+renderHelpItem(item))
		}

		if i != len(sections)-1 {
			lines = append(lines, "")
		}
	}
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := renderFramedBox(title, boxWidth, colorPrimary, content)
	return lipgloss.Place(width, bodyHeight, lipgloss.Center, lipgloss.Center, box)
}

func joinNonEmpty(items []string, sep string) string {
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		filtered = append(filtered, item)
	}
	return strings.Join(filtered, sep)
}
