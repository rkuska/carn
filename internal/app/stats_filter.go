package app

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	conv "github.com/rkuska/carn/internal/conversation"
)

func renderFilterOverlayWithConversations(
	conversations []conv.Conversation,
	filter browserFilterState,
	width, height int,
) string {
	boxWidth := min(max(width-8, 40), 96)
	bodyHeight := max(height-framedFooterRows, 1)
	contentWidth := max(boxWidth-2, 1)

	lines := []string{""}
	for i := range filterDimCount {
		dim := filterDimension(i)
		lines = append(lines, renderFilterDimensionRow(filter, dim, contentWidth))
		if filter.expanded == int(dim) {
			lines = append(lines, renderFilterExpandedValues(filter, dim, contentWidth)...)
		}
	}

	lines = append(lines, "")
	lines = append(lines, renderFilterMatchLine(conversations, filter, contentWidth))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := renderFramedBox("Filter", boxWidth, colorPrimary, content)
	return lipgloss.Place(width, bodyHeight, lipgloss.Center, lipgloss.Center, box)
}

func renderFilterDimensionRow(filter browserFilterState, dim filterDimension, width int) string {
	cursor := filterOverlayCursorOff
	if filter.cursor == int(dim) && filter.expanded < 0 {
		cursor = filterOverlayCursorOn
	}

	label := filterDimensionLabel(dim)
	labelRendered := styleMetaLabel.Render(label)
	labelWidth := lipgloss.Width(labelRendered)

	summaryWidth := max(width-lipgloss.Width(filterOverlayIndent+cursor)-labelWidth-2, 1)
	summary := renderFilterDimensionSummary(filter, dim, summaryWidth)

	row := filterOverlayIndent + cursor + labelRendered + "  " + summary
	return ansi.Truncate(row, width, "…")
}

func renderFilterDimensionSummary(filter browserFilterState, dim filterDimension, maxWidth int) string {
	f := filter.dimensions[dim]

	if filterDimensionIsBool(dim) {
		return renderBoolSummary(f.boolState, maxWidth)
	}

	if filter.regexEditing && filter.cursor == int(dim) {
		return ansi.Truncate(filter.regexInput.View(), maxWidth, "")
	}

	if f.useRegex && f.regex != "" {
		text := lipgloss.NewStyle().Foreground(colorPrimary).Render("/" + f.regex + "/")
		return ansi.Truncate(text, maxWidth, "…")
	}

	return renderSelectionSummary(f, filter.values[dim], maxWidth)
}

func renderFilterExpandedValues(filter browserFilterState, dim filterDimension, width int) []string {
	values := filter.values[dim]
	f := filter.dimensions[dim]
	indent := filterOverlayIndent + "    "

	lines := make([]string, 0, len(values))
	for i, value := range values {
		cursor := filterOverlayCursorOff
		if filter.expandedCursor == i {
			cursor = filterOverlayCursorOn
		}

		check := filterOverlayCheckOff
		if f.selected[value] {
			check = lipgloss.NewStyle().Foreground(colorAccent).Render("✓ ")
		}

		row := indent + cursor + check + value
		lines = append(lines, ansi.Truncate(row, width, "…"))
	}
	return lines
}

func renderFilterMatchLine(
	conversations []conv.Conversation,
	filter browserFilterState,
	width int,
) string {
	count := filter.matchCount(conversations)
	total := len(conversations)
	text := fmt.Sprintf("%d of %d matching", count, total)
	rendered := lipgloss.NewStyle().Foreground(colorNormalDesc).Render(text)
	return filterOverlayIndent + ansi.Truncate(rendered, max(width-4, 1), "…")
}

func filterFooterStatusParts(
	conversations []conv.Conversation,
	filter browserFilterState,
) []string {
	count := filter.matchCount(conversations)
	total := len(conversations)
	return []string{fmt.Sprintf("%d/%d sessions", count, total)}
}

func filterFooterItems(filter browserFilterState) []helpItem {
	if filter.regexEditing {
		return []helpItem{
			{key: "enter", desc: "apply"},
			{key: "esc", desc: "cancel"},
		}
	}
	if filter.expanded >= 0 {
		return []helpItem{
			{key: "j/k", desc: "move"},
			{key: "space", desc: "toggle"},
			{key: "enter", desc: "done"},
			{key: "/", desc: "regex"},
			{key: "x", desc: "clear"},
			{key: "q/esc", desc: "back"},
		}
	}
	return filterDimensionFooterItems(filter)
}

func filterDimensionFooterItems(filter browserFilterState) []helpItem {
	dim := filterDimension(filter.cursor)
	items := []helpItem{{key: "j/k", desc: "move"}}

	if filterDimensionIsBool(dim) {
		items = append(items, helpItem{key: "space", desc: "toggle"})
	} else {
		items = append(items, helpItem{key: "enter", desc: "select"})
		items = append(items, helpItem{key: "/", desc: "regex"})
	}

	if filter.dimensions[dim].isActive() {
		items = append(items, helpItem{key: "x", desc: "clear"})
	}
	if filter.hasActiveFilters() {
		items = append(items, helpItem{key: "X", desc: "clear all"})
	}

	items = append(items, helpItem{key: "q/esc", desc: "close"})
	return items
}

func copyBrowserFilterState(filter browserFilterState) browserFilterState {
	cloned := filter
	cloned.active = false
	cloned.regexEditing = false
	cloned.expanded = -1
	cloned.regexInput.Blur()

	for i := range filterDimCount {
		if filter.dimensions[i].selected != nil {
			cloned.dimensions[i].selected = mapsClone(filter.dimensions[i].selected)
		}
		cloned.values[i] = slices.Clone(filter.values[i])
	}
	return cloned
}

func mapsClone[K comparable, V any](src map[K]V) map[K]V {
	if src == nil {
		return nil
	}

	dst := make(map[K]V, len(src))
	maps.Copy(dst, src)
	return dst
}
