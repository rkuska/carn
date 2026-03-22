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
	return renderFilterOverlayWithConversations(m.mainConversations, m.filter, m.width, m.height)
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

func (m browserModel) filterFooterStatusParts() []string {
	return filterFooterStatusParts(m.mainConversations, m.filter)
}

func (m browserModel) filterFooterItems() []helpItem {
	return filterFooterItems(m.filter)
}
