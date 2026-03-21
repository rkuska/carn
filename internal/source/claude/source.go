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
