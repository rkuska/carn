package app

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const (
	deepSearchDebounceDelay  = 200 * time.Millisecond
	delegateHeightDefault    = 3
	delegateHeightDeepSearch = 5
)

type deepSearchDebounceMsg struct {
	revision int
	query    string
}

func newBrowserSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.CharLimit = 100
	ti.Blur()
	return ti
}

func deepSearchDebounceCmd(revision int, query string) tea.Cmd {
	return tea.Tick(deepSearchDebounceDelay, func(time.Time) tea.Msg {
		return deepSearchDebounceMsg{revision: revision, query: query}
	})
}

func (m *browserModel) searchEditing() bool {
	return m.search.editing
}

func (m *browserModel) beginSearchEditing() tea.Cmd {
	m.search.editing = true
	m.searchInput.Focus()
	return textinput.Blink
}

func (m *browserModel) stopSearchEditing() {
	m.search.editing = false
	m.searchInput.Blur()
}

func (m *browserModel) cancelActiveDeepSearch() {
	if m.searchCancel != nil {
		m.searchCancel()
		m.searchCancel = nil
	}
}

func (m *browserModel) updateSelectedConversationID() {
	if conv, ok := m.selectedConversation(); ok {
		m.search.selectedConversationID = conv.cacheKey()
		return
	}
	m.search.selectedConversationID = ""
}

func (m *browserModel) restoreSelection() {
	if len(m.search.visibleConversations) == 0 {
		return
	}

	if m.search.selectedConversationID != "" {
		for i, conv := range m.search.visibleConversations {
			if conv.cacheKey() == m.search.selectedConversationID {
				m.list.Select(i)
				return
			}
		}
	}

	m.list.Select(0)
	m.updateSelectedConversationID()
}

func (m *browserModel) setSearchItems(items []conversationListItem, cmds *[]tea.Cmd) {
	m.search.visibleConversations = make([]conversation, 0, len(items))
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		m.search.visibleConversations = append(m.search.visibleConversations, item.conversation)
		listItems = append(listItems, item)
	}

	*cmds = append(*cmds, m.list.SetItems(listItems))
	m.restoreSelection()
}

func (m *browserModel) setDelegateHeight(height int) {
	m.delegate.SetHeight(height)
	m.list.SetDelegate(m.delegate)
}

func (m *browserModel) applyMetadataSearch(cmds *[]tea.Cmd) {
	m.search.status = searchStatusIdle
	m.search.appliedRevision = m.search.revision
	m.cancelActiveDeepSearch()
	m.setDelegateHeight(delegateHeightDefault)
	m.setSearchItems(buildMetadataSearchItems(m.search.query, m.search.baseConversations), cmds)
}

func (m *browserModel) applyFullConversationList(cmds *[]tea.Cmd) {
	m.search.status = searchStatusIdle
	m.search.appliedRevision = m.search.revision
	m.cancelActiveDeepSearch()
	m.setDelegateHeight(delegateHeightDefault)
	m.setSearchItems(buildPlainConversationItems(m.search.baseConversations), cmds)
}

func (m *browserModel) scheduleDeepSearch(cmds *[]tea.Cmd) {
	m.cancelActiveDeepSearch()
	if m.search.query == "" {
		m.applyFullConversationList(cmds)
		return
	}

	m.search.status = searchStatusDebouncing
	*cmds = append(*cmds, deepSearchDebounceCmd(m.search.revision, m.search.query))
}

func (m *browserModel) startDeepSearch(cmds *[]tea.Cmd) {
	m.cancelActiveDeepSearch()
	if m.search.query == "" {
		m.applyFullConversationList(cmds)
		return
	}
	if !m.deepSearchAvailable {
		m.search.status = searchStatusIdle
		return
	}

	searchCtx, cancel := context.WithCancel(m.ctx)
	m.searchCancel = cancel
	m.search.status = searchStatusSearching
	*cmds = append(
		*cmds,
		deepSearchCmd(
			searchCtx,
			m.search.query,
			m.search.revision,
			m.search.baseConversations,
			m.searchCorpus,
		),
	)
}

func (m *browserModel) refreshSearchResults(cmds *[]tea.Cmd) {
	switch m.search.mode {
	case searchModeMetadata:
		m.applyMetadataSearch(cmds)
	case searchModeDeep:
		m.scheduleDeepSearch(cmds)
	}
}

func (m *browserModel) setSearchQuery(query string, cmds *[]tea.Cmd) {
	if query == m.search.query {
		return
	}

	m.search.query = query
	m.search.revision++
	m.refreshSearchResults(cmds)
}

func (m *browserModel) toggleSearchMode(cmds *[]tea.Cmd) {
	if m.search.mode == searchModeDeep {
		m.search.mode = searchModeMetadata
		m.search.status = searchStatusIdle
		m.search.revision++
		m.applyMetadataSearch(cmds)
		return
	}
	if !m.deepSearchAvailable {
		m.setNotification(
			infoNotification("deep search unavailable; re-import to rebuild the local index").notification,
			cmds,
		)
		return
	}

	m.search.mode = searchModeDeep
	m.search.revision++
	m.startDeepSearch(cmds)
}

func (m *browserModel) handleDeepSearchToggle(cmds *[]tea.Cmd) {
	previousMode := m.search.mode
	m.toggleSearchMode(cmds)
	if m.search.mode == previousMode {
		return
	}

	m.setNotification(
		infoNotification(fmt.Sprintf("search scope: %s", m.searchScopeLabel())).notification,
		cmds,
	)
	m.syncTranscriptSelection(cmds)
}

func (m browserModel) handleSearchKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	if key.Matches(msg, browserKeys.DeepSearch) {
		m.handleDeepSearchToggle(cmds)
		return m, nil
	}

	switch msg.Code {
	case tea.KeyEnter:
		m.stopSearchEditing()
		return m, nil
	case tea.KeyEscape:
		m.stopSearchEditing()
		m.searchInput.SetValue("")
		m.setSearchQuery("", cmds)
		return m, nil
	}

	var cmd tea.Cmd
	before := m.searchInput.Value()
	m.searchInput, cmd = m.searchInput.Update(msg)
	if after := m.searchInput.Value(); after != before {
		m.setSearchQuery(after, cmds)
	}

	return m, cmd
}
