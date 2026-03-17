package app

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/rs/zerolog"

	conv "github.com/rkuska/carn/internal/conversation"
)

func (m browserModel) handleMsg(msg tea.Msg, cmds *[]tea.Cmd) browserModel {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		return m.handleKeyMsg(keyMsg, cmds)
	}

	if next, handled := m.handleViewMsg(msg, cmds); handled {
		return next
	}
	if next, handled := m.handleAsyncMsg(msg, cmds); handled {
		return next
	}
	return m
}

func (m browserModel) handleViewMsg(msg tea.Msg, cmds *[]tea.Cmd) (browserModel, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m.updateLayout(), true
	case notificationMsg:
		return m.setNotification(msg.notification, cmds), true
	case clearNotificationMsg:
		return m.clearNotifications(), true
	case spinner.TickMsg:
		if !m.resyncSpinnerActive() {
			return m, true
		}
		var cmd tea.Cmd
		m.resyncSpinner, cmd = m.resyncSpinner.Update(msg)
		appendCmd(cmds, cmd)
		return m, true
	}
	return m, false
}

func (m browserModel) handleAsyncMsg(msg tea.Msg, cmds *[]tea.Cmd) (browserModel, bool) {
	switch msg := msg.(type) {
	case conversationsLoadedMsg:
		return m.applyConversationsLoaded(msg, cmds), true
	case sessionsLoadErrorMsg:
		return m.setNotification(
			errorNotification(fmt.Sprintf("load sessions failed: %v", msg.err)).notification,
			cmds,
		), true
	case openViewerMsg:
		return m.applyOpenViewer(msg), true
	case deepSearchDebounceMsg:
		return m.applyDeepSearchDebounce(msg, cmds), true
	case deepSearchResultMsg:
		return m.applyDeepSearchResult(msg, cmds), true
	}
	return m, false
}

func (m browserModel) handleKeyMsg(msg tea.KeyPressMsg, cmds *[]tea.Cmd) browserModel {
	if m.searchEditing() && !m.transcriptFocused() {
		var cmd tea.Cmd
		m, cmd = m.handleSearchKey(msg, cmds)
		appendCmd(cmds, cmd)
		return m
	}
	var cmd tea.Cmd
	m, cmd = m.handleKey(msg, cmds)
	appendCmd(cmds, cmd)
	return m
}

func (m browserModel) applyDeepSearchDebounce(
	msg deepSearchDebounceMsg,
	cmds *[]tea.Cmd,
) browserModel {
	if msg.revision == m.search.revision &&
		msg.query == m.search.query {
		return m.startDeepSearch(cmds)
	}
	return m
}

func (m browserModel) applyConversationsLoaded(
	msg conversationsLoadedMsg,
	cmds *[]tea.Cmd,
) browserModel {
	m.allConversations = msg.conversations
	mainConvs := filterMainConversations(msg.conversations)
	m.mainConversations = mainConvs
	m.filter.values = extractFilterValues(mainConvs)
	filtered := applyStructuredFilters(mainConvs, m.filter.dimensions)
	m.search.baseConversations = filtered
	if m.search.query == "" {
		m = m.applyFullConversationList(cmds)
		m = m.reloadTranscriptAfterResync(cmds)
		return m.syncTranscriptSelection(cmds)
	}

	return m.refreshSearchResults(cmds)
}

func (m browserModel) applyDeepSearchResult(
	msg deepSearchResultMsg,
	cmds *[]tea.Cmd,
) browserModel {
	if !m.matchesActiveDeepSearch(msg) {
		return m
	}
	if msg.err != nil {
		m.search.status = searchStatusIdle
		m.searchCancel = nil
		return m.setNotification(
			errorNotification(fmt.Sprintf("deep search failed: %v", msg.err)).notification,
			cmds,
		)
	}
	m.search.appliedRevision = msg.revision
	m.search.status = searchStatusIdle
	m.searchCancel = nil
	m = m.setDelegateHeight(delegateHeightDeepSearch)
	m = m.setSearchItems(buildDeepSearchItems(msg.query, msg.conversations), cmds)
	m = m.reloadTranscriptAfterResync(cmds)
	return m.syncTranscriptSelection(cmds)
}

func (m browserModel) matchesActiveDeepSearch(msg deepSearchResultMsg) bool {
	return msg.revision == m.search.revision &&
		msg.query == m.search.query
}

func (m browserModel) handleKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	if m.filter.active {
		return m.handleFilterKey(msg, cmds)
	}
	if m.helpOpen {
		return m.handleHelpKey(msg, cmds)
	}
	if m.transcriptVisible() {
		if next, handled := m.handleTranscriptKey(msg); handled {
			return next, nil
		}
	}
	if m.transcriptFocused() {
		return m, nil
	}
	if key.Matches(msg, browserKeys.Help) && !m.searchEditing() {
		m.helpOpen = true
		return m, nil
	}
	if m.searchEditing() {
		m.pendingListGotoTopKey = false
		return m, nil
	}
	return m.handleListNavigation(msg, cmds)
}

func (m browserModel) handleHelpKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	if key.Matches(msg, browserKeys.Help) || key.Matches(msg, browserKeys.Close) {
		m.helpOpen = false
		m = m.reloadTranscriptAfterResync(cmds)
	}
	return m, nil
}

func (m browserModel) handleListNavigation(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	if msg.Text == "g" {
		if m.pendingListGotoTopKey {
			m.list.GoToStart()
			m.pendingListGotoTopKey = false
			return m.syncTranscriptSelection(cmds), nil
		}
		m.pendingListGotoTopKey = true
		return m, nil
	}
	m.pendingListGotoTopKey = false
	return m.handleListKey(msg, cmds)
}

func (m browserModel) handleListKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, browserKeys.Search):
		return m.beginSearchEditing()
	case key.Matches(msg, browserKeys.ClearSearch) && m.hasActiveSearch():
		return m.clearSearch(cmds), nil
	case key.Matches(msg, browserKeys.Filter):
		return m.openFilterOverlay(), nil
	case key.Matches(msg, browserKeys.Enter):
		if conv, ok := m.selectedConversation(); ok {
			m.transcriptMode = transcriptSplit
			m.focus = focusList
			m = m.updateLayout()
			return m.openTranscript(conv)
		}
	case key.Matches(msg, browserKeys.Quit):
		return m, tea.Quit
	}
	return m.handleListActionKey(msg)
}

func (m browserModel) handleListActionKey(msg tea.KeyPressMsg) (browserModel, tea.Cmd) {
	switch {
	case key.Matches(msg, browserKeys.Editor):
		if conv, ok := m.selectedConversation(); ok {
			return m, openInEditorCmd(conv.LatestFilePath())
		}
	case key.Matches(msg, browserKeys.Resume):
		if conv, ok := m.selectedConversation(); ok {
			return m, resumeSessionCmd(conv.ResumeTarget(), m.launcher)
		}
	case key.Matches(msg, browserKeys.Resync):
		if !m.resync.active {
			return m, m.requestResyncCmd()
		}
	}
	return m, nil
}

func (m browserModel) selectedConversation() (conv.Conversation, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return conv.Conversation{}, false
	}
	return conversationFromItem(item)
}

func (m browserModel) setNotification(n notification, cmds *[]tea.Cmd) browserModel {
	if n.kind == notificationError {
		zerolog.Ctx(m.ctx).Warn().Str("notification", n.text).Msg("error shown")
	}
	m.notification = n
	*cmds = append(*cmds, clearNotificationAfter(n.kind))
	return m
}
