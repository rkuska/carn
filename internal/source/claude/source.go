package claude

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
	return conv.ProviderClaude
}

func (Source) SourceEnvVars() []string {
	return []string{"CARN_CLAUDE_SOURCE_DIR", "CARN_SOURCE_DIR"}
}

func (Source) DefaultSourceDir(home string) string {
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".claude", "projects")
}

func (Source) Scan(ctx context.Context, rawDir string) ([]conv.Conversation, error) {
	sessions, err := scanSessions(ctx, rawDir)
	if err != nil {
		return nil, fmt.Errorf("scanSessions: %w", err)
	}
	return groupConversations(sessions), nil
}

func (Source) Load(ctx context.Context, conversation conv.Conversation) (conv.Session, error) {
	session, err := parseConversationWithSubagents(ctx, conversation)
	if err != nil {
		return conv.Session{}, fmt.Errorf("parseConversationWithSubagents: %w", err)
	}
	return session, nil
}
