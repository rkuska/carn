package elements

import "charm.land/lipgloss/v2"

// applyContextStyles builds the context growth styles once so render paths never
// construct styles per call.
func (t *Theme) applyContextStyles() {
	t.StyleContextFill = lipgloss.NewStyle().Foreground(t.ColorContextFill)
	t.StyleContextPressure = lipgloss.NewStyle().Foreground(t.ColorContextPressure)
	t.StyleContextPressureBold = lipgloss.NewStyle().Foreground(t.ColorContextPressure).Bold(true)
	t.StyleContextWell = lipgloss.NewStyle().Background(t.ColorContextWell)
	t.StyleContextWellFill = lipgloss.NewStyle().Foreground(t.ColorContextFill).Background(t.ColorContextWell)
	t.StyleContextWellPressure = lipgloss.NewStyle().Foreground(t.ColorContextPressure).Background(t.ColorContextWell)
	t.StyleContextCursor = lipgloss.NewStyle().Background(t.ColorContextCursor)
	t.StyleContextCursorFill = lipgloss.NewStyle().Foreground(t.ColorContextFill).Background(t.ColorContextCursor)
	t.StyleContextCursorPressure = lipgloss.NewStyle().
		Foreground(t.ColorContextPressure).
		Background(t.ColorContextCursor)
}
