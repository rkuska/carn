package elements

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	FilterOverlayIndent    = "  "
	FilterOverlayCursorOn  = "▸ "
	FilterOverlayCursorOff = "  "
	FilterOverlayCheckOff  = "  "
)

func RenderBoolSummary(state BoolFilterState, maxWidth int) string {
	var text string
	switch state {
	case BoolFilterYes:
		text = lipgloss.NewStyle().Foreground(ColorAccent).Render(BoolValueYes)
	case BoolFilterNo:
		text = lipgloss.NewStyle().Foreground(ColorDiffRemove).Render(BoolValueNo)
	case BoolFilterAny:
		text = lipgloss.NewStyle().Foreground(ColorNormalDesc).Render("─")
	}
	return ansi.Truncate(text, maxWidth, "…")
}

func RenderSelectionSummary(f DimensionFilter, values []string, maxWidth int) string {
	if len(f.Selected) == 0 {
		text := fmt.Sprintf("all (%d values)", len(values))
		return lipgloss.NewStyle().Foreground(ColorNormalDesc).Render(text)
	}

	if len(f.Selected) <= 3 {
		parts := make([]string, 0, len(values))
		for _, v := range values {
			if f.Selected[v] {
				parts = append(parts, lipgloss.NewStyle().Foreground(ColorAccent).Render(v+" ✓"))
			} else {
				parts = append(parts, lipgloss.NewStyle().Foreground(ColorNormalDesc).Render(v))
			}
		}
		text := strings.Join(parts, "  ")
		return ansi.Truncate(text, maxWidth, "…")
	}

	text := fmt.Sprintf("%d of %d selected", len(f.Selected), len(values))
	return lipgloss.NewStyle().Foreground(ColorAccent).Render(text)
}
