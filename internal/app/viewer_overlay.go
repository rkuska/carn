package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	conv "github.com/rkuska/carn/internal/conversation"
)

func (m viewerModel) paneContent() string {
	if m.planPicker.active {
		return m.renderPlanPickerOverlay()
	}
	return highlightViewportMatches(
		m.viewport.View(),
		m.searchQuery,
		m.matches,
		m.currentMatch,
		m.viewport.YOffset(),
	)
}

func (m viewerModel) renderPlanPickerOverlay() string {
	plans := conv.AllPlans(m.session.Messages)
	width := max(m.contentWidth()-8, 24)

	lines := []string{styleSubtitle.Render(fmt.Sprintf("Choose a plan to %s", m.planPicker.action.String())), ""}
	for i, plan := range plans {
		cursor := "  "
		if i == m.planPicker.selected {
			cursor = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("> ")
		}
		lines = append(lines, cursor+planFileName(plan))
	}

	box := renderInsetBox(width, colorPrimary, strings.Join(lines, "\n"))
	return lipgloss.Place(
		m.viewportWidth(),
		framedBodyHeight(m.height),
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}
