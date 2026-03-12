package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m viewerModel) clearSearch() viewerModel {
	m.searchQuery = ""
	m.matches = nil
	m.currentMatch = 0
	m.searchMatchesValid = false
	return m
}

func (m viewerModel) handleSearchKey(msg tea.KeyPressMsg) (viewerModel, tea.Cmd) {
	if msg.Code == tea.KeyEnter {
		m.searching = false
		m.searchQuery = m.searchInput.Value()
		m.searchInput.Blur()
		m = m.performSearch()
		return m, nil
	}

	if msg.Code == tea.KeyEscape {
		m.searching = false
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m = m.clearSearch()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m viewerModel) rebuildSearchIndex(content string) viewerModel {
	lines := strings.Split(content, "\n")
	indexedLines := make([]searchLineIndex, len(lines))
	for i, line := range lines {
		indexedLines[i] = buildSearchLineIndex(line, 0)
	}
	m.searchLines = indexedLines
	m.searchIndexVersion++
	return m
}

func (m viewerModel) performSearch() viewerModel {
	m.currentMatch = 0
	if !m.searchMatchesValid ||
		m.searchAppliedVersion != m.searchIndexVersion ||
		m.searchAppliedQuery != m.searchQuery {
		m.matches = collectSearchOccurrences(m.searchLines, m.searchQuery)
		m.searchAppliedVersion = m.searchIndexVersion
		m.searchAppliedQuery = m.searchQuery
		m.searchMatchesValid = true
	}

	if len(m.matches) == 0 {
		return m
	}

	m.viewport.SetYOffset(m.matches[0].line)
	return m
}

func (m viewerModel) jumpToMatch(delta int) viewerModel {
	if len(m.matches) == 0 || delta == 0 {
		return m
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
	return m
}
