package canonical

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/internal/source/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreRebuildListLoadAndDeepSearch(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	copyFixtureCorpusToArchive(t, archiveDir)

	source := claude.New()
	store := New(source)

	err := store.Rebuild(context.Background(), archiveDir, conv.ProviderClaude, nil)
	require.NoError(t, err)

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.NotEmpty(t, conversations)

	var toolConversation conv.Conversation
	for _, conversation := range conversations {
		if conversation.Name == "tool-runbook" {
			toolConversation = conversation
			break
		}
	}
	require.NotZero(t, toolConversation)

	session, err := store.Load(context.Background(), archiveDir, toolConversation)
	require.NoError(t, err)
	require.NotEmpty(t, session.Messages)

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "tests passed", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.NotEmpty(t, results)
	assert.Equal(t, toolConversation.CacheKey(), results[0].CacheKey())
	assert.Contains(t, results[0].SearchPreview, "tests passed")
}

func TestStoreRebuildInvalidatesDeepSearchCache(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	copyFixtureCorpusToArchive(t, archiveDir)

	store := New(claude.New())
	require.NoError(t, store.Rebuild(context.Background(), archiveDir, conv.ProviderClaude, nil))

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.NotEmpty(t, conversations)

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "tests passed", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.NotEmpty(t, results)

	rawPath := filepath.Join(
		providerRawDir(archiveDir, conversationProvider("claude")),
		"project-a",
		"session-with-tools.jsonl",
	)
	rawData, err := os.ReadFile(rawPath)
	require.NoError(t, err)

	updated := strings.ReplaceAll(string(rawData), "tests passed", "cache reloaded")
	require.NoError(t, os.WriteFile(rawPath, []byte(updated), 0o644))

	require.NoError(t, store.Rebuild(context.Background(), archiveDir, conv.ProviderClaude, []string{rawPath}))

	conversations, err = store.List(context.Background(), archiveDir)
	require.NoError(t, err)

	results, available, err = store.DeepSearch(context.Background(), archiveDir, "tests passed", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	assert.Empty(t, results)

	results, available, err = store.DeepSearch(context.Background(), archiveDir, "cache reloaded", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.NotEmpty(t, results)
}

func fixtureCorpusDir(tb testing.TB) string {
	tb.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "claude_raw")
}

func copyFixtureCorpusToArchive(tb testing.TB, archiveDir string) {
	tb.Helper()
	copyFixtureDir(tb, fixtureCorpusDir(tb), providerRawDir(archiveDir, conversationProvider("claude")))
}

func copyFixtureDir(tb testing.TB, srcDir, dstDir string) {
	tb.Helper()

	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		dstPath := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0o644)
	})
	require.NoError(tb, err)
}
