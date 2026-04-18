package stats

import (
	"regexp"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) openFilterOverlay() statsModel {
	m.filter.Active = true
	m.filter.RegexEditing = false
	m.filter.RegexInput.Blur()
	m.filter.Expanded = -1
	m.splitExpanded = false
	m.splitExpandedCursor = 0
	if m.performanceScopeGateActive() {
		target := m.performanceScopeFilterDimension()
		m.filter.Cursor = int(target)
		m.filter.Expanded = int(target)
		m.filter.ExpandedCursor = 0
		m.filter.ExpandedScroll = 0
		return m
	}
	return m
}

func (m statsModel) closeFilterOverlay() statsModel {
	m.filter.Active = false
	m.filter.Expanded = -1
	m.filter.RegexEditing = false
	m.filter.RegexInput.Blur()
	m.splitExpanded = false
	m.splitExpandedCursor = 0
	return m
}

func (m statsModel) handleFilterKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if m.filter.RegexEditing {
		return m.handleFilterRegexKey(msg)
	}
	if m.splitExpanded {
		return m.handleSplitExpandedKey(msg)
	}
	if m.filter.Expanded >= 0 {
		return m.handleFilterExpandedKey(msg)
	}
	return m.handleFilterDimensionKey(msg)
}

func (m statsModel) handleFilterDimensionKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if updated, cmd, handled := m.handleFilterNavigation(msg); handled {
		return updated, cmd
	}
	if m.filter.Cursor == splitRowCursor {
		return m.handleSplitRowAction(msg)
	}
	return m.handleFilterDimensionAction(msg, filterDimension(m.filter.Cursor))
}

func (m statsModel) handleFilterNavigation(msg tea.KeyPressMsg) (statsModel, tea.Cmd, bool) {
	switch {
	case msg.Code == tea.KeyEscape || msg.Text == "q":
		return m.closeFilterOverlay(), nil, true
	case msg.Text == "j" || msg.Code == tea.KeyDown:
		if m.filter.Cursor < int(filterDimCount)-1 {
			m.filter.Cursor++
		}
		return m, nil, true
	case msg.Text == "k" || msg.Code == tea.KeyUp:
		if m.filter.Cursor > splitRowCursor {
			m.filter.Cursor--
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
		m.filter.Dimensions[dim] = dimensionFilter{}
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
		m.filter.Dimensions[dim] = dimensionFilter{
			BoolState: cycleBoolFilter(m.filter.Dimensions[dim].BoolState),
		}
		return m.applyFilterChangeAndMaybeLoad()
	}
	m.filter.Expanded = m.filter.Cursor
	m.filter.ExpandedCursor = 0
	m.filter.ExpandedScroll = 0
	return m, nil
}

func (m statsModel) filterToggleBool(dim filterDimension) (statsModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		m.filter.Dimensions[dim] = dimensionFilter{
			BoolState: cycleBoolFilter(m.filter.Dimensions[dim].BoolState),
		}
		return m.applyFilterChangeAndMaybeLoad()
	}
	return m, nil
}

func (m statsModel) filterClearAll() (statsModel, tea.Cmd) {
	for i := range filterDimCount {
		m.filter.Dimensions[i] = dimensionFilter{}
	}
	m.splitBy = statspkg.SplitDimensionNone
	m, cmd := m.applyFilterChangeAndMaybeLoad()
	return m.closeFilterOverlay(), cmd
}

func (m statsModel) filterStartRegex(dim filterDimension) (statsModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		return m, nil
	}
	m.filter.RegexEditing = true
	m.filter.RegexInput.SetValue(m.filter.Dimensions[dim].Regex)
	m.filter.RegexInput.Focus()
	return m, textinput.Blink
}

func (m statsModel) handleFilterExpandedKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	dim := filterDimension(m.filter.Expanded)
	values := m.filter.Values[dim]
	if updated, cmd, handled := m.handleFilterExpandedNav(msg, len(values)); handled {
		return updated, cmd
	}
	return m.handleFilterExpandedAction(msg, dim, values)
}

func (m statsModel) handleFilterExpandedNav(msg tea.KeyPressMsg, count int) (statsModel, tea.Cmd, bool) {
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

func (m statsModel) handleFilterExpandedAction(
	msg tea.KeyPressMsg,
	dim filterDimension,
	values []string,
) (statsModel, tea.Cmd) {
	switch msg.Text {
	case " ":
		return m.filterToggleValue(dim, values)
	case "/":
		m.filter.RegexEditing = true
		m.filter.Expanded = -1
		m.filter.RegexInput.SetValue(m.filter.Dimensions[dim].Regex)
		m.filter.RegexInput.Focus()
		return m, textinput.Blink
	case "x":
		m.filter.Dimensions[dim] = dimensionFilter{}
		m.filter.Expanded = -1
		return m.applyFilterChangeAndMaybeLoad()
	}
	return m, nil
}

func (m statsModel) filterToggleValue(
	dim filterDimension,
	values []string,
) (statsModel, tea.Cmd) {
	if m.filter.ExpandedCursor >= len(values) {
		return m, nil
	}
	value := values[m.filter.ExpandedCursor]
	filter := m.filter.Dimensions[dim]
	if filter.UseRegex {
		filter = dimensionFilter{}
	}
	if filter.Selected == nil {
		filter.Selected = make(map[string]bool)
	}
	if filter.Selected[value] {
		delete(filter.Selected, value)
	} else {
		filter.Selected[value] = true
	}
	filter.UseRegex = false
	m.filter.Dimensions[dim] = filter
	return m.applyFilterChangeAndMaybeLoad()
}

func (m statsModel) handleFilterRegexKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if m.filter.Cursor == splitRowCursor {
		m.filter.RegexEditing = false
		m.filter.RegexInput.Blur()
		return m, nil
	}
	dim := filterDimension(m.filter.Cursor)

	switch msg.Code {
	case tea.KeyEnter:
		return m.filterApplyRegex(dim)
	case tea.KeyEscape:
		m.filter.RegexEditing = false
		m.filter.RegexInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.filter.RegexInput, cmd = m.filter.RegexInput.Update(msg)
	return m, cmd
}

func (m statsModel) filterApplyRegex(dim filterDimension) (statsModel, tea.Cmd) {
	regex := m.filter.RegexInput.Value()
	if regex != "" {
		re, err := regexp.Compile(regex)
		if err == nil {
			m.filter.Dimensions[dim] = dimensionFilter{
				UseRegex:   true,
				Regex:      regex,
				CompiledRe: re,
			}
			returned, cmd := m.applyFilterChangeAndMaybeLoad()
			m = returned
			m.filter.RegexEditing = false
			m.filter.RegexInput.Blur()
			return m, cmd
		}
	}
	m.filter.RegexEditing = false
	m.filter.RegexInput.Blur()
	return m, nil
}
