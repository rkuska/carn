package main

import "charm.land/lipgloss/v2"

var (
	// Colors
	colorPrimary   = lipgloss.Color("62")  // purple
	colorSecondary = lipgloss.Color("241") // gray
	colorAccent    = lipgloss.Color("220") // yellow

	// Layout
	stylePreviewBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary)

	// Text
	styleTitle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	styleToolCall = lipgloss.NewStyle().
			Foreground(colorAccent)

	styleStatusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)
)
