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
	"github.com/rkuska/carn/internal/source/codex"
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

func TestStoreRebuildAddsClaudeSessionToExistingSlugGroup(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(claude.New())
	claudeRawDir := providerRawDir(archiveDir, conversationProvider("claude"))

	writeTestConversation(t, claudeRawDir, "project-a", "session-1", "demo", []string{
		"first answer",
	})
	require.NoError(t, store.Rebuild(context.Background(), archiveDir, conv.ProviderClaude, nil))

	second := writeTestConversation(t, claudeRawDir, "project-a", "session-2", "demo", []string{
		"second answer",
	})
	require.NoError(t, store.Rebuild(
		context.Background(),
		archiveDir,
		conv.ProviderClaude,
		[]string{second.Sessions[0].FilePath},
	))

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	assert.Equal(t, "claude:group:project-a:demo", conversations[0].CacheKey())
	require.Len(t, conversations[0].Sessions, 2)

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "second answer", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.Len(t, results, 1)
	assert.Equal(t, conversations[0].CacheKey(), results[0].CacheKey())
}

func TestStoreRebuildMovesClaudeConversationWhenSlugChanges(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(claude.New())
	claudeRawDir := providerRawDir(archiveDir, conversationProvider("claude"))

	convValue := writeTestConversation(t, claudeRawDir, "project-a", "session-1", "old-slug", []string{
		"before rename",
	})
	require.NoError(t, store.Rebuild(context.Background(), archiveDir, conv.ProviderClaude, nil))

	rawPath := convValue.Sessions[0].FilePath
	rawData, err := os.ReadFile(rawPath)
	require.NoError(t, err)
	updated := strings.ReplaceAll(string(rawData), `"slug":"old-slug"`, `"slug":"new-slug"`)
	updated = strings.ReplaceAll(updated, "before rename", "after rename")
	require.NoError(t, os.WriteFile(rawPath, []byte(updated), 0o644))

	require.NoError(t, store.Rebuild(context.Background(), archiveDir, conv.ProviderClaude, []string{rawPath}))

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	assert.Equal(t, "claude:group:project-a:new-slug", conversations[0].CacheKey())
	assert.Equal(t, "new-slug", conversations[0].Name)

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "after rename", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.Len(t, results, 1)
	assert.Equal(t, conversations[0].CacheKey(), results[0].CacheKey())
}

func TestStoreCodexLoadPreservesHiddenSystemAndGroupedSubagents(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	copyFixtureDir(t, codexFixtureCorpusDir(t), providerRawDir(archiveDir, conversationProvider("codex")))

	store := New(codex.New())
	require.NoError(t, store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, nil))

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 2)

	var mainConversation conv.Conversation
	for _, conversation := range conversations {
		if conversation.ID() == "019cexample-main" {
			mainConversation = conversation
			break
		}
	}
	require.NotZero(t, mainConversation)
	assert.Equal(t, "019cexample-main", mainConversation.ResumeID())
	assert.Equal(t, 1, mainConversation.PartCount())
	require.Len(t, mainConversation.Sessions, 2)
	assert.True(t, mainConversation.Sessions[1].IsSubagent)

	session, err := store.Load(context.Background(), archiveDir, mainConversation)
	require.NoError(t, err)
	require.Len(t, session.Messages, 8)
	assert.Equal(t, conv.RoleSystem, session.Messages[0].Role)
	assert.Equal(t, conv.MessageVisibilityHiddenSystem, session.Messages[0].Visibility)
	assert.Equal(t, "Inspect the parser.", session.Messages[6].Text)
	assert.Equal(t, "Parser inspected.", session.Messages[7].Text)
}

func TestStoreDeepSearchSkipsHiddenSystemMessages(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	copyFixtureDir(t, codexFixtureCorpusDir(t), providerRawDir(archiveDir, conversationProvider("codex")))

	store := New(codex.New())
	require.NoError(t, store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, nil))

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "Filesystem sandboxing", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	assert.Empty(t, results)

	results, available, err = store.DeepSearch(context.Background(), archiveDir, "Parser inspected", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.Len(t, results, 1)
	assert.Equal(t, "019cexample-main", results[0].ID())
}

func TestStoreRebuildUpdatesParentConversationWhenCodexChildRolloutChanges(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	copyFixtureDir(t, codexFixtureCorpusDir(t), providerRawDir(archiveDir, conversationProvider("codex")))

	store := New(codex.New())
	require.NoError(t, store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, nil))

	childPath := filepath.Join(
		providerRawDir(archiveDir, conversationProvider("codex")),
		"2026",
		"03",
		"13",
		"rollout-2026-03-13T10-10-00-019cexample-child.jsonl",
	)
	rawData, err := os.ReadFile(childPath)
	require.NoError(t, err)
	updated := strings.ReplaceAll(string(rawData), "Parser inspected.", "Parser revised.")
	require.NoError(t, os.WriteFile(childPath, []byte(updated), 0o644))

	require.NoError(t, store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, []string{childPath}))

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "Parser revised", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.Len(t, results, 1)
	assert.Equal(t, "019cexample-main", results[0].ID())

	session, err := store.Load(context.Background(), archiveDir, results[0])
	require.NoError(t, err)
	assert.Equal(t, "Parser revised.", session.Messages[len(session.Messages)-1].Text)
}

func TestStoreRebuildUpdatesParentConversationWhenCodexChildRolloutIsDeleted(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	copyFixtureDir(t, codexFixtureCorpusDir(t), providerRawDir(archiveDir, conversationProvider("codex")))

	store := New(codex.New())
	require.NoError(t, store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, nil))

	childPath := filepath.Join(
		providerRawDir(archiveDir, conversationProvider("codex")),
		"2026",
		"03",
		"13",
		"rollout-2026-03-13T10-10-00-019cexample-child.jsonl",
	)
	require.NoError(t, os.Remove(childPath))
	require.NoError(t, store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, []string{childPath}))

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)

	var mainConversation conv.Conversation
	for _, conversation := range conversations {
		if conversation.ID() == "019cexample-main" {
			mainConversation = conversation
			break
		}
	}
	require.NotZero(t, mainConversation)
	require.Len(t, mainConversation.Sessions, 1)

	session, err := store.Load(context.Background(), archiveDir, mainConversation)
	require.NoError(t, err)
	require.Len(t, session.Messages, 6)
	assert.True(t, session.Messages[4].IsAgentDivider)
	assert.Equal(t, "Planck is inspecting the parser.", session.Messages[4].Text)
}

func fixtureCorpusDir(tb testing.TB) string {
	tb.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "claude_raw")
}

func codexFixtureCorpusDir(tb testing.TB) string {
	tb.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "codex_raw")
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
