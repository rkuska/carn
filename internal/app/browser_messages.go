package app

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	conv "github.com/rkuska/carn/internal/conversation"
)

func (m browserModel) handleMsg(msg tea.Msg, cmds *[]tea.Cmd) browserModel {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKeyMsg(msg, cmds)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m.updateLayout()
	case conversationsLoadedMsg:
		return m.applyConversationsLoaded(msg, cmds)
	case sessionsLoadErrorMsg:
		return m.setNotification(
			errorNotification(fmt.Sprintf("load sessions failed: %v", msg.err)).notification,
			cmds,
		)
	case openViewerMsg:
		return m.applyOpenViewer(msg)
	case deepSearchDebounceMsg:
		return m.applyDeepSearchDebounce(msg, cmds)
	case deepSearchResultMsg:
		return m.applyDeepSearchResult(msg, cmds)
	case notificationMsg:
		return m.setNotification(msg.notification, cmds)
	case clearNotificationMsg:
		return m.clearNotifications()
	}
	return m
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
	if m.search.mode == searchModeDeep &&
		msg.revision == m.search.revision &&
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
	m.mainConversationCount = len(mainConvs)
	m.deepSearchAvailable = msg.deepSearchAvailable
	m.search.baseConversations = mainConvs
	m.search.visibleConversations = mainConvs
	if !m.deepSearchAvailable && m.search.mode == searchModeDeep {
		m.search.mode = searchModeMetadata
		m.search.status = searchStatusIdle
	}
	if m.search.query == "" {
		m = m.applyFullConversationList(cmds)
	} else {
		m = m.refreshSearchResults(cmds)
	}
	return m.syncTranscriptSelection(cmds)
}

func (m browserModel) applyDeepSearchResult(
	msg deepSearchResultMsg,
	cmds *[]tea.Cmd,
) browserModel {
	if msg.err != nil {
		m.search.status = searchStatusIdle
		m.searchCancel = nil
		return m.setNotification(
			errorNotification(fmt.Sprintf("deep search failed: %v", msg.err)).notification,
			cmds,
		)
	}
	if !msg.available {
		m.search.status = searchStatusIdle
		m.searchCancel = nil
		m.deepSearchAvailable = false
		if m.search.mode == searchModeDeep {
			m.search.mode = searchModeMetadata
			m = m.applyMetadataSearch(cmds)
		}
		return m.setNotification(
			infoNotification("deep search unavailable; re-import to rebuild the local index").notification,
			cmds,
		)
	}
	if m.search.mode == searchModeDeep &&
		msg.revision == m.search.revision &&
		msg.query == m.search.query {
		m.search.appliedRevision = msg.revision
		m.search.status = searchStatusIdle
		m.searchCancel = nil
		m = m.setDelegateHeight(delegateHeightDeepSearch)
		m = m.setSearchItems(buildDeepSearchItems(msg.query, msg.conversations), cmds)
		return m.syncTranscriptSelection(cmds)
	}
	return m
}

func (m browserModel) handleKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (browserModel, tea.Cmd) {
	if m.helpOpen {
		if key.Matches(msg, browserKeys.Help) || key.Matches(msg, browserKeys.Close) {
			m.helpOpen = false
		}
		return m, nil
	}

	var handled bool
	if m.transcriptVisible() {
		m, handled = m.handleTranscriptKey(msg)
		if handled {
			return m, nil
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
	case key.Matches(msg, browserKeys.Enter):
		if conv, ok := m.selectedConversation(); ok {
			m.transcriptMode = transcriptSplit
			m.focus = focusList
			m = m.updateLayout()
			return m.openTranscript(conv)
		}
	case key.Matches(msg, browserKeys.DeepSearch):
		return m.handleDeepSearchToggle(cmds), nil
	case key.Matches(msg, browserKeys.Editor):
		if conv, ok := m.selectedConversation(); ok {
			return m, openInEditorCmd(conv.LatestFilePath())
		}
	case key.Matches(msg, browserKeys.Resume):
		if conv, ok := m.selectedConversation(); ok {
			return m, resumeSessionCmd(conv.ResumeID(), conv.ResumeCWD())
		}
	case key.Matches(msg, browserKeys.Quit):
		return m, tea.Quit
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
	m.notification = n
	*cmds = append(*cmds, clearNotificationAfter(n.kind))
	return m
}
