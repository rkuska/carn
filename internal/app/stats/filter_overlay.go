package stats

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const statsFilterVersionCursor = int(filterDimCount)

func (m statsModel) renderStatsFilterOverlay() string {
	boxWidth := min(max(m.contentWidth()-8, 40), 96)
	bodyHeight := max(m.contentHeight(), 1)
	contentWidth := max(boxWidth-2, 1)

	lines := []string{""}
	for i := range filterDimCount {
		dim := filterDimension(i)
		lines = append(lines, renderFilterDimensionRow(m.filter, dim, contentWidth))
		if m.filter.Expanded == int(dim) {
			lines = append(lines, renderFilterExpandedValues(m.filter, dim, contentWidth)...)
		}
	}

	lines = append(lines, renderStatsVersionFilterRow(m, contentWidth))
	if m.filter.Expanded == statsFilterVersionCursor {
		lines = append(lines, renderStatsVersionExpandedValues(m, contentWidth)...)
	}

	lines = append(lines, "")
	lines = append(lines, renderStatsFilterMatchLine(m, contentWidth))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := renderFramedBox("Filter", boxWidth, colorPrimary, content)
	return lipgloss.Place(m.contentWidth(), bodyHeight, lipgloss.Center, lipgloss.Center, box)
}

func renderStatsVersionFilterRow(m statsModel, width int) string {
	cursor := filterOverlayCursorOff
	if m.filter.Cursor == statsFilterVersionCursor && m.filter.Expanded < 0 {
		cursor = filterOverlayCursorOn
	}

	labelRendered := styleMetaLabel.Render("Version")
	labelWidth := lipgloss.Width(labelRendered)
	summaryWidth := max(width-lipgloss.Width(filterOverlayIndent+cursor)-labelWidth-2, 1)
	summary := renderSelectionSummary(m.versionFilter, m.versionValues, summaryWidth)
	row := filterOverlayIndent + cursor + labelRendered + "  " + summary
	return ansi.Truncate(row, width, "…")
}

func renderStatsVersionExpandedValues(m statsModel, width int) []string {
	indent := filterOverlayIndent + "    "
	lines := make([]string, 0, len(m.versionValues))
	for i, value := range m.versionValues {
		cursor := filterOverlayCursorOff
		if m.filter.ExpandedCursor == i {
			cursor = filterOverlayCursorOn
		}

		check := filterOverlayCheckOff
		if m.versionFilter.Selected[value] {
			check = lipgloss.NewStyle().Foreground(colorAccent).Render("✓ ")
		}

		lines = append(lines, ansi.Truncate(indent+cursor+check+value, width, "…"))
	}
	return lines
}

func renderStatsFilterMatchLine(m statsModel, width int) string {
	baseSessions := flattenStatsSessions(m.filteredConversations())
	_, filteredSessions := filterStatsConversationsByVersion(m.filteredConversations(), m.versionFilter)
	text := fmt.Sprintf("%d of %d sessions matching", len(filteredSessions), len(baseSessions))
	rendered := lipgloss.NewStyle().Foreground(colorNormalDesc).Render(text)
	return filterOverlayIndent + ansi.Truncate(rendered, max(width-4, 1), "…")
}

func (m statsModel) statsFilterFooterStatusParts() []string {
	baseSessions := flattenStatsSessions(m.filteredConversations())
	_, filteredSessions := filterStatsConversationsByVersion(m.filteredConversations(), m.versionFilter)
	return []string{fmt.Sprintf("%d/%d sessions", len(filteredSessions), len(baseSessions))}
}

func (m statsModel) statsFilterFooterItems() []helpItem {
	if m.filter.RegexEditing {
		return []helpItem{
			{Key: "enter", Desc: "apply"},
			{Key: "esc", Desc: "cancel"},
		}
	}
	if m.filter.Expanded >= 0 {
		items := []helpItem{
			{Key: "j/k", Desc: "move"},
			{Key: "space", Desc: "toggle"},
			{Key: "enter", Desc: "done"},
			{Key: "x", Desc: "clear"},
			{Key: "q/esc", Desc: "back"},
		}
		if m.filter.Expanded != statsFilterVersionCursor {
			items = append(items[:3], append([]helpItem{{Key: "/", Desc: "regex"}}, items[3:]...)...)
		}
		return items
	}

	if m.filter.Cursor == statsFilterVersionCursor {
		items := []helpItem{
			{Key: "j/k", Desc: "move"},
			{Key: "enter", Desc: "select"},
		}
		if m.versionFilter.IsActive() {
			items = append(items, helpItem{Key: "x", Desc: "clear"})
		}
		if m.filter.HasActiveFilters() || m.versionFilter.IsActive() {
			items = append(items, helpItem{Key: "X", Desc: "clear all"})
		}
		items = append(items, helpItem{Key: "q/esc", Desc: "close"})
		return items
	}

	items := filterDimensionFooterItems(m.filter)
	if m.versionFilter.IsActive() && !containsHelpKey(items, "X") {
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
