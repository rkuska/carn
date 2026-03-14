package app

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type helpItem struct {
	key      string
	desc     string
	detail   string
	toggle   bool
	on       bool
	glow     bool
	priority helpPriority
}

type helpSection struct {
	title string
	items []helpItem
}

type helpPriority int

const (
	helpPriorityLow helpPriority = iota
	helpPriorityNormal
	helpPriorityHigh
	helpPriorityEssential
)

func renderHelpFooter(width int, items []helpItem, rightParts []string, n notification) string {
	contentWidth := framedFooterContentWidth(width)
	minLeftWidth := essentialHelpWidth(items)
	right := joinNonEmpty(rightParts, "  ")
	maxRightWidth := contentWidth
	if minLeftWidth > 0 && contentWidth > minLeftWidth {
		maxRightWidth = contentWidth - minLeftWidth - 1
	}
	right = truncateFooterText(right, maxRightWidth)
	leftWidth := contentWidth
	if right != "" {
		leftWidth = max(contentWidth-lipgloss.Width(right)-1, 0)
	}

	return renderFramedFooter(
		width,
		composeFooterRow(width, renderFittedHelpItems(items, leftWidth), right),
		renderNotification(n),
	)
}

func renderSearchFooter(width int, prompt, right string, n notification) string {
	return renderFramedFooter(
		width,
		composeFooterRow(width, prompt, right),
		renderNotification(n),
	)
}

func renderHelpItems(items []helpItem) string {
	return renderHelpItemsWithKeep(items, nil)
}

func renderFittedHelpItems(items []helpItem, width int) string {
	if width <= 0 {
		return ""
	}

	keep := keepHelpItems(items, width)
	return fitToWidth(renderHelpItemsWithKeep(items, keep), width)
}

func renderHelpItemsWithKeep(items []helpItem, keep []bool) string {
	helpStyle := lipgloss.NewStyle().Foreground(colorSecondary)

	parts := make([]string, 0, len(items))
	for i, item := range items {
		if keep != nil && !keep[i] {
			continue
		}
		if item.key == "" || item.desc == "" {
			continue
		}

		keyText := helpItemKeyText(item)
		parts = append(parts, helpItemKeyStyle(item).Render(keyText)+helpStyle.Render(" "+item.desc))
	}

	return strings.Join(parts, "  ")
}

func helpItemKeyStyle(item helpItem) lipgloss.Style {
	if item.glow {
		return lipgloss.NewStyle().Foreground(colorPrimary)
	}
	return lipgloss.NewStyle().Foreground(colorAccent)
}

func renderHelpItem(item helpItem) string {
	return renderHelpItems([]helpItem{item})
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

func helpItemKeyText(item helpItem) string {
	if !item.toggle {
		return item.key
	}
	if item.on {
		return "+" + item.key
	}
	return "-" + item.key
}

func withHelpDetail(item helpItem, detail string) helpItem {
	item.detail = detail
	return item
}
