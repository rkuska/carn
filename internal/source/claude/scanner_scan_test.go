package claude

import (
	"context"
	"path/filepath"
	"testing"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceScanGroupsFixtureCorpusIntoConversations(t *testing.T) {
	t.Parallel()

	baseDir := copyScannerFixtureCorpus(t)

	conversations, err := New().Scan(context.Background(), baseDir)
	require.NoError(t, err)
	require.NotEmpty(t, conversations)

	var sawSubagent bool
	for _, conversation := range conversations {
		if conversation.IsSubagent() {
			sawSubagent = true
		}
	}

	assert.True(t, sawSubagent)
}

func TestSourceScanOwnsClaudeConversationRefs(t *testing.T) {
	t.Parallel()

	baseDir := copyScannerFixtureCorpus(t)

	conversations, err := New().Scan(context.Background(), baseDir)
	require.NoError(t, err)
	require.NotEmpty(t, conversations)

	var grouped conv.Conversation
	var subagent conv.Conversation
	for _, conversation := range conversations {
		switch {
		case conversation.Name == "fixture-basic":
			grouped = conversation
		case conversation.IsSubagent():
			subagent = conversation
		}
	}

	require.NotEmpty(t, grouped.Ref.ID)
	assert.Equal(t, conv.ProviderClaude, grouped.Ref.Provider)
	assert.Equal(t, "group:project-a:fixture-basic", grouped.Ref.ID)

	require.NotEmpty(t, subagent.Ref.ID)
	assert.Equal(t, conv.ProviderClaude, subagent.Ref.Provider)
	assert.Equal(t, "path:project-a/session-with-subagent/subagents/agent-1.jsonl", subagent.Ref.ID)
}

func TestSourceOwnsSyncCandidates(t *testing.T) {
	t.Parallel()

	sourceDir := copyScannerFixtureCorpus(t)
	rawDir := t.TempDir()
	backend := New()

	candidates, err := backend.SyncCandidates(context.Background(), sourceDir, rawDir)
	require.NoError(t, err)
	require.NotEmpty(t, candidates)
	assert.Equal(t,
		filepath.Join(rawDir, "project-a", "session-basic.jsonl"),
		candidates[0].DestPath,
	)
}

func TestSourceScanSkipsCommandOnlySession(t *testing.T) {
	t.Parallel()

	baseDir := copyScannerFixtureCorpus(t)

	conversations, err := New().Scan(context.Background(), baseDir)
	require.NoError(t, err)

	for _, conversation := range conversations {
		assert.NotEqual(t, "command-only", conversation.Name)
	}
}
