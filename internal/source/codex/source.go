package codex

import (
	"context"
	"fmt"

	conv "github.com/rkuska/carn/internal/conversation"
)

type Source struct{}

func New() Source {
	return Source{}
}

func (Source) Provider() conv.Provider {
	return conv.ProviderCodex
}

func (Source) Scan(ctx context.Context, rawDir string) ([]conv.Conversation, error) {
	conversations, err := scanRollouts(ctx, rawDir)
	if err != nil {
		return nil, fmt.Errorf("scan_scanRollouts: %w", err)
	}
	return conversations, nil
}

func (Source) Load(ctx context.Context, conversation conv.Conversation) (conv.Session, error) {
	session, err := loadConversation(ctx, conversation)
	if err != nil {
		return conv.Session{}, fmt.Errorf("load_loadConversation: %w", err)
	}
	return session, nil
}
