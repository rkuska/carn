package app

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	conv "github.com/rkuska/carn/internal/conversation"
)

type statsSessionLoadedMsg struct {
	conversation conv.Conversation
	session      conv.Session
}

func (m statsModel) updateViewer(msg tea.Msg) (statsModel, tea.Cmd) {
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
	}
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok &&
		!m.viewer.searching &&
		!m.viewer.hasActiveOverlay() &&
		keyMatches(keyMsg, viewerKeys.Back) {
		m.viewerOpen = false
		return m, nil
	}

	var cmd tea.Cmd
	m.viewer, cmd = m.viewer.Update(msg)
	return m, cmd
}

func (m statsModel) openHeavySession(rank int) (statsModel, tea.Cmd) {
	conversation, sessionMeta, ok := m.heavySessionTarget(rank)
	if !ok {
		return m, nil
	}
	return m, loadStatsSessionCmd(m.ctx, m.store, conversation, sessionMeta)
}

func (m statsModel) heavySessionTarget(rank int) (conv.Conversation, conv.SessionMeta, bool) {
	if m.tab != statsTabOverview || rank < 0 || rank >= len(m.snapshot.Overview.TopSessions) {
		return conv.Conversation{}, conv.SessionMeta{}, false
	}

	target := m.snapshot.Overview.TopSessions[rank]
	for _, conversation := range m.filteredConversations() {
		for _, session := range conversation.Sessions {
			if session.ID != target.SessionID {
				continue
			}
			if target.FilePath != "" && session.FilePath != "" && session.FilePath != target.FilePath {
				continue
			}
			return conversation, session, true
		}
	}
	return conv.Conversation{}, conv.SessionMeta{}, false
}

func loadStatsSessionCmd(
	ctx context.Context,
	store browserStore,
	conversation conv.Conversation,
	sessionMeta conv.SessionMeta,
) tea.Cmd {
	return func() tea.Msg {
		session, err := store.LoadSession(ctx, conversation, sessionMeta)
		if err != nil {
			return errorNotification(fmt.Sprintf("load session failed: %v", err))
		}
		return statsSessionLoadedMsg{
			conversation: conversation,
			session:      session,
		}
	}
}

func (m statsModel) openLoadedViewer(msg statsSessionLoadedMsg) statsModel {
	m.viewer = newViewerModelWithLauncher(
		msg.session,
		msg.conversation,
		m.glamourStyle,
		m.timestampFormat,
		m.width,
		m.height,
		m.launcher,
	)
	m.viewerOpen = true
	return m
}
