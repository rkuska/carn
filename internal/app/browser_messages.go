package app

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	conv "github.com/rkuska/carn/internal/conversation"
)

func (m *browserModel) handleMsg(msg tea.Msg, cmds *[]tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		m.handleKeyMsg(msg, cmds)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
	case conversationsLoadedMsg:
		m.applyConversationsLoaded(msg, cmds)
	case sessionsLoadErrorMsg:
		m.setNotification(
			errorNotification(fmt.Sprintf("load sessions failed: %v", msg.err)).notification,
			cmds,
		)
	case openViewerMsg:
		m.applyOpenViewer(msg)
	case deepSearchDebounceMsg:
		m.applyDeepSearchDebounce(msg, cmds)
	case deepSearchResultMsg:
		m.applyDeepSearchResult(msg, cmds)
	case notificationMsg:
		m.setNotification(msg.notification, cmds)
	case clearNotificationMsg:
		m.clearNotifications()
	}
}

func (m *browserModel) handleKeyMsg(msg tea.KeyPressMsg, cmds *[]tea.Cmd) {
	if m.searchEditing() && !m.transcriptFocused() {
		var cmd tea.Cmd
		*m, cmd = m.handleSearchKey(msg, cmds)
		appendCmd(cmds, cmd)
		return
	}
	appendCmd(cmds, m.handleKey(msg, cmds))
}

func (m *browserModel) applyDeepSearchDebounce(msg deepSearchDebounceMsg, cmds *[]tea.Cmd) {
	if m.search.mode == searchModeDeep &&
		msg.revision == m.search.revision &&
		msg.query == m.search.query {
		m.startDeepSearch(cmds)
	}
}

func (m *browserModel) applyConversationsLoaded(msg conversationsLoadedMsg, cmds *[]tea.Cmd) {
	m.allConversations = msg.conversations
	mainConvs := filterMainConversations(msg.conversations)
	m.mainConversationCount = len(mainConvs)
	m.deepSearchAvailable = msg.deepSearchAvailable
	m.search.baseConversations = mainConvs
	m.search.visibleConversations = mainConvs
	if !m.deepSearchAvailable && m.search.mode == searchModeDeep {
		m.search.mode = searchModeMetadata
		m.search.status = searchStatusIdle
	}
	if m.search.query == "" {
		m.applyFullConversationList(cmds)
	} else {
		m.refreshSearchResults(cmds)
	}
	m.syncTranscriptSelection(cmds)
}

func (m *browserModel) applyDeepSearchResult(msg deepSearchResultMsg, cmds *[]tea.Cmd) {
	if msg.err != nil {
		m.search.status = searchStatusIdle
		m.searchCancel = nil
		m.setNotification(
			errorNotification(fmt.Sprintf("deep search failed: %v", msg.err)).notification,
			cmds,
		)
		return
	}
	if !msg.available {
		m.search.status = searchStatusIdle
		m.searchCancel = nil
		m.deepSearchAvailable = false
		if m.search.mode == searchModeDeep {
			m.search.mode = searchModeMetadata
			m.applyMetadataSearch(cmds)
		}
		m.setNotification(
			infoNotification("deep search unavailable; re-import to rebuild the local index").notification,
			cmds,
		)
		return
	}
	if m.search.mode == searchModeDeep &&
		msg.revision == m.search.revision &&
		msg.query == m.search.query {
		m.search.appliedRevision = msg.revision
		m.search.status = searchStatusIdle
		m.searchCancel = nil
		m.setDelegateHeight(delegateHeightDeepSearch)
		m.setSearchItems(buildDeepSearchItems(msg.query, msg.conversations), cmds)
		m.syncTranscriptSelection(cmds)
	}
}

func (m *browserModel) handleKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) tea.Cmd {
	if m.helpOpen {
		if key.Matches(msg, browserKeys.Help) || key.Matches(msg, browserKeys.Close) {
			m.helpOpen = false
		}
		return nil
	}

	if m.transcriptVisible() && m.handleTranscriptKey(msg) {
		return nil
	}
	if m.transcriptFocused() {
		return nil
	}
	if key.Matches(msg, browserKeys.Help) && !m.searchEditing() {
		m.helpOpen = true
		return nil
	}
	if m.searchEditing() {
		m.pendingListGotoTopKey = false
		return nil
	}

	return m.handleListNavigation(msg, cmds)
}

func (m *browserModel) handleListNavigation(msg tea.KeyPressMsg, cmds *[]tea.Cmd) tea.Cmd {
	if msg.Text == "g" {
		if m.pendingListGotoTopKey {
			m.list.GoToStart()
			m.pendingListGotoTopKey = false
			m.syncTranscriptSelection(cmds)
			return nil
		}
		m.pendingListGotoTopKey = true
		return nil
	}
	m.pendingListGotoTopKey = false
	return m.handleListKey(msg, cmds)
}

func (m *browserModel) handleListKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) tea.Cmd {
	switch {
	case key.Matches(msg, browserKeys.Search):
		return m.beginSearchEditing()
	case key.Matches(msg, browserKeys.Enter):
		if conv, ok := m.selectedConversation(); ok {
			m.transcriptMode = transcriptSplit
			m.focus = focusList
			m.updateLayout()
			return m.openTranscript(conv)
		}
	case key.Matches(msg, browserKeys.DeepSearch):
		m.handleDeepSearchToggle(cmds)
		return nil
	case key.Matches(msg, browserKeys.Editor):
		if conv, ok := m.selectedConversation(); ok {
			return openInEditorCmd(conv.LatestFilePath())
		}
	case key.Matches(msg, browserKeys.Resume):
		if conv, ok := m.selectedConversation(); ok {
			return resumeSessionCmd(conv.ResumeID(), conv.ResumeCWD())
		}
	case key.Matches(msg, browserKeys.Quit):
		return tea.Quit
	}
	return nil
}

func (m *browserModel) selectedConversation() (conv.Conversation, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return conv.Conversation{}, false
	}
	return conversationFromItem(item)
}

func (m *browserModel) setNotification(n notification, cmds *[]tea.Cmd) {
	m.notification = n
	*cmds = append(*cmds, clearNotificationAfter(n.kind))
}
