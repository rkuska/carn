package claude

import (
	"context"
	"fmt"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

type Source struct{}

func New() Source {
	return Source{}
}

func (Source) Provider() conv.Provider {
	return conv.ProviderClaude
}

func (Source) Scan(ctx context.Context, rawDir string) (src.ScanResult, error) {
	sessions, drift, err := scanSessions(ctx, rawDir)
	if err != nil {
		return src.ScanResult{}, fmt.Errorf("scanSessions: %w", err)
	}
	drift.Log(ctx, conv.ProviderClaude)
	return src.ScanResult{
		Conversations: groupConversations(sessions),
		Drift:         drift,
	}, nil
}

func (Source) Load(ctx context.Context, conversation conv.Conversation) (conv.Session, error) {
	session, err := parseConversationWithSubagents(ctx, conversation)
	if err != nil {
		return conv.Session{}, fmt.Errorf("parseConversationWithSubagents: %w", err)
	}
	return session, nil
}

func (s Source) LoadSession(
	ctx context.Context,
	conversation conv.Conversation,
	meta conv.SessionMeta,
) (conv.Session, error) {
	session, err := s.Load(ctx, singleSessionConversation(conversation, meta))
	if err != nil {
		return conv.Session{}, fmt.Errorf("load: %w", err)
	}
	return session, nil
}

func singleSessionConversation(conversation conv.Conversation, meta conv.SessionMeta) conv.Conversation {
	return conv.Conversation{
		Ref:       conv.Ref{Provider: conversation.Ref.Provider, ID: meta.ID},
		Name:      conversation.Name,
		Project:   conversation.Project,
		Sessions:  []conv.SessionMeta{meta},
		PlanCount: conversation.PlanCount,
	}
}
