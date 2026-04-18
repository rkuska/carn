package stats

import (
	tea "charm.land/bubbletea/v2"

	statspkg "github.com/rkuska/carn/internal/stats"
)

const splitRowCursor = -1

func (m statsModel) handleSplitRowAction(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEnter || msg.Text == "l" || msg.Text == " " || msg.Code == tea.KeyRight:
		m.splitExpanded = true
		m.splitExpandedCursor = currentSplitOptionIndex(m.splitBy)
		return m, nil
	case msg.Text == "x":
		m.splitBy = statspkg.SplitDimensionNone
		return m.applyFilterChangeAndMaybeLoad()
	case msg.Text == "X":
		return m.filterClearAll()
	}
	return m, nil
}

func (m statsModel) handleSplitExpandedKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if isFilterCollapseKey(msg) {
		m.splitExpanded = false
		return m, nil
	}
	switch {
	case msg.Text == "j" || msg.Code == tea.KeyDown:
		if m.splitExpandedCursor < len(splitDimensionOptions)-1 {
			m.splitExpandedCursor++
		}
		return m, nil
	case msg.Text == "k" || msg.Code == tea.KeyUp:
		if m.splitExpandedCursor > 0 {
			m.splitExpandedCursor--
		}
		return m, nil
	case msg.Text == " ":
		return m.toggleSplitOption()
	case msg.Text == "x":
		m.splitBy = statspkg.SplitDimensionNone
		m.splitExpanded = false
		return m.applyFilterChangeAndMaybeLoad()
	}
	return m, nil
}

func (m statsModel) toggleSplitOption() (statsModel, tea.Cmd) {
	if m.splitExpandedCursor < 0 || m.splitExpandedCursor >= len(splitDimensionOptions) {
		return m, nil
	}
	choice := splitDimensionOptions[m.splitExpandedCursor]
	if m.splitBy == choice {
		m.splitBy = statspkg.SplitDimensionNone
	} else {
		m.splitBy = choice
	}
	return m.applyFilterChangeAndMaybeLoad()
}

func currentSplitOptionIndex(dim statspkg.SplitDimension) int {
	for i, option := range splitDimensionOptions {
		if option == dim {
			return i
		}
	}
	return 0
}
