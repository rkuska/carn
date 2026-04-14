package app

import (
	"regexp"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

func (m statsModel) openFilterOverlay() statsModel {
	m.filter.active = true
	m.filter.regexEditing = false
	m.filter.regexInput.Blur()
	m.filter.expanded = -1
	if m.performanceScopeGateActive() {
		target := m.performanceScopeFilterDimension()
		m.filter.cursor = int(target)
		m.filter.expanded = int(target)
		m.filter.expandedCursor = 0
		m.filter.expandedScroll = 0
		return m
	}
	return m
}

func (m statsModel) closeFilterOverlay() statsModel {
	m.filter.active = false
	m.filter.expanded = -1
	m.filter.regexEditing = false
	m.filter.regexInput.Blur()
	return m
}

func (m statsModel) handleFilterKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if m.filter.regexEditing {
		return m.handleFilterRegexKey(msg)
	}
	if m.filter.expanded >= 0 {
		return m.handleFilterExpandedKey(msg)
	}
	return m.handleFilterDimensionKey(msg)
}

func (m statsModel) handleFilterDimensionKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if updated, cmd, handled := m.handleFilterNavigation(msg); handled {
		return updated, cmd
	}

	if m.filter.cursor == statsFilterVersionCursor {
		return m.handleVersionFilterDimensionAction(msg)
	}
	return m.handleFilterDimensionAction(msg, filterDimension(m.filter.cursor))
}

func (m statsModel) handleFilterNavigation(msg tea.KeyPressMsg) (statsModel, tea.Cmd, bool) {
	switch {
	case msg.Code == tea.KeyEscape || msg.Text == "q":
		return m.closeFilterOverlay(), nil, true
	case msg.Text == "j" || msg.Code == tea.KeyDown:
		if m.filter.cursor < statsFilterVersionCursor {
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

func (m statsModel) handleFilterDimensionAction(
	msg tea.KeyPressMsg,
	dim filterDimension,
) (statsModel, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEnter || msg.Text == "l" || msg.Code == tea.KeyRight:
		return m.filterExpandOrCycleBool(dim)
	case msg.Text == " ":
		return m.filterToggleBool(dim)
	case msg.Text == "x":
		m.filter.dimensions[dim] = dimensionFilter{}
		return m.applyFilterChangeAndMaybeLoad()
	case msg.Text == "X":
		return m.filterClearAll()
	case msg.Text == "/":
		return m.filterStartRegex(dim)
	}
	return m, nil
}

func (m statsModel) filterExpandOrCycleBool(dim filterDimension) (statsModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		m.filter.dimensions[dim] = dimensionFilter{
			boolState: cycleBoolFilter(m.filter.dimensions[dim].boolState),
		}
		return m.applyFilterChangeAndMaybeLoad()
	}
	m.filter.expanded = m.filter.cursor
	m.filter.expandedCursor = 0
	m.filter.expandedScroll = 0
	return m, nil
}

func (m statsModel) filterToggleBool(dim filterDimension) (statsModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		m.filter.dimensions[dim] = dimensionFilter{
			boolState: cycleBoolFilter(m.filter.dimensions[dim].boolState),
		}
		return m.applyFilterChangeAndMaybeLoad()
	}
	return m, nil
}

func (m statsModel) filterClearAll() (statsModel, tea.Cmd) {
	for i := range filterDimCount {
		m.filter.dimensions[i] = dimensionFilter{}
	}
	m.versionFilter = dimensionFilter{}
	m, cmd := m.applyFilterChangeAndMaybeLoad()
	return m.closeFilterOverlay(), cmd
}

func (m statsModel) filterStartRegex(dim filterDimension) (statsModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		return m, nil
	}
	m.filter.regexEditing = true
	m.filter.regexInput.SetValue(m.filter.dimensions[dim].regex)
	m.filter.regexInput.Focus()
	return m, textinput.Blink
}

func (m statsModel) handleFilterExpandedKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if m.filter.expanded == statsFilterVersionCursor {
		if updated, cmd, handled := m.handleFilterExpandedNav(msg, len(m.versionValues)); handled {
			return updated, cmd
		}
		return m.handleVersionFilterExpandedAction(msg)
	}

	dim := filterDimension(m.filter.expanded)
	values := m.filter.values[dim]
	if updated, cmd, handled := m.handleFilterExpandedNav(msg, len(values)); handled {
		return updated, cmd
	}
	return m.handleFilterExpandedAction(msg, dim, values)
}

