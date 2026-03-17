package app

import (
	"regexp"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

func (m browserModel) openFilterOverlay() browserModel {
	m.filter.active = true
	m.filter.expanded = -1
	m.filter.regexEditing = false
	m.filter.regexInput.Blur()
	return m
}

func (m browserModel) closeFilterOverlay() browserModel {
	m.filter.active = false
	m.filter.expanded = -1
	m.filter.regexEditing = false
	m.filter.regexInput.Blur()
	return m
}

func (m browserModel) handleFilterKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	if m.filter.regexEditing {
		return m.handleFilterRegexKey(msg, cmds)
	}
	if m.filter.expanded >= 0 {
		return m.handleFilterExpandedKey(msg, cmds)
	}
	return m.handleFilterDimensionKey(msg, cmds)
}

func (m browserModel) handleFilterDimensionKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	dim := filterDimension(m.filter.cursor)

	if updated, cmd, handled := m.handleFilterNavigation(msg); handled {
		return updated, cmd
	}

	return m.handleFilterDimensionAction(msg, dim, cmds)
}

func (m browserModel) handleFilterNavigation(msg tea.KeyPressMsg) (browserModel, tea.Cmd, bool) {
	switch {
	case msg.Code == tea.KeyEscape || msg.Text == "q":
		return m.closeFilterOverlay(), nil, true
	case msg.Text == "j" || msg.Code == tea.KeyDown:
		if m.filter.cursor < int(filterDimCount)-1 {
			m.filter.cursor++
		}
		return m, nil, true
	case msg.Text == "k" || msg.Code == tea.KeyUp:
		if m.filter.cursor > 0 {
			m.filter.cursor--
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m browserModel) handleFilterDimensionAction(
	msg tea.KeyPressMsg,
	dim filterDimension,
	cmds *[]tea.Cmd,
) (browserModel, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEnter || msg.Text == "l" || msg.Code == tea.KeyRight:
		return m.filterExpandOrCycleBool(dim, cmds)
	case msg.Text == " ":
		return m.filterToggleBool(dim, cmds)
	case msg.Text == "x":
		m.filter.dimensions[dim] = dimensionFilter{}
		m = m.applyFilterChange(cmds)
		return m, nil
	case msg.Text == "X":
		return m.filterClearAll(cmds)
	case msg.Text == "/":
		return m.filterStartRegex(dim)
	}
	return m, nil
}

func (m browserModel) filterExpandOrCycleBool(dim filterDimension, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		m.filter.dimensions[dim] = dimensionFilter{
			boolState: cycleBoolFilter(m.filter.dimensions[dim].boolState),
		}
		m = m.applyFilterChange(cmds)
		return m, nil
	}
	m.filter.expanded = m.filter.cursor
	m.filter.expandedCursor = 0
	m.filter.expandedScroll = 0
	return m, nil
}

func (m browserModel) filterToggleBool(dim filterDimension, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		m.filter.dimensions[dim] = dimensionFilter{
			boolState: cycleBoolFilter(m.filter.dimensions[dim].boolState),
		}
		m = m.applyFilterChange(cmds)
	}
	return m, nil
}

func (m browserModel) filterClearAll(cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	for i := range filterDimCount {
		m.filter.dimensions[i] = dimensionFilter{}
	}
	m = m.applyFilterChange(cmds)
	return m.closeFilterOverlay(), nil
}

func (m browserModel) filterStartRegex(dim filterDimension) (browserModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		return m, nil
	}
	m.filter.regexEditing = true
	m.filter.regexInput.SetValue(m.filter.dimensions[dim].regex)
	m.filter.regexInput.Focus()
	return m, textinput.Blink
}

func (m browserModel) handleFilterExpandedKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	dim := filterDimension(m.filter.expanded)
	values := m.filter.values[dim]

	if updated, cmd, handled := m.handleFilterExpandedNav(msg, len(values)); handled {
		return updated, cmd
	}
	return m.handleFilterExpandedAction(msg, dim, values, cmds)
}

func (m browserModel) handleFilterExpandedNav(msg tea.KeyPressMsg, count int) (browserModel, tea.Cmd, bool) {
	if isFilterCollapseKey(msg) {
		m.filter.expanded = -1
		return m, nil, true
	}
	switch {
	case msg.Text == "j" || msg.Code == tea.KeyDown:
		if m.filter.expandedCursor < count-1 {
			m.filter.expandedCursor++
		}
		return m, nil, true
	case msg.Text == "k" || msg.Code == tea.KeyUp:
		if m.filter.expandedCursor > 0 {
			m.filter.expandedCursor--
		}
		return m, nil, true
	}
	return m, nil, false
}

func isFilterCollapseKey(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyEscape || msg.Code == tea.KeyEnter ||
		msg.Text == "h" || msg.Code == tea.KeyLeft || msg.Text == "q"
}

func (m browserModel) handleFilterExpandedAction(
	msg tea.KeyPressMsg,
	dim filterDimension,
	values []string,
	cmds *[]tea.Cmd,
) (browserModel, tea.Cmd) {
	switch msg.Text {
	case " ":
		return m.filterToggleValue(dim, values, cmds)
	case "/":
		m.filter.regexEditing = true
		m.filter.expanded = -1
		m.filter.regexInput.SetValue(m.filter.dimensions[dim].regex)
		m.filter.regexInput.Focus()
		return m, textinput.Blink
	case "x":
		m.filter.dimensions[dim] = dimensionFilter{}
		m.filter.expanded = -1
		m = m.applyFilterChange(cmds)
		return m, nil
	}
	return m, nil
}

func (m browserModel) filterToggleValue(
	dim filterDimension,
	values []string,
	cmds *[]tea.Cmd,
) (browserModel, tea.Cmd) {
	if m.filter.expandedCursor >= len(values) {
		return m, nil
	}
	value := values[m.filter.expandedCursor]
	f := m.filter.dimensions[dim]
	if f.useRegex {
		f = dimensionFilter{}
	}
	if f.selected == nil {
		f.selected = make(map[string]bool)
	}
	if f.selected[value] {
		delete(f.selected, value)
	} else {
		f.selected[value] = true
	}
	f.useRegex = false
	m.filter.dimensions[dim] = f
	m = m.applyFilterChange(cmds)
	return m, nil
}

func (m browserModel) handleFilterRegexKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	dim := filterDimension(m.filter.cursor)

	switch msg.Code {
	case tea.KeyEnter:
		return m.filterApplyRegex(dim, cmds)
	case tea.KeyEscape:
		m.filter.regexEditing = false
		m.filter.regexInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.filter.regexInput, cmd = m.filter.regexInput.Update(msg)
	return m, cmd
}

func (m browserModel) filterApplyRegex(dim filterDimension, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	regex := m.filter.regexInput.Value()
	if regex != "" {
		re, err := regexp.Compile(regex)
		if err != nil {
			m = m.setNotification(
				errorNotification("invalid regex: "+err.Error()).notification,
				cmds,
			)
			return m, nil
		}
		m.filter.dimensions[dim] = dimensionFilter{
			useRegex:   true,
			regex:      regex,
			compiledRe: re,
		}
		m = m.applyFilterChange(cmds)
	}
	m.filter.regexEditing = false
	m.filter.regexInput.Blur()
	return m, nil
}
