package main

import "charm.land/lipgloss/v2"

var (
	// Colors
	colorPrimary    = lipgloss.Color("99")  // brighter purple (#875fff)
	colorSecondary  = lipgloss.Color("243") // neutral gray
	colorAccent     = lipgloss.Color("114") // soft green (#87d787)
	colorHighlight  = lipgloss.Color("53")  // dark purple (#5f005f) for filter match bg
	colorSelectedFg = lipgloss.Color("156") // light green (#afff87) for selected items

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