func (m statsModel) handleFilterExpandedNav(msg tea.KeyPressMsg, count int) (statsModel, tea.Cmd, bool) {
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

func (m statsModel) handleFilterExpandedAction(
	msg tea.KeyPressMsg,
	dim filterDimension,
	values []string,
) (statsModel, tea.Cmd) {
	switch msg.Text {
	case " ":
		return m.filterToggleValue(dim, values)
	case "/":
		m.filter.regexEditing = true
		m.filter.expanded = -1
		m.filter.regexInput.SetValue(m.filter.dimensions[dim].regex)
		m.filter.regexInput.Focus()
		return m, textinput.Blink
	case "x":
		m.filter.dimensions[dim] = dimensionFilter{}
		m.filter.expanded = -1
		return m.applyFilterChangeAndMaybeLoad()
	}
	return m, nil
}

func (m statsModel) filterToggleValue(
	dim filterDimension,
	values []string,
) (statsModel, tea.Cmd) {
	if m.filter.expandedCursor >= len(values) {
		return m, nil
	}
	value := values[m.filter.expandedCursor]
	filter := m.filter.dimensions[dim]
	if filter.useRegex {
		filter = dimensionFilter{}
	}
	if filter.selected == nil {
		filter.selected = make(map[string]bool)
	}
	if filter.selected[value] {
		delete(filter.selected, value)
	} else {
		filter.selected[value] = true
	}
	filter.useRegex = false
	m.filter.dimensions[dim] = filter
	return m.applyFilterChangeAndMaybeLoad()
}

func (m statsModel) handleFilterRegexKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if m.filter.cursor == statsFilterVersionCursor {
		m.filter.regexEditing = false
		m.filter.regexInput.Blur()
		return m, nil
	}
	dim := filterDimension(m.filter.cursor)

	switch msg.Code {
	case tea.KeyEnter:
		return m.filterApplyRegex(dim)
	case tea.KeyEscape:
		m.filter.regexEditing = false
		m.filter.regexInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.filter.regexInput, cmd = m.filter.regexInput.Update(msg)
	return m, cmd
}

func (m statsModel) filterApplyRegex(dim filterDimension) (statsModel, tea.Cmd) {
	regex := m.filter.regexInput.Value()
	if regex != "" {
		re, err := regexp.Compile(regex)
		if err == nil {
			m.filter.dimensions[dim] = dimensionFilter{
				useRegex:   true,
				regex:      regex,
				compiledRe: re,
			}
			returned, cmd := m.applyFilterChangeAndMaybeLoad()
			m = returned
			m.filter.regexEditing = false
			m.filter.regexInput.Blur()
			return m, cmd
		}
	}
	m.filter.regexEditing = false
	m.filter.regexInput.Blur()
	return m, nil
}

func (m statsModel) handleVersionFilterDimensionAction(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEnter || msg.Text == "l" || msg.Code == tea.KeyRight:
		m.filter.expanded = statsFilterVersionCursor
		m.filter.expandedCursor = 0
		return m, nil
	case msg.Text == "x":
		m.versionFilter = dimensionFilter{}
		return m.applyFilterChangeAndMaybeLoad()
	case msg.Text == "X":
		return m.filterClearAll()
	default:
		return m, nil
	}
}

func (m statsModel) handleVersionFilterExpandedAction(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	switch msg.Text {
	case " ":
		return m.toggleVersionFilterValue()
	case "x":
		m.versionFilter = dimensionFilter{}
		m.filter.expanded = -1
		return m.applyFilterChangeAndMaybeLoad()
	default:
		return m, nil
	}
}

func (m statsModel) toggleVersionFilterValue() (statsModel, tea.Cmd) {
	if m.filter.expandedCursor >= len(m.versionValues) {
		return m, nil
	}
	value := m.versionValues[m.filter.expandedCursor]
	filter := m.versionFilter
	if filter.selected == nil {
		filter.selected = make(map[string]bool)
	}
	if filter.selected[value] {
		delete(filter.selected, value)
	} else {
		filter.selected[value] = true
	}
	m.versionFilter = filter
	return m.applyFilterChangeAndMaybeLoad()
}

func (m statsModel) handleStatsGroupAction() (statsModel, tea.Cmd, bool) {
	if !m.versionGroupingSupportedTab() {
		return m, nil, false
	}
	if m.versionGroupingActive() {
		m = m.setVersionGrouping(false)
		if !m.anyVersionGroupingActive() {
			m = m.clearGroupScope()
		}
		return m.renderViewportContent(true), nil, true
	}

	m = m.setVersionGrouping(true)
	m = m.seedSingleProviderGroupScope()
	if !m.groupScope.hasProvider() {
		return m.openGroupScopeOverlay(), nil, true
	}
	return m.renderViewportContent(true), nil, true
}
