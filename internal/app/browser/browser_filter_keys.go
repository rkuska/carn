package browser

import (
	"regexp"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

func (m browserModel) openFilterOverlay() browserModel {
	m.filter.Active = true
	m.filter.Expanded = -1
	m.filter.RegexEditing = false
	m.filter.RegexInput.Blur()
	return m
}

func (m browserModel) closeFilterOverlay() browserModel {
	m.filter.Active = false
	m.filter.Expanded = -1
	m.filter.RegexEditing = false
	m.filter.RegexInput.Blur()
	return m
}

func (m browserModel) handleFilterKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	if m.filter.RegexEditing {
		return m.handleFilterRegexKey(msg, cmds)
	}
	if m.filter.Expanded >= 0 {
		return m.handleFilterExpandedKey(msg, cmds)
	}
	return m.handleFilterDimensionKey(msg, cmds)
}

func (m browserModel) handleFilterDimensionKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	dim := filterDimension(m.filter.Cursor)

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
		if m.filter.Cursor < int(filterDimCount)-1 {
			m.filter.Cursor++
		}
		return m, nil, true
	case msg.Text == "k" || msg.Code == tea.KeyUp:
		if m.filter.Cursor > 0 {
			m.filter.Cursor--
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
		m.filter.Dimensions[dim] = dimensionFilter{}
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
		m.filter.Dimensions[dim] = dimensionFilter{
			BoolState: cycleBoolFilter(m.filter.Dimensions[dim].BoolState),
		}
		m = m.applyFilterChange(cmds)
		return m, nil
	}
	m.filter.Expanded = m.filter.Cursor
	m.filter.ExpandedCursor = 0
	m.filter.ExpandedScroll = 0
	return m, nil
}

func (m browserModel) filterToggleBool(dim filterDimension, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		m.filter.Dimensions[dim] = dimensionFilter{
			BoolState: cycleBoolFilter(m.filter.Dimensions[dim].BoolState),
		}
		m = m.applyFilterChange(cmds)
	}
	return m, nil
}

func (m browserModel) filterClearAll(cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	for i := range filterDimCount {
		m.filter.Dimensions[i] = dimensionFilter{}
	}
	m = m.applyFilterChange(cmds)
	return m.closeFilterOverlay(), nil
}

func (m browserModel) filterStartRegex(dim filterDimension) (browserModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		return m, nil
	}
	m.filter.RegexEditing = true
	m.filter.RegexInput.SetValue(m.filter.Dimensions[dim].Regex)
	m.filter.RegexInput.Focus()
	return m, textinput.Blink
}

func (m browserModel) handleFilterExpandedKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	dim := filterDimension(m.filter.Expanded)
	values := m.filter.Values[dim]

	if updated, cmd, handled := m.handleFilterExpandedNav(msg, len(values)); handled {
		return updated, cmd
	}
	return m.handleFilterExpandedAction(msg, dim, values, cmds)
}

func (m browserModel) handleFilterExpandedNav(msg tea.KeyPressMsg, count int) (browserModel, tea.Cmd, bool) {
	if isFilterCollapseKey(msg) {
		m.filter.Expanded = -1
		return m, nil, true
	}
	switch {
	case msg.Text == "j" || msg.Code == tea.KeyDown:
		if m.filter.ExpandedCursor < count-1 {
			m.filter.ExpandedCursor++
		}
		return m, nil, true
	case msg.Text == "k" || msg.Code == tea.KeyUp:
		if m.filter.ExpandedCursor > 0 {
			m.filter.ExpandedCursor--
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
		m.filter.RegexEditing = true
		m.filter.Expanded = -1
		m.filter.RegexInput.SetValue(m.filter.Dimensions[dim].Regex)
		m.filter.RegexInput.Focus()
		return m, textinput.Blink
	case "x":
		m.filter.Dimensions[dim] = dimensionFilter{}
		m.filter.Expanded = -1
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
	if m.filter.ExpandedCursor >= len(values) {
		return m, nil
	}
	value := values[m.filter.ExpandedCursor]
	f := m.filter.Dimensions[dim]
	if f.UseRegex {
		f = dimensionFilter{}
	}
	if f.Selected == nil {
		f.Selected = make(map[string]bool)
	}
	if f.Selected[value] {
		delete(f.Selected, value)
	} else {
		f.Selected[value] = true
	}
	f.UseRegex = false
	m.filter.Dimensions[dim] = f
	m = m.applyFilterChange(cmds)
	return m, nil
}

func (m browserModel) handleFilterRegexKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	dim := filterDimension(m.filter.Cursor)

	switch msg.Code {
	case tea.KeyEnter:
		return m.filterApplyRegex(dim, cmds)
	case tea.KeyEscape:
		m.filter.RegexEditing = false
		m.filter.RegexInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.filter.RegexInput, cmd = m.filter.RegexInput.Update(msg)
	return m, cmd
}

func (m browserModel) filterApplyRegex(dim filterDimension, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	regex := m.filter.RegexInput.Value()
	if regex != "" {
		re, err := regexp.Compile(regex)
		if err != nil {
			m = m.setNotification(
				errorNotification("invalid regex: "+err.Error()).Notification,
				cmds,
			)
			return m, nil
		}
		m.filter.Dimensions[dim] = dimensionFilter{
			UseRegex:   true,
			Regex:      regex,
			CompiledRe: re,
		}
		m = m.applyFilterChange(cmds)
	}
	m.filter.RegexEditing = false
	m.filter.RegexInput.Blur()
	return m, nil
}
