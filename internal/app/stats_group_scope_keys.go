package app

import tea "charm.land/bubbletea/v2"

func (m statsModel) openGroupScopeOverlay() statsModel {
	m.groupScope.active = true
	m.groupScope.expanded = -1
	m.groupScope.expandedCursor = 0
	if !m.groupScope.hasProvider() {
		m.groupScope.cursor = statsGroupScopeProvider
	}
	return m
}

func (m statsModel) closeGroupScopeOverlay() statsModel {
	m.groupScope.active = false
	m.groupScope.expanded = -1
	m.groupScope.expandedCursor = 0
	return m
}

func (m statsModel) clearGroupScope() statsModel {
	m.groupScope = m.groupScope.clear()
	return m
}

func (m statsModel) handleGroupScopeKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if m.groupScope.expanded >= 0 {
		return m.handleGroupScopeExpandedKey(msg)
	}

	if msg.Code == tea.KeyEscape || msg.Text == "q" {
		return m.closeGroupScopeOverlay().renderViewportContent(false), nil
	}
	if updated, handled := m.handleGroupScopeCursorKey(msg); handled {
		return updated, nil
	}
	if msg.Code == tea.KeyEnter || msg.Text == "l" || msg.Code == tea.KeyRight {
		return m.openGroupScopeSelection(), nil
	}
	if msg.Text == "x" {
		return m.clearGroupScopeSelection(), nil
	}
	return m, nil
}

func (m statsModel) handleGroupScopeExpandedKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if isGroupScopeCollapseKey(msg) {
		m.groupScope.expanded = -1
		return m.renderViewportContent(false), nil
	}
	if updated, handled := m.handleGroupScopeExpandedCursorKey(msg); handled {
		return updated, nil
	}
	switch {
	case msg.Text == "x":
		return m.clearGroupScopeExpandedSelection().renderViewportContent(false), nil
	case msg.Text == " " || msg.Code == tea.KeyEnter:
		return m.applyGroupScopeExpandedSelection().renderViewportContent(false), nil
	default:
		return m, nil
	}
}

func isGroupScopeCollapseKey(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyEscape ||
		msg.Text == "h" ||
		msg.Code == tea.KeyLeft ||
		msg.Text == "q"
}

func (m statsModel) handleGroupScopeCursorKey(msg tea.KeyPressMsg) (statsModel, bool) {
	switch {
	case msg.Text == "j" || msg.Code == tea.KeyDown:
		if m.groupScope.cursor < statsGroupScopeCount-1 {
			m.groupScope.cursor++
		}
		return m, true
	case msg.Text == "k" || msg.Code == tea.KeyUp:
		if m.groupScope.cursor > 0 {
			m.groupScope.cursor--
		}
		return m, true
	default:
		return m, false
	}
}

func (m statsModel) openGroupScopeSelection() statsModel {
	switch m.groupScope.cursor {
	case statsGroupScopeProvider:
		m.groupScope.expanded = statsGroupScopeProvider
		m.groupScope.expandedCursor = 0
	case statsGroupScopeVersion:
		if m.groupScope.hasProvider() {
			m.groupScope.expanded = statsGroupScopeVersion
			m.groupScope.expandedCursor = 0
		}
	}
	return m
}

func (m statsModel) clearGroupScopeSelection() statsModel {
	switch m.groupScope.cursor {
	case statsGroupScopeProvider:
		return m.clearGroupScope()
	case statsGroupScopeVersion:
		m.groupScope.versions = nil
	}
	return m
}

func (m statsModel) handleGroupScopeExpandedCursorKey(
	msg tea.KeyPressMsg,
) (statsModel, bool) {
	valuesCount := m.groupScopeExpandedValueCount()
	switch {
	case msg.Text == "j" || msg.Code == tea.KeyDown:
		if m.groupScope.expandedCursor < valuesCount-1 {
			m.groupScope.expandedCursor++
		}
		return m, true
	case msg.Text == "k" || msg.Code == tea.KeyUp:
		if m.groupScope.expandedCursor > 0 {
			m.groupScope.expandedCursor--
		}
		return m, true
	default:
		return m, false
	}
}

func (m statsModel) groupScopeExpandedValueCount() int {
	if m.groupScope.expanded == statsGroupScopeVersion {
		return len(m.groupScopeVersionValues(m.groupScope.provider))
	}
	return len(m.groupScopeProviderValues())
}

func (m statsModel) clearGroupScopeExpandedSelection() statsModel {
	if m.groupScope.expanded == statsGroupScopeProvider {
		return m.clearGroupScope()
	}
	m.groupScope.versions = nil
	m.groupScope.expanded = -1
	return m
}

func (m statsModel) applyGroupScopeExpandedSelection() statsModel {
	switch m.groupScope.expanded {
	case statsGroupScopeProvider:
		providers := m.groupScopeProviderValues()
		if m.groupScope.expandedCursor >= len(providers) {
			return m
		}
		m.groupScope.provider = providers[m.groupScope.expandedCursor]
		m.groupScope.versions = nil
		m.groupScope.expanded = -1
	case statsGroupScopeVersion:
		versions := m.groupScopeVersionValues(m.groupScope.provider)
		if m.groupScope.expandedCursor >= len(versions) {
			return m
		}
		selectedVersion := versions[m.groupScope.expandedCursor]
		if m.groupScope.versions == nil {
			m.groupScope.versions = make(map[string]bool)
		}
		if m.groupScope.versions[selectedVersion] {
			delete(m.groupScope.versions, selectedVersion)
		} else {
			m.groupScope.versions[selectedVersion] = true
		}
	}
	return m
}
