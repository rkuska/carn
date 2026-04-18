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

type statsSessionStore interface {
	LoadSession(ctx context.Context, conversation conv.Conversation, sessionMeta conv.SessionMeta) (conv.Session, error)
}

func loadStatsSessionCmd(
	ctx context.Context,
	store statsSessionStore,
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
