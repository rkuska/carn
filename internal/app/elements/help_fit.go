package elements

import "charm.land/lipgloss/v2"

func essentialHelpWidth(items []HelpItem) int {
	parts := make([]string, len(items))
	keep := make([]bool, len(items))
	for i, item := range items {
		if item.Key == "" || item.Desc == "" || item.Priority < HelpPriorityEssential {
			continue
		}
		parts[i] = RenderHelpItem(item)
		keep[i] = true
	}
	return helpPartsWidth(parts, keep)
}

func keepHelpItems(items []HelpItem, width int) []bool {
	parts := make([]string, len(items))
	keep := make([]bool, len(items))
	for i, item := range items {
		if item.Key == "" || item.Desc == "" {
			continue
		}
		parts[i] = RenderHelpItem(item)
		keep[i] = true
	}

	if helpPartsWidth(parts, keep) <= width {
		return keep
	}

	for helpPartsWidth(parts, keep) > width {
		index := nextHelpDropIndex(items, keep)
		if index == -1 {
			break
		}
		keep[index] = false
	}

	return keep
}

func helpPartsWidth(parts []string, keep []bool) int {
	width := 0
	for i, part := range parts {
		if !keep[i] {
			continue
		}
		if width > 0 {
			width += 2
		}
		width += lipgloss.Width(part)
	}
	return width
}

func nextHelpDropIndex(items []HelpItem, keep []bool) int {
	index := -1
	priority := HelpPriorityEssential + 1
	for i, item := range items {
		if !isDroppableHelpItem(item, keep, i) {
			continue
		}
		if item.Priority < priority {
			index = i
			priority = item.Priority
			continue
		}
		if item.Priority == priority && index != -1 && i < index {
			index = i
		}
	}
	return index
}

func isDroppableHelpItem(item HelpItem, keep []bool, index int) bool {
	if index >= len(keep) || !keep[index] {
		return false
	}
	if item.Key == "" || item.Desc == "" {
		return false
	}
	return item.Priority < HelpPriorityEssential
}
