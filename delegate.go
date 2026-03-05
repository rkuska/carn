package main

import (
	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
)

func newDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = true
	d.SetSpacing(1)
	d.SetHeight(3)

	d.Styles.FilterMatch = lipgloss.NewStyle().
		Background(colorHighlight).
		Bold(true)

	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(colorAccent).
		Foreground(colorSelectedFg).
		Padding(0, 0, 0, 1)

	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(colorAccent).
		Foreground(colorSecondary).
		Padding(0, 0, 0, 1)

	return d
}
