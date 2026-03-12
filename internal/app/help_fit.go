package app

import "charm.land/lipgloss/v2"

func essentialHelpWidth(items []helpItem) int {
	parts := make([]string, len(items))
	keep := make([]bool, len(items))
	for i, item := range items {
		if item.key == "" || item.desc == "" || item.priority < helpPriorityEssential {
			continue
		}
		parts[i] = renderHelpItem(item)
		keep[i] = true
	}
	return helpPartsWidth(parts, keep)
}

func keepHelpItems(items []helpItem, width int) []bool {
	parts := make([]string, len(items))
	keep := make([]bool, len(items))
	for i, item := range items {
		if item.key == "" || item.desc == "" {
			continue
		}
		parts[i] = renderHelpItem(item)
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

func nextHelpDropIndex(items []helpItem, keep []bool) int {
	index := -1
	priority := helpPriorityEssential + 1
	for i, item := range items {
		if !isDroppableHelpItem(item, keep, i) {
			continue
		}
		if item.priority < priority {
			index = i
			priority = item.priority
			continue
		}
		if item.priority == priority && index != -1 && i < index {
			index = i
		}
	}
	return index
}

func isDroppableHelpItem(item helpItem, keep []bool, index int) bool {
	if index >= len(keep) || !keep[index] {
		return false
	}
	if item.key == "" || item.desc == "" {
		return false
	}
	return item.priority < helpPriorityEssential
}
