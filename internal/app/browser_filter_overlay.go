package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	filterOverlayIndent    = "  "
	filterOverlayCursorOn  = "▸ "
	filterOverlayCursorOff = "  "
	filterOverlayCheckOff  = "  "
)

func (m browserModel) renderFilterOverlay() string {
	boxWidth := min(max(m.width-8, 40), 96)
	bodyHeight := max(m.height-framedFooterRows, 1)
	contentWidth := max(boxWidth-2, 1)

	lines := []string{""}
	for i := range filterDimCount {
		dim := filterDimension(i)
		lines = append(lines, m.renderFilterDimensionRow(dim, contentWidth))
		if m.filter.expanded == int(dim) {
			lines = append(lines, m.renderFilterExpandedValues(dim, contentWidth)...)
		}
	}

	lines = append(lines, "")
	lines = append(lines, m.renderFilterMatchLine(contentWidth))
	lines = append(lines, m.renderFilterHintLine(contentWidth))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := renderFramedBox("Filter", boxWidth, colorPrimary, content)
	return lipgloss.Place(m.width, bodyHeight, lipgloss.Center, lipgloss.Center, box)
}

func (m browserModel) renderFilterDimensionRow(dim filterDimension, width int) string {
	cursor := filterOverlayCursorOff
	if m.filter.cursor == int(dim) && m.filter.expanded < 0 {
		cursor = filterOverlayCursorOn
	}

	label := filterDimensionLabel(dim)
	labelRendered := styleMetaLabel.Render(label)
	labelWidth := lipgloss.Width(labelRendered)

	summaryWidth := max(width-lipgloss.Width(filterOverlayIndent+cursor)-labelWidth-2, 1)
	summary := m.renderFilterDimensionSummary(dim, summaryWidth)

	row := filterOverlayIndent + cursor + labelRendered + "  " + summary
	return ansi.Truncate(row, width, "…")
}

func (m browserModel) renderFilterDimensionSummary(dim filterDimension, maxWidth int) string {
	f := m.filter.dimensions[dim]

	if filterDimensionIsBool(dim) {
		return renderBoolSummary(f.boolState, maxWidth)
	}

	if m.filter.regexEditing && m.filter.cursor == int(dim) {
		return ansi.Truncate(m.filter.regexInput.View(), maxWidth, "")
	}

	if f.useRegex && f.regex != "" {
		text := lipgloss.NewStyle().Foreground(colorPrimary).Render("/" + f.regex + "/")
		return ansi.Truncate(text, maxWidth, "…")
	}

	return renderSelectionSummary(f, m.filter.values[dim], maxWidth)
}

func renderBoolSummary(state boolFilterState, maxWidth int) string {
	var text string
	switch state {
	case boolFilterYes:
		text = lipgloss.NewStyle().Foreground(colorAccent).Render(boolValueYes)
	case boolFilterNo:
		text = lipgloss.NewStyle().Foreground(colorDiffRemove).Render(boolValueNo)
	case boolFilterAny:
		text = lipgloss.NewStyle().Foreground(colorNormalDesc).Render("─")
	}
	return ansi.Truncate(text, maxWidth, "…")
}

func renderSelectionSummary(f dimensionFilter, values []string, maxWidth int) string {
	if len(f.selected) == 0 {
		text := fmt.Sprintf("all (%d values)", len(values))
		return lipgloss.NewStyle().Foreground(colorNormalDesc).Render(text)
	}

	if len(f.selected) <= 3 {
		parts := make([]string, 0, len(values))
		for _, v := range values {
			if f.selected[v] {
				parts = append(parts, lipgloss.NewStyle().Foreground(colorAccent).Render(v+" ✓"))
			} else {
				parts = append(parts, lipgloss.NewStyle().Foreground(colorNormalDesc).Render(v))
			}
		}
		text := strings.Join(parts, "  ")
		return ansi.Truncate(text, maxWidth, "…")
	}

	text := fmt.Sprintf("%d of %d selected", len(f.selected), len(values))
	return lipgloss.NewStyle().Foreground(colorAccent).Render(text)
}

func (m browserModel) renderFilterExpandedValues(dim filterDimension, width int) []string {
	values := m.filter.values[dim]
	f := m.filter.dimensions[dim]
	indent := filterOverlayIndent + "    "

	lines := make([]string, 0, len(values))
	for i, v := range values {
		cursor := filterOverlayCursorOff
		if m.filter.expandedCursor == i {
			cursor = filterOverlayCursorOn
		}

		check := filterOverlayCheckOff
		if f.selected[v] {
			check = lipgloss.NewStyle().Foreground(colorAccent).Render("✓ ")
		}

		row := indent + cursor + check + v
		row = ansi.Truncate(row, width, "…")
		lines = append(lines, row)
	}
	return lines
}

func (m browserModel) renderFilterMatchLine(width int) string {
	count := m.filter.matchCount(m.mainConversations)
	total := len(m.mainConversations)
	text := fmt.Sprintf("%d of %d matching", count, total)
	rendered := lipgloss.NewStyle().Foreground(colorNormalDesc).Render(text)
	return filterOverlayIndent + ansi.Truncate(rendered, max(width-4, 1), "…")
}

func (m browserModel) renderFilterHintLine(width int) string {
	if m.filter.regexEditing {
		return filterOverlayIndent + renderFilterHints([]string{
			"enter apply", "esc cancel",
		}, width-4)
	}
	if m.filter.expanded >= 0 {
		return filterOverlayIndent + renderFilterHints([]string{
			"space toggle", "enter done", "/ regex", "x clear", "esc back",
		}, width-4)
	}
	return filterOverlayIndent + renderFilterHints([]string{
		"enter select", "space toggle", "/ regex", "x clear", "X clear all", "esc close",
	}, width-4)
}

func renderFilterHints(hints []string, maxWidth int) string {
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, lipgloss.NewStyle().Foreground(colorNormalDesc).Render(h))
	}
	text := strings.Join(parts, "  ")
	return ansi.Truncate(text, max(maxWidth, 1), "…")
}

func (m browserModel) filterFooterItems() []helpItem {
	return []helpItem{
		{key: "j/k", desc: "move"},
		{key: "space", desc: "toggle"},
		{key: "/", desc: "regex"},
		{key: "x", desc: "clear"},
		{key: "esc", desc: "close filter"},
	}
}
