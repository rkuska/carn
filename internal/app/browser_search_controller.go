package app

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	conv "github.com/rkuska/carn/internal/conversation"
)

const (
	delegateHeightDefault    = 3
	delegateHeightDeepSearch = 5
)

type deepSearchResultMsg struct {
	revision      int
	query         string
	conversations []conv.Conversation
	err           error
}

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

func deepSearchDebounceCmd(revision int, query string, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return deepSearchDebounceMsg{revision: revision, query: query}
	})
}

func (m browserModel) searchEditing() bool {
	return m.search.editing
}

func (m browserModel) beginSearchEditing() (browserModel, tea.Cmd) {
	m.search.editing = true
	m.searchInput.Focus()
	return m, textinput.Blink
}

func (m browserModel) stopSearchEditing() browserModel {
	m.search.editing = false
	m.searchInput.Blur()
	return m
}

func (m browserModel) cancelActiveDeepSearch() browserModel {
	if m.searchCancel != nil {
		m.searchCancel()
		m.searchCancel = nil
	}
	return m
}

func (m browserModel) updateSelectedConversationID() browserModel {
	if conv, ok := m.selectedConversation(); ok {
		m.search.selectedConversationID = conv.CacheKey()
		return m
	}
	m.search.selectedConversationID = ""
	return m
}

func (m browserModel) selectVisibleConversation(id string) (browserModel, bool) {
	for i, conv := range m.search.visibleConversations {
		if conv.CacheKey() != id {
			continue
		}
		m.list.Select(i)
		return m.updateSelectedConversationID(), true
	}
	return m, false
}

func (m browserModel) restoreSelection() browserModel {
	if len(m.search.visibleConversations) == 0 {
		return m
	}

	if m.search.selectedConversationID != "" {
		var ok bool
		m, ok = m.selectVisibleConversation(m.search.selectedConversationID)
		if ok {
			return m
		}
	}

	m.list.Select(0)
	return m.updateSelectedConversationID()
}

func (m browserModel) setSearchItems(items []conversationListItem, cmds *[]tea.Cmd) browserModel {
	m.search.visibleConversations = make([]conv.Conversation, 0, len(items))
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		m.search.visibleConversations = append(m.search.visibleConversations, item.conversation)
		listItems = append(listItems, item)
	}

	*cmds = append(*cmds, m.list.SetItems(listItems))
	return m.restoreSelection()
}

func (m browserModel) setDelegateHeight(height int) browserModel {
	m.delegate.SetHeight(height)
	m.list.SetDelegate(m.delegate)
	return m
}

func (m browserModel) applyMetadataSearch(cmds *[]tea.Cmd) browserModel {
	m.search.status = searchStatusIdle
	m.search.appliedRevision = m.search.revision
	m = m.cancelActiveDeepSearch()
	m = m.setDelegateHeight(delegateHeightDefault)
	return m.setSearchItems(buildMetadataSearchItems(m.search.query, m.search.baseConversations), cmds)
}

func (m browserModel) applyFullConversationList(cmds *[]tea.Cmd) browserModel {
	m.search.status = searchStatusIdle
	m.search.appliedRevision = m.search.revision
	m = m.cancelActiveDeepSearch()
	m = m.setDelegateHeight(delegateHeightDefault)
	return m.setSearchItems(buildPlainConversationItems(m.search.baseConversations), cmds)
}

func (m browserModel) scheduleDeepSearch(cmds *[]tea.Cmd) browserModel {
	m = m.cancelActiveDeepSearch()
	if m.search.query == "" {
		return m.applyFullConversationList(cmds)
	}

	m.search.status = searchStatusDebouncing
	delay := time.Duration(m.deepSearchDebounceMs) * time.Millisecond
	*cmds = append(*cmds, deepSearchDebounceCmd(m.search.revision, m.search.query, delay))
	return m
}

func (m browserModel) startDeepSearch(cmds *[]tea.Cmd) browserModel {
	m = m.cancelActiveDeepSearch()
	if m.search.query == "" {
		return m.applyFullConversationList(cmds)
	}

	searchCtx, cancel := context.WithCancel(m.ctx)
	m.searchCancel = cancel
	m.search.status = searchStatusSearching
	*cmds = append(
		*cmds,
		deepSearchRepositoryCmd(
			searchCtx,
			m.archiveDir,
			m.search.query,
			m.search.revision,
			m.search.baseConversations,
			m.store,
		),
	)
	return m
}

func deepSearchRepositoryCmd(
	ctx context.Context,
	archiveDir string,
	query string,
	revision int,
	mainConversations []conv.Conversation,
	store browserStore,
) tea.Cmd {
	return func() tea.Msg {
		conversations, err := store.DeepSearch(
			ctx,
			archiveDir,
			query,
			mainConversations,
		)
		return deepSearchResultMsg{
			revision:      revision,
			query:         query,
			conversations: conversations,
			err:           err,
		}
	}
}

func (m browserModel) refreshSearchResults(cmds *[]tea.Cmd) browserModel {
	switch m.search.mode {
	case searchModeMetadata:
		return m.applyMetadataSearch(cmds)
	case searchModeDeep:
		return m.scheduleDeepSearch(cmds)
	}
	return m
}

func (m browserModel) setSearchQuery(query string, cmds *[]tea.Cmd) browserModel {
	if query == m.search.query {
		return m
	}

	m.search.query = query
	m.search.revision++
	return m.refreshSearchResults(cmds)
}

func (m browserModel) toggleSearchMode(cmds *[]tea.Cmd) browserModel {
	if m.search.mode == searchModeDeep {
		m.search.mode = searchModeMetadata
		m.search.status = searchStatusIdle
		m.search.revision++
		return m.applyMetadataSearch(cmds)
	}

	m.search.mode = searchModeDeep
	m.search.revision++
	return m.startDeepSearch(cmds)
}

func (m browserModel) handleDeepSearchToggle(cmds *[]tea.Cmd) browserModel {
	previousMode := m.search.mode
	m = m.toggleSearchMode(cmds)
	if m.search.mode == previousMode {
		return m
	}

	m = m.setNotification(
		infoNotification(fmt.Sprintf("search scope: %s", m.searchScopeLabel())).notification,
		cmds,
	)
	return m.syncTranscriptSelection(cmds)
}

func (m browserModel) handleSearchKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	if key.Matches(msg, browserKeys.DeepSearch) {
		m = m.handleDeepSearchToggle(cmds)
		return m, nil
	}

	switch msg.Code {
	case tea.KeyEnter:
		m = m.stopSearchEditing()
		return m, nil
	case tea.KeyEscape:
		m = m.stopSearchEditing()
		m.searchInput.SetValue("")
		m = m.setSearchQuery("", cmds)
		return m, nil
	}

	var cmd tea.Cmd
	before := m.searchInput.Value()
	m.searchInput, cmd = m.searchInput.Update(msg)
	if after := m.searchInput.Value(); after != before {
		m = m.setSearchQuery(after, cmds)
	}

	return m, cmd
}
