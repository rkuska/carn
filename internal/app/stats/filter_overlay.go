package stats

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m statsModel) renderStatsFilterOverlay() string {
	boxWidth := min(max(m.contentWidth()-8, 40), 96)
	bodyHeight := max(m.contentHeight(), 1)
	contentWidth := max(boxWidth-2, 1)

	lines := []string{""}
	lines = append(lines, renderStatsSplitRow(m, contentWidth))
	if m.splitExpanded {
		lines = append(lines, renderStatsSplitOptions(m, contentWidth)...)
	}
	lines = append(lines, "")
	for i := range filterDimCount {
		dim := filterDimension(i)
		lines = append(lines, renderFilterDimensionRow(m.theme, m.filter, dim, contentWidth))
		if m.filter.Expanded == int(dim) {
			lines = append(lines, renderFilterExpandedValues(m.theme, m.filter, dim, contentWidth)...)
		}
	}

	lines = append(lines, "")
	lines = append(lines, renderStatsFilterMatchLine(m, contentWidth))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := renderFramedBox(m.theme, "Filter", boxWidth, m.theme.ColorPrimary, content)
	return lipgloss.Place(m.contentWidth(), bodyHeight, lipgloss.Center, lipgloss.Center, box)
}

func renderStatsSplitRow(m statsModel, width int) string {
	cursor := filterOverlayCursorOff
	if m.filter.Cursor == splitRowCursor && !m.splitExpanded {
		cursor = filterOverlayCursorOn
	}

	labelStyle := lipgloss.NewStyle().Foreground(m.theme.ColorAccent).Bold(true)
	label := labelStyle.Render("Split by")
	labelWidth := lipgloss.Width(label)

	value := lipgloss.NewStyle().Foreground(m.theme.ColorNormalDesc).Render("off")
	if m.splitBy.IsActive() {
		value = lipgloss.NewStyle().Foreground(m.theme.ColorAccent).Bold(true).Render(m.splitBy.Label())
	}

	summaryWidth := max(width-lipgloss.Width(filterOverlayIndent+cursor)-labelWidth-2, 1)
	row := filterOverlayIndent + cursor + label + "  " + ansi.Truncate(value, summaryWidth, "…")
	return ansi.Truncate(row, width, "…")
}

func renderStatsSplitOptions(m statsModel, width int) []string {
	indent := filterOverlayIndent + "    "
	lines := make([]string, 0, len(splitDimensionOptions))
	for i, option := range splitDimensionOptions {
		cursor := filterOverlayCursorOff
		if m.splitExpandedCursor == i {
			cursor = filterOverlayCursorOn
		}
		check := filterOverlayCheckOff
		if m.splitBy == option {
			check = lipgloss.NewStyle().Foreground(m.theme.ColorAccent).Render("✓ ")
		}
		lines = append(lines, ansi.Truncate(indent+cursor+check+option.Label(), width, "…"))
	}
	return lines
}

func renderStatsFilterMatchLine(m statsModel, width int) string {
	matched := m.filter.MatchCount(m.conversations)
	total := len(m.conversations)
	text := fmt.Sprintf("%d of %d conversations matching", matched, total)
	rendered := lipgloss.NewStyle().Foreground(m.theme.ColorNormalDesc).Render(text)
	return filterOverlayIndent + ansi.Truncate(rendered, max(width-4, 1), "…")
}

func (m statsModel) statsFilterFooterStatusParts() []string {
	matched := m.filter.MatchCount(m.conversations)
	total := len(m.conversations)
	return []string{fmt.Sprintf("%d/%d conversations", matched, total)}
}

func (m statsModel) statsFilterFooterItems() []helpItem {
	if m.filter.RegexEditing {
		return regexEditingFooterItems()
	}
	if m.splitExpanded {
		return m.splitExpandedFooterItems()
	}
	if m.filter.Expanded >= 0 {
		return expandedDimensionFooterItems()
	}
	if m.filter.Cursor == splitRowCursor {
		return m.splitRowFooterItems()
	}
	return m.filterDimensionFooter()
}

func regexEditingFooterItems() []helpItem {
	return []helpItem{
		{Key: "enter", Desc: "apply"},
		{Key: "esc", Desc: "cancel"},
	}
}

func expandedDimensionFooterItems() []helpItem {
	return []helpItem{
		{Key: "j/k", Desc: "move"},
		{Key: "space", Desc: "toggle"},
		{Key: "enter", Desc: "done"},
		{Key: "/", Desc: "regex"},
		{Key: "x", Desc: "clear"},
		{Key: "q/esc", Desc: "back"},
	}
}

func (m statsModel) splitExpandedFooterItems() []helpItem {
	items := []helpItem{
		{Key: "j/k", Desc: "move"},
		{Key: "space", Desc: "select"},
	}
	if m.splitBy.IsActive() {
		items = append(items, helpItem{Key: "x", Desc: "clear"})
	}
	items = append(items, helpItem{Key: "q/esc", Desc: "back"})
	return items
}

func (m statsModel) splitRowFooterItems() []helpItem {
	items := []helpItem{
		{Key: "j/k", Desc: "move"},
		{Key: "enter", Desc: "select"},
	}
	if m.splitBy.IsActive() {
		items = append(items, helpItem{Key: "x", Desc: "clear"})
	}
	if m.filter.HasActiveFilters() || m.splitBy.IsActive() {
		items = append(items, helpItem{Key: "X", Desc: "clear all"})
	}
	items = append(items, helpItem{Key: "q/esc", Desc: "close"})
	return items
}

func (m statsModel) filterDimensionFooter() []helpItem {
	items := filterDimensionFooterItems(m.filter)
	if m.splitBy.IsActive() && !containsHelpKey(items, "X") {
		items = append(items[:len(items)-1], append([]helpItem{{Key: "X", Desc: "clear all"}}, items[len(items)-1:]...)...)
	}
	return items
}

func containsHelpKey(items []helpItem, key string) bool {
	for _, item := range items {
		if item.Key == key {
			return true
		}
	}
	return false
}
