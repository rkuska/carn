package stats

import (
	tea "charm.land/bubbletea/v2"

	conv "github.com/rkuska/carn/internal/conversation"
)

func (m statsModel) openHeavySession(rank int) (statsModel, tea.Cmd) {
	conversation, sessionMeta, ok := m.heavySessionTarget(rank)
	if !ok {
		return m, nil
	}
	return m, openSessionCmd(conversation, sessionMeta)
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
