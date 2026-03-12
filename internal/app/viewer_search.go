package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m *viewerModel) clearSearch() {
	m.searchQuery = ""
	m.matches = nil
	m.currentMatch = 0
}

func (m viewerModel) handleSearchKey(msg tea.KeyPressMsg) (viewerModel, tea.Cmd) {
	if msg.Code == tea.KeyEnter {
		m.searching = false
		m.searchQuery = m.searchInput.Value()
		m.searchInput.Blur()
		m.performSearch()
		return m, nil
	}

	if msg.Code == tea.KeyEscape {
		m.searching = false
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.clearSearch()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m *viewerModel) rebuildSearchIndex(content string) {
	lines := strings.Split(content, "\n")
	indexedLines := make([]searchLineIndex, len(lines))
	for i, line := range lines {
		indexedLines[i] = buildSearchLineIndex(line, 0)
	}
	m.searchLines = indexedLines
}

func (m *viewerModel) performSearch() {
	m.matches = collectSearchOccurrences(m.searchLines, m.searchQuery)
	m.currentMatch = 0

	if len(m.matches) == 0 {
		return
	}

	m.viewport.SetYOffset(m.matches[0].line)
}

func (m *viewerModel) jumpToMatch(delta int) {
	if len(m.matches) == 0 || delta == 0 {
		return
	}

	steps := delta
	if steps < 0 {
		steps = -steps
	}

	for range steps {
		if delta > 0 {
			m.currentMatch = (m.currentMatch + 1) % len(m.matches)
			m.viewport.SetYOffset(m.matches[m.currentMatch].line)
			continue
		}

		m.currentMatch = (m.currentMatch - 1 + len(m.matches)) % len(m.matches)
		m.viewport.SetYOffset(m.matches[m.currentMatch].line)
	}
}
