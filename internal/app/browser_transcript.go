package app

import (
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	conv "github.com/rkuska/carn/internal/conversation"
)

func (m browserModel) canHandleTranscriptAction() bool {
	return !m.isFiltering() && !m.viewer.searching
}

func (m *browserModel) handleTranscriptKey(msg tea.KeyPressMsg) bool {
	switch {
	case keyMatchesBrowserHelp(msg):
		if m.canHandleTranscriptAction() {
			m.helpOpen = true
		}
		return true
	case keyMatchesTranscriptToggle(msg):
		if m.canHandleTranscriptAction() {
			m.toggleTranscriptLayout()
		}
		return true
	case keyMatchesBrowserClose(msg):
		if m.canHandleTranscriptAction() {
			m.closeTranscript()
		}
		return true
	case m.transcriptMode == transcriptSplit && keyMatchesBrowserFocus(msg):
		if m.canHandleTranscriptAction() {
			m.toggleFocus()
		}
		return true
	}
	return false
}

func keyMatchesBrowserHelp(msg tea.KeyPressMsg) bool {
	return keyMatches(msg, browserKeys.Help)
}

func keyMatchesBrowserClose(msg tea.KeyPressMsg) bool {
	return keyMatches(msg, browserKeys.Close)
}

func keyMatchesBrowserFocus(msg tea.KeyPressMsg) bool {
	return keyMatches(msg, browserKeys.FocusPane)
}

func keyMatchesTranscriptToggle(msg tea.KeyPressMsg) bool {
	return keyMatches(msg, browserKeys.ToggleFullscreen)
}

func keyMatches(msg tea.KeyPressMsg, binding key.Binding) bool {
	return key.Matches(msg, binding)
}

func (m *browserModel) toggleFocus() {
	if m.focus == focusList {
		m.focus = focusTranscript
		return
	}
	m.focus = focusList
}

func (m *browserModel) openTranscript(conversation conv.Conversation) tea.Cmd {
	if session, ok := m.transcriptCache[conversation.CacheKey()]; ok {
		m.installViewer(session, conversation)
		return nil
	}

	m.openConversationID = ""
	m.loadingConversationID = conversation.CacheKey()
	if session, ok := m.sessionCache[conversation.CacheKey()]; ok {
		return openConversationCmdCachedWithStore(m.ctx, conversation, session)
	}
	return openConversationCmdWithStore(m.ctx, m.archiveDir, conversation, m.store)
}

func (m *browserModel) installViewer(session conv.Session, conversation conv.Conversation) {
	key := conversation.CacheKey()
	if key == "" {
		key = session.Meta.ID
	}
	m.openConversationID = key
	m.loadingConversationID = ""
	m.transcriptCache[key] = session
	m.sessionCache[key] = session
	m.addToCache(key)

	m.viewer = newViewerModel(session, conversation, m.glamourStyle, m.viewerWidth(), m.height)
	if m.transcriptMode == transcriptClosed {
		m.transcriptMode = transcriptSplit
	}
	if m.transcriptMode == transcriptFullscreen {
		m.focus = focusTranscript
	}
}

func (m *browserModel) syncTranscriptSelection(cmds *[]tea.Cmd) {
	if m.transcriptMode != transcriptSplit || m.helpOpen {
		return
	}

	conversation, ok := m.selectedConversation()
	if !ok || conversation.CacheKey() == m.openConversationID || conversation.CacheKey() == m.loadingConversationID {
		return
	}

	cmd := m.openTranscript(conversation)
	if cmd != nil {
		*cmds = append(*cmds, cmd)
	}
}

func (m *browserModel) closeTranscript() {
	m.transcriptMode = transcriptClosed
	m.focus = focusList
	m.helpOpen = false
	m.loadingConversationID = ""
	m.openConversationID = ""
	m.updateLayout()
}

func (m *browserModel) toggleTranscriptLayout() {
	switch m.transcriptMode {
	case transcriptClosed:
		return
	case transcriptSplit:
		m.transcriptMode = transcriptFullscreen
		m.focus = focusTranscript
	case transcriptFullscreen:
		m.transcriptMode = transcriptSplit
		m.focus = focusList
	}
	m.updateLayout()
}

func (m browserModel) transcriptVisible() bool {
	return m.transcriptMode != transcriptClosed
}

func (m browserModel) transcriptFocused() bool {
	return m.transcriptVisible() &&
		(m.transcriptMode == transcriptFullscreen || m.focus == focusTranscript)
}

func (m browserModel) isFiltering() bool {
	return m.search.editing
}

func (m browserModel) shouldUpdateList(isKey bool) bool {
	if !isKey {
		return true
	}
	if m.helpOpen || m.transcriptMode == transcriptFullscreen || m.searchEditing() {
		return false
	}
	return m.focus == focusList
}

func (m browserModel) shouldUpdateViewer(isKey bool) bool {
	if !m.transcriptVisible() || m.viewer.session.Meta.ID == "" || !isKey || m.helpOpen {
		return false
	}
	return m.transcriptFocused()
}

func (m *browserModel) addToCache(id string) {
	if slices.Contains(m.cacheOrder, id) {
		return
	}

	m.cacheOrder = append(m.cacheOrder, id)
	for len(m.cacheOrder) > browserCacheSize {
		evictID := m.cacheOrder[0]
		m.cacheOrder = m.cacheOrder[1:]
		delete(m.sessionCache, evictID)
		delete(m.transcriptCache, evictID)
	}
}

func filterMainConversations(conversations []conv.Conversation) []conv.Conversation {
	items := make([]conv.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		if !conversation.IsSubagent() {
			items = append(items, conversation)
		}
	}
	return items
}
