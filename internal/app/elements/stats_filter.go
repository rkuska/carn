package elements

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	conv "github.com/rkuska/carn/internal/conversation"
)

func (t *Theme) RenderFilterOverlayWithConversations(
	conversations []conv.Conversation,
	filter FilterState,
	width, height int,
) string {
	boxWidth := min(max(width-8, 40), 96)
	bodyHeight := max(height-FramedFooterRows, 1)
	contentWidth := max(boxWidth-2, 1)

	lines := []string{""}
	for i := range FilterDimCount {
		dim := FilterDimension(i)
		lines = append(lines, t.RenderFilterDimensionRow(filter, dim, contentWidth))
		if filter.Expanded == int(dim) {
			lines = append(lines, t.RenderFilterExpandedValues(filter, dim, contentWidth)...)
		}
	}

	lines = append(lines, "")
	lines = append(lines, t.renderFilterMatchLine(conversations, filter, contentWidth))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := t.RenderFramedBox("Filter", boxWidth, t.ColorPrimary, content)
	return lipgloss.Place(width, bodyHeight, lipgloss.Center, lipgloss.Center, box)
}

func (t *Theme) RenderFilterDimensionRow(filter FilterState, dim FilterDimension, width int) string {
	cursor := FilterOverlayCursorOff
	if filter.Cursor == int(dim) && filter.Expanded < 0 {
		cursor = FilterOverlayCursorOn
	}

	label := FilterDimensionLabel(dim)
	labelRendered := t.StyleMetaLabel.Render(label)
	labelWidth := lipgloss.Width(labelRendered)

	summaryWidth := max(width-lipgloss.Width(FilterOverlayIndent+cursor)-labelWidth-2, 1)
	summary := t.RenderFilterDimensionSummary(filter, dim, summaryWidth)

	row := FilterOverlayIndent + cursor + labelRendered + "  " + summary
	return ansi.Truncate(row, width, "…")
}

func (t *Theme) RenderFilterDimensionSummary(filter FilterState, dim FilterDimension, maxWidth int) string {
	f := filter.Dimensions[dim]

	if FilterDimensionIsBool(dim) {
		return t.RenderBoolSummary(f.BoolState, maxWidth)
	}

	if filter.RegexEditing && filter.Cursor == int(dim) {
		return ansi.Truncate(filter.RegexInput.View(), maxWidth, "")
	}

	if f.UseRegex && f.Regex != "" {
		text := lipgloss.NewStyle().Foreground(t.ColorPrimary).Render("/" + f.Regex + "/")
		return ansi.Truncate(text, maxWidth, "…")
	}

	return t.RenderSelectionSummary(f, filter.Values[dim], maxWidth)
}

func (t *Theme) RenderFilterExpandedValues(filter FilterState, dim FilterDimension, width int) []string {
	values := filter.Values[dim]
	f := filter.Dimensions[dim]
	indent := FilterOverlayIndent + "    "

	lines := make([]string, 0, len(values))
	for i, value := range values {
		cursor := FilterOverlayCursorOff
		if filter.ExpandedCursor == i {
			cursor = FilterOverlayCursorOn
		}

		check := FilterOverlayCheckOff
		if f.Selected[value] {
			check = lipgloss.NewStyle().Foreground(t.ColorAccent).Render("✓ ")
		}

		row := indent + cursor + check + value
		lines = append(lines, ansi.Truncate(row, width, "…"))
	}
	return lines
}

func (t *Theme) renderFilterMatchLine(
	conversations []conv.Conversation,
	filter FilterState,
	width int,
) string {
	count := filter.MatchCount(conversations)
	total := len(conversations)
	text := fmt.Sprintf("%d of %d matching", count, total)
	rendered := lipgloss.NewStyle().Foreground(t.ColorNormalDesc).Render(text)
	return FilterOverlayIndent + ansi.Truncate(rendered, max(width-4, 1), "…")
}

func FilterFooterStatusParts(
	conversations []conv.Conversation,
	filter FilterState,
) []string {
	count := filter.MatchCount(conversations)
	total := len(conversations)
	return []string{fmt.Sprintf("%d/%d sessions", count, total)}
}

func FilterFooterItems(filter FilterState) []HelpItem {
	if filter.RegexEditing {
		return []HelpItem{
			{Key: "enter", Desc: "apply"},
			{Key: "esc", Desc: "cancel"},
		}
	}
	if filter.Expanded >= 0 {
		return []HelpItem{
			{Key: "j/k", Desc: "move"},
			{Key: "space", Desc: "toggle"},
			{Key: "enter", Desc: "done"},
			{Key: "/", Desc: "regex"},
			{Key: "x", Desc: "clear"},
			{Key: "q/esc", Desc: "back"},
		}
	}
	return FilterDimensionFooterItems(filter)
}

func FilterDimensionFooterItems(filter FilterState) []HelpItem {
	dim := FilterDimension(filter.Cursor)
	items := []HelpItem{{Key: "j/k", Desc: "move"}}

	if FilterDimensionIsBool(dim) {
		items = append(items, HelpItem{Key: "space", Desc: "toggle"})
	} else {
		items = append(items, HelpItem{Key: "enter", Desc: "select"})
		items = append(items, HelpItem{Key: "/", Desc: "regex"})
	}

	if filter.Dimensions[dim].IsActive() {
		items = append(items, HelpItem{Key: "x", Desc: "clear"})
	}
	if filter.HasActiveFilters() {
		items = append(items, HelpItem{Key: "X", Desc: "clear all"})
	}

	items = append(items, HelpItem{Key: "q/esc", Desc: "close"})
	return items
}

func CopyFilterState(filter FilterState) FilterState {
	cloned := filter
	cloned.Active = false
	cloned.RegexEditing = false
	cloned.Expanded = -1
	cloned.RegexInput.Blur()

	for i := range FilterDimCount {
		if filter.Dimensions[i].Selected != nil {
			cloned.Dimensions[i].Selected = mapsClone(filter.Dimensions[i].Selected)
		}
		cloned.Values[i] = slices.Clone(filter.Values[i])
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
