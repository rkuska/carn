package stats

import (
	tea "charm.land/bubbletea/v2"

	conv "github.com/rkuska/carn/internal/conversation"
)

type CloseRequestedMsg struct{}

type OpenSessionRequestedMsg struct {
	Conversation conv.Conversation
	SessionMeta  conv.SessionMeta
}

func closeStatsCmd() tea.Cmd {
	return func() tea.Msg {
		return CloseRequestedMsg{}
	}
}

func openSessionCmd(conversation conv.Conversation, sessionMeta conv.SessionMeta) tea.Cmd {
	return func() tea.Msg {
		return OpenSessionRequestedMsg{
			Conversation: conversation,
			SessionMeta:  sessionMeta,
		}
	}
}
