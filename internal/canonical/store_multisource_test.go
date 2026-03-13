package canonical

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubMultiSource struct {
	provider      conversationProvider
	conversations []conversation
	sessions      map[string]sessionFull
}

func (s stubMultiSource) Provider() conversationProvider {
	return s.provider
}

func (s stubMultiSource) Scan(context.Context, string) ([]conversation, error) {
	return s.conversations, nil
}

func (s stubMultiSource) Load(_ context.Context, conversation conversation) (sessionFull, error) {
	if session, ok := s.sessions[conversation.CacheKey()]; ok {
		return session, nil
	}
	for _, session := range s.sessions {
		if session.Meta.ID == conversation.ResumeID() {
			return session, nil
		}
	}
	return sessionFull{}, nil
}

func TestStoreRebuildAllKeepsMultipleProviders(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	claudeRawDir := providerRawDir(archiveDir, conversationProvider("claude"))
	codexRawDir := providerRawDir(archiveDir, conversationProvider("codex"))
	require.NoError(t, os.MkdirAll(filepath.Join(claudeRawDir, "project-a"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(codexRawDir, "2026", "03", "13"), 0o755))

	claudePath := filepath.Join(claudeRawDir, "project-a", "session.jsonl")
	codexPath := filepath.Join(codexRawDir, "2026", "03", "13", "rollout.jsonl")
	require.NoError(t, os.WriteFile(claudePath, []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(codexPath, []byte("{}"), 0o644))

	claudeConversation := testProviderConversation(
		conversationProvider("claude"),
		"claude-session",
		"claude-thread",
		claudePath,
	)
	codexConversation := testProviderConversation(
		conversationProvider("codex"),
		"codex-session",
		"",
		codexPath,
	)

	store := New(
		stubMultiSource{
			provider: conversationProvider("claude"),
			conversations: []conversation{
				claudeConversation,
			},
			sessions: map[string]sessionFull{
				claudeConversation.CacheKey(): {
					Meta: sessionMeta{ID: "claude-session"},
					Messages: []message{
						{Role: role("assistant"), Text: "claude content"},
					},
				},
			},
		},
		stubMultiSource{
			provider: conversationProvider("codex"),
			conversations: []conversation{
				codexConversation,
			},
			sessions: map[string]sessionFull{
				codexConversation.CacheKey(): {
					Meta: sessionMeta{ID: "codex-session"},
					Messages: []message{
						{Role: role("assistant"), Text: "codex content"},
					},
				},
			},
		},
	)

	require.NoError(t, store.RebuildAll(context.Background(), archiveDir, nil))

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 2)

	providers := []conv.Provider{
		conversations[0].Ref.Provider,
		conversations[1].Ref.Provider,
	}
	assert.ElementsMatch(t, []conv.Provider{conv.ProviderClaude, conv.ProviderCodex}, providers)

	loadedClaude, err := store.Load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	loadedCodex, err := store.Load(context.Background(), archiveDir, conversations[1])
	require.NoError(t, err)
	assert.NotEmpty(t, loadedClaude.Messages)
	assert.NotEmpty(t, loadedCodex.Messages)
	assert.Contains(t, []string{
		claudeConversation.Ref.ID,
		codexConversation.Ref.ID,
	}, conversations[0].Ref.ID)
	assert.Contains(t, []string{
		claudeConversation.Ref.ID,
		codexConversation.Ref.ID,
	}, conversations[1].Ref.ID)
}

func testProviderConversation(
	provider conversationProvider,
	sessionID string,
	name string,
	path string,
) conversation {
	return conversation{
		Ref:     conversationRef{Provider: provider, ID: sessionID},
		Name:    name,
		Project: project{DisplayName: string(provider)},
		Sessions: []sessionMeta{{
			ID:        sessionID,
			Timestamp: time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC),
			FilePath:  path,
			Project:   project{DisplayName: string(provider)},
		}},
	}
}
