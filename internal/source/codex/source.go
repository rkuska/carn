package codex

import (
	"context"
	"fmt"
	"path/filepath"

	conv "github.com/rkuska/carn/internal/conversation"
)

type Source struct{}

func New() Source {
	return Source{}
}

func (Source) Provider() conv.Provider {
	return conv.ProviderCodex
}

func (Source) SourceEnvVars() []string {
	return []string{"CARN_CODEX_SOURCE_DIR"}
}

func (Source) DefaultSourceDir(home string) string {
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".codex", "sessions")
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
