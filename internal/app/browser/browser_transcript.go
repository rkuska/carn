package browser

import (
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	conv "github.com/rkuska/carn/internal/conversation"
)

func (m browserModel) canHandleTranscriptAction() bool {
	return !m.isFiltering() && !m.filter.Active && !m.viewer.searching && !m.viewer.hasActiveOverlay()
}

func (m browserModel) handleTranscriptKey(msg tea.KeyPressMsg) (browserModel, bool) {
	switch {
	case keyMatchesBrowserHelp(msg):
		m = m.handleTranscriptHelpKey()
		return m, true
	case keyMatchesTranscriptToggle(msg):
		m = m.handleTranscriptToggleKey()
		return m, true
	case keyMatchesBrowserClose(msg):
		m = m.handleTranscriptCloseKey()
		return m, true
	case m.transcriptMode == transcriptSplit && keyMatchesBrowserFocus(msg):
		m = m.handleTranscriptFocusKey()
		return m, true
	}
	return m, false
}

func (m browserModel) handleTranscriptHelpKey() browserModel {
	if m.viewer.hasActiveOverlay() {
		m.helpOpen = true
		return m
	}
	if m.canHandleTranscriptAction() {
		m.helpOpen = true
	}
	return m
}

func (m browserModel) handleTranscriptToggleKey() browserModel {
	if m.canHandleTranscriptAction() {
		return m.toggleTranscriptLayout()
	}
	return m
}

func (m browserModel) handleTranscriptCloseKey() browserModel {
	if m.canHandleTranscriptAction() {
		return m.closeTranscript()
	}
	return m
}

func (m browserModel) handleTranscriptFocusKey() browserModel {
	if m.canHandleTranscriptAction() {
		return m.toggleFocus()
	}
	return m
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

func (m browserModel) toggleFocus() browserModel {
	if m.focus == focusList {
		m.focus = focusTranscript
		return m
	}
	m.focus = focusList
	return m
}

func (m browserModel) openTranscript(conversation conv.Conversation) (browserModel, tea.Cmd) {
	if session, ok := m.sessionCache[conversation.CacheKey()]; ok {
		return m.installViewer(session, conversation), nil
	}

	m.openConversationID = ""
	m.loadingConversationID = conversation.CacheKey()
	return m, openConversationCmdWithStore(m.ctx, m.archiveDir, conversation, m.store)
}

func (m browserModel) installViewer(session conv.Session, conversation conv.Conversation) browserModel {
	key := conversation.CacheKey()
	if key == "" {
		key = session.Meta.ID
	}
	m.openConversationID = key
	m.loadingConversationID = ""
	m.sessionCache[key] = session
	m = m.addToCache(key)
	if m.transcriptMode == transcriptClosed {
		m.transcriptMode = transcriptSplit
	}

	if cached, ok := m.cachedViewerForOpen(key, session, conversation); ok {
		m.viewer = cached
	} else {
		m.viewer = newViewerModelWithLauncher(
			session,
			conversation,
			m.glamourStyle,
			m.timestampFormat,
			m.viewerWidth(),
			m.height,
			m.theme,
			m.launcher,
		)
		m.viewerCache[key] = m.viewer
	}
	if m.transcriptMode == transcriptFullscreen {
		m.focus = focusTranscript
	}
	return m
}

func (m browserModel) syncTranscriptSelection(cmds *[]tea.Cmd) browserModel {
	if m.transcriptMode != transcriptSplit || m.helpOpen {
		return m
	}

	conversation, ok := m.selectedConversation()
	if !ok || conversation.CacheKey() == m.openConversationID || conversation.CacheKey() == m.loadingConversationID {
		return m
	}

	var cmd tea.Cmd
	m, cmd = m.openTranscript(conversation)
	if cmd != nil {
		*cmds = append(*cmds, cmd)
	}
	return m
}

func (m browserModel) reloadTranscriptAfterResync(cmds *[]tea.Cmd) browserModel {
	if m.pendingResyncTranscriptID == "" || m.helpOpen {
		return m
	}
	pendingID := m.pendingResyncTranscriptID
	m.pendingResyncTranscriptID = ""
	if !m.transcriptVisible() {
		return m
	}

	var ok bool
	m, ok = m.selectVisibleConversation(pendingID)
	if !ok {
		return m.closeTranscript()
	}

	conversation, ok := m.selectedConversation()
	if !ok {
		return m.closeTranscript()
	}

	var cmd tea.Cmd
	m, cmd = m.openTranscript(conversation)
	if cmd != nil {
		*cmds = append(*cmds, cmd)
	}
	return m
}

func (m browserModel) closeTranscript() browserModel {
	m.transcriptMode = transcriptClosed
	m.focus = focusList
	m.helpOpen = false
	m.loadingConversationID = ""
	m.openConversationID = ""
	m.pendingResyncTranscriptID = ""
	return m.updateLayout()
}

func (m browserModel) toggleTranscriptLayout() browserModel {
	switch m.transcriptMode {
	case transcriptClosed:
		return m
	case transcriptSplit:
		m.transcriptMode = transcriptFullscreen
		m.focus = focusTranscript
	case transcriptFullscreen:
		m.transcriptMode = transcriptSplit
		m.focus = focusList
	}
	return m.updateLayout()
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
	if m.helpOpen || m.filter.Active || m.transcriptMode == transcriptFullscreen || m.searchEditing() {
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

func (m browserModel) addToCache(id string) browserModel {
	if slices.Contains(m.cacheOrder, id) {
		return m
	}

	m.cacheOrder = append(m.cacheOrder, id)
	for len(m.cacheOrder) > m.browserCacheSize {
		evictID := m.cacheOrder[0]
		m.cacheOrder = m.cacheOrder[1:]
		delete(m.sessionCache, evictID)
		delete(m.viewerCache, evictID)
	}
	return m
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
