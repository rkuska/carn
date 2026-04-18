package browser

import (
	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

func (m browserModel) cachedViewerForOpen(
	key string,
	session conv.Session,
	conversation conv.Conversation,
) (viewerModel, bool) {
	cached, ok := m.viewerCache[key]
	if !ok {
		return viewerModel{}, false
	}
	if !cached.canReuseForOpen(
		session,
		conversation,
		m.viewerWidth(),
		m.theme,
		m.glamourStyle,
		m.timestampFormat,
	) {
		return viewerModel{}, false
	}
	return cached.resetForOpen(
		session,
		conversation,
		m.height,
		m.theme,
		m.launcher,
	), true
}

func (m viewerModel) canReuseForOpen(
	session conv.Session,
	conversation conv.Conversation,
	width int,
	theme *el.Theme,
	glamourStyle string,
	timestampFormat string,
) bool {
	if m.theme != theme ||
		m.width != width ||
		m.glamourStyle != glamourStyle ||
		m.timestampFormat != timestampFormat {
		return false
	}

	if m.conversation.CacheKey() != conversation.CacheKey() ||
		m.session.Meta.ID != session.Meta.ID ||
		!m.session.Meta.Timestamp.Equal(session.Meta.Timestamp) ||
		!m.session.Meta.LastTimestamp.Equal(session.Meta.LastTimestamp) ||
		len(m.session.Messages) != len(session.Messages) {
		return false
	}

	return true
}

func (m viewerModel) resetForOpen(
	session conv.Session,
	conversation conv.Conversation,
	height int,
	theme *el.Theme,
	launcher sessionLauncher,
) viewerModel {
	m.conversation = conversation
	m.session = session
	m.theme = theme
	m.launcher = resolveSessionLauncher(launcher)
	m.content = scanContentFlags(session.Messages)
	m.height = height
	m.selectionMode = false
	m = m.applySelectionMode()
	m.viewport.GotoTop()
	m.viewport.ClearHighlights()

	m.opts = transcriptOptions{}
	m.searching = false
	m.searchQuery = ""
	m.searchAppliedQuery = ""
	m.searchMatchesValid = false
	m.matches = nil
	m.currentMatch = 0
	m.notification = notification{}
	m.pendingGotoTopKey = false
	m.planExpanded = false
	m.actionMode = viewerActionNone
	m.planPicker = viewerPlanPickerState{}
	m.searchInput.SetValue("")
	m.searchInput.Blur()

	return m
}
