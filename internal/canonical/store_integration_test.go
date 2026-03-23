package canonical

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
	"github.com/rkuska/carn/internal/source/claude"
	"github.com/rkuska/carn/internal/source/codex"
)

func TestStoreRebuildListLoadAndDeepSearch(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	copyFixtureCorpusToArchive(t, archiveDir)

	source := claude.New()
	store := New(source)

	_, err := store.Rebuild(context.Background(), archiveDir, conv.ProviderClaude, nil)
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
	_, err := store.Rebuild(context.Background(), archiveDir, conv.ProviderClaude, nil)
	require.NoError(t, err)

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.NotEmpty(t, conversations)

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "tests passed", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.NotEmpty(t, results)

	rawPath := filepath.Join(
		src.ProviderRawDir(archiveDir, conversationProvider("claude")),
		"project-a",
		"session-with-tools.jsonl",
	)
	rawData, err := os.ReadFile(rawPath)
	require.NoError(t, err)

	updated := strings.ReplaceAll(string(rawData), "tests passed", "cache reloaded")
	require.NoError(t, os.WriteFile(rawPath, []byte(updated), 0o644))

	_, err = store.Rebuild(context.Background(), archiveDir, conv.ProviderClaude, []string{rawPath})
	require.NoError(t, err)

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
	claudeRawDir := src.ProviderRawDir(archiveDir, conversationProvider("claude"))

	writeTestConversation(t, claudeRawDir, "project-a", "session-1", "demo", []string{
		"first answer",
	})
	_, err := store.Rebuild(context.Background(), archiveDir, conv.ProviderClaude, nil)
	require.NoError(t, err)

	second := writeTestConversation(t, claudeRawDir, "project-a", "session-2", "demo", []string{
		"second answer",
	})
	_, err = store.Rebuild(
		context.Background(),
		archiveDir,
		conv.ProviderClaude,
		[]string{second.Sessions[0].FilePath},
	)
	require.NoError(t, err)

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
	claudeRawDir := src.ProviderRawDir(archiveDir, conversationProvider("claude"))

	convValue := writeTestConversation(t, claudeRawDir, "project-a", "session-1", "old-slug", []string{
		"before rename",
	})
	_, err := store.Rebuild(context.Background(), archiveDir, conv.ProviderClaude, nil)
	require.NoError(t, err)

	rawPath := convValue.Sessions[0].FilePath
	rawData, err := os.ReadFile(rawPath)
	require.NoError(t, err)
	updated := strings.ReplaceAll(string(rawData), `"slug":"old-slug"`, `"slug":"new-slug"`)
	updated = strings.ReplaceAll(updated, "before rename", "after rename")
	require.NoError(t, os.WriteFile(rawPath, []byte(updated), 0o644))

	_, err = store.Rebuild(context.Background(), archiveDir, conv.ProviderClaude, []string{rawPath})
	require.NoError(t, err)

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

func TestStoreIncrementalRebuildMatchesFullRebuildForChangedClaudeConversation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	incrementalArchive := t.TempDir()
	fullArchive := t.TempDir()

	incrementalRawDir := src.ProviderRawDir(incrementalArchive, conversationProvider("claude"))
	fullRawDir := src.ProviderRawDir(fullArchive, conversationProvider("claude"))

	writeTestConversation(t, incrementalRawDir, "project-a", "session-1", "keep", []string{
		"keep answer",
	})
	changedIncremental := writeTestConversation(t, incrementalRawDir, "project-a", "session-2", "change", []string{
		"before update",
	})

	writeTestConversation(t, fullRawDir, "project-a", "session-1", "keep", []string{
		"keep answer",
	})
	writeTestConversation(t, fullRawDir, "project-a", "session-2", "change", []string{
		"before update",
	})

	incrementalStore := New(claude.New())
	fullStore := New(claude.New())
	_, err := incrementalStore.Rebuild(ctx, incrementalArchive, conv.ProviderClaude, nil)
	require.NoError(t, err)
	_, err = fullStore.Rebuild(ctx, fullArchive, conv.ProviderClaude, nil)
	require.NoError(t, err)

	changedFullPath := filepath.Join(fullRawDir, "project-a", "session-2.jsonl")
	replaceClaudeAssistantText(t, changedIncremental.Sessions[0].FilePath, "before update", "after update")
	replaceClaudeAssistantText(t, changedFullPath, "before update", "after update")

	_, err = incrementalStore.Rebuild(
		ctx,
		incrementalArchive,
		conv.ProviderClaude,
		[]string{changedIncremental.Sessions[0].FilePath},
	)
	require.NoError(t, err)
	_, err = fullStore.Rebuild(ctx, fullArchive, conv.ProviderClaude, nil)
	require.NoError(t, err)

	incrementalConversations, err := incrementalStore.List(ctx, incrementalArchive)
	require.NoError(t, err)
	fullConversations, err := fullStore.List(ctx, fullArchive)
	require.NoError(t, err)
	assert.Equal(t, conversationListSnapshot(fullConversations), conversationListSnapshot(incrementalConversations))

	changedByKey := make(map[string]conv.Conversation, len(fullConversations))
	for _, conversation := range fullConversations {
		changedByKey[conversation.CacheKey()] = conversation
	}
	for _, conversation := range incrementalConversations {
		fullConversation, ok := changedByKey[conversation.CacheKey()]
		require.True(t, ok)

		incrementalSession, loadErr := incrementalStore.Load(ctx, incrementalArchive, conversation)
		require.NoError(t, loadErr)
		fullSession, loadErr := fullStore.Load(ctx, fullArchive, fullConversation)
		require.NoError(t, loadErr)

		assert.Equal(t, snapshotSession(fullSession), snapshotSession(incrementalSession))
	}

	incrementalResults, available, err := incrementalStore.DeepSearch(
		ctx,
		incrementalArchive,
		"after update",
		incrementalConversations,
	)
	require.NoError(t, err)
	assert.True(t, available)

	fullResults, available, err := fullStore.DeepSearch(
		ctx,
		fullArchive,
		"after update",
		fullConversations,
	)
	require.NoError(t, err)
	assert.True(t, available)
	assert.Equal(t, snapshotSearchResults(fullResults), snapshotSearchResults(incrementalResults))

	incrementalOldResults, available, err := incrementalStore.DeepSearch(
		ctx,
		incrementalArchive,
		"before update",
		incrementalConversations,
	)
	require.NoError(t, err)
	assert.True(t, available)

	fullOldResults, available, err := fullStore.DeepSearch(
		ctx,
		fullArchive,
		"before update",
		fullConversations,
	)
	require.NoError(t, err)
	assert.True(t, available)
	assert.Equal(t, snapshotSearchResults(fullOldResults), snapshotSearchResults(incrementalOldResults))
}

func TestStoreCodexLoadPreservesHiddenSystemAndGroupedSubagents(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	copyFixtureDir(t, codexFixtureCorpusDir(t), src.ProviderRawDir(archiveDir, conversationProvider("codex")))

	store := New(codex.New())
	_, err := store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, nil)
	require.NoError(t, err)

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 3)

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

func TestStoreCodexLoadPreservesHiddenThinkingAndDoesNotIndexViewerNote(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	copyFixtureDir(t, codexFixtureCorpusDir(t), src.ProviderRawDir(archiveDir, conversationProvider("codex")))

	store := New(codex.New())
	_, err := store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, nil)
	require.NoError(t, err)

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)

	var hiddenConversation conv.Conversation
	for _, conversation := range conversations {
		if conversation.ID() == "019cexample-hidden" {
			hiddenConversation = conversation
			break
		}
	}
	require.NotZero(t, hiddenConversation)

	session, err := store.Load(context.Background(), archiveDir, hiddenConversation)
	require.NoError(t, err)
	require.Len(t, session.Messages, 4)
	assert.True(t, session.Messages[1].HasHiddenThinking)
	assert.Empty(t, session.Messages[1].Thinking)

	results, available, err := store.DeepSearch(
		context.Background(),
		archiveDir,
		"Codex recorded reasoning for this reply",
		conversations,
	)
	require.NoError(t, err)
	assert.True(t, available)
	assert.Empty(t, results)
}

func TestStoreCodexLoadPreservesReconstructedTurnUsage(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	rawDir := src.ProviderRawDir(archiveDir, conversationProvider("codex"))
	writeCodexRolloutFixture(t, rawDir, "2026/03/16/rollout-2026-03-16T10-00-00-usage.jsonl", []map[string]any{
		{
			"timestamp": "2026-03-16T10:00:00Z",
			"type":      "session_meta",
			"payload": map[string]any{
				"id":             "usage-thread",
				"timestamp":      "2026-03-16T10:00:00Z",
				"cwd":            "/workspace/project",
				"cli_version":    "0.114.0",
				"model_provider": "openai",
				"git":            map[string]any{"branch": "main"},
			},
		},
		{
			"timestamp": "2026-03-16T10:00:01Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type":    "user_message",
				"message": "Explain the parser.",
			},
		},
		{
			"timestamp": "2026-03-16T10:00:02Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type": "message",
				"role": "assistant",
				"content": []map[string]any{
					{"type": "output_text", "text": "Parser updated."},
				},
			},
		},
		{
			"timestamp": "2026-03-16T10:00:03Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type": "token_count",
				"info": map[string]any{
					"total_token_usage": map[string]any{
						"input_tokens":            500,
						"cached_input_tokens":     50,
						"output_tokens":           140,
						"reasoning_output_tokens": 10,
					},
					"last_token_usage": map[string]any{
						"input_tokens":            120,
						"cached_input_tokens":     15,
						"output_tokens":           30,
						"reasoning_output_tokens": 5,
					},
				},
			},
		},
	})

	store := New(codex.New())
	_, err := store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, nil)
	require.NoError(t, err)

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)

	session, err := store.Load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	require.Len(t, session.Messages, 2)
	assert.Equal(t, conv.TokenUsage{
		InputTokens:          120,
		CacheReadInputTokens: 15,
		OutputTokens:         35,
	}, session.Messages[1].Usage)
	assert.Equal(t, conv.TokenUsage{
		InputTokens:          500,
		CacheReadInputTokens: 50,
		OutputTokens:         150,
	}, session.Meta.TotalUsage)
}

func TestStoreDeepSearchSkipsHiddenSystemMessages(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	copyFixtureDir(t, codexFixtureCorpusDir(t), src.ProviderRawDir(archiveDir, conversationProvider("codex")))

	store := New(codex.New())
	_, err := store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, nil)
	require.NoError(t, err)

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
	copyFixtureDir(t, codexFixtureCorpusDir(t), src.ProviderRawDir(archiveDir, conversationProvider("codex")))

	store := New(codex.New())
	_, err := store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, nil)
	require.NoError(t, err)

	childPath := filepath.Join(
		src.ProviderRawDir(archiveDir, conversationProvider("codex")),
		"2026",
		"03",
		"13",
		"rollout-2026-03-13T10-10-00-019cexample-child.jsonl",
	)
	rawData, err := os.ReadFile(childPath)
	require.NoError(t, err)
	updated := strings.ReplaceAll(string(rawData), "Parser inspected.", "Parser revised.")
	require.NoError(t, os.WriteFile(childPath, []byte(updated), 0o644))

	_, err = store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, []string{childPath})
	require.NoError(t, err)

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
	copyFixtureDir(t, codexFixtureCorpusDir(t), src.ProviderRawDir(archiveDir, conversationProvider("codex")))

	store := New(codex.New())
	_, err := store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, nil)
	require.NoError(t, err)

	childPath := filepath.Join(
		src.ProviderRawDir(archiveDir, conversationProvider("codex")),
		"2026",
		"03",
		"13",
		"rollout-2026-03-13T10-10-00-019cexample-child.jsonl",
	)
	require.NoError(t, os.Remove(childPath))
	_, err = store.Rebuild(context.Background(), archiveDir, conv.ProviderCodex, []string{childPath})
	require.NoError(t, err)

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
	copyFixtureDir(tb, fixtureCorpusDir(tb), src.ProviderRawDir(archiveDir, conversationProvider("claude")))
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

func writeCodexRolloutFixture(tb testing.TB, rawDir, name string, lines []map[string]any) {
	tb.Helper()

	encoded := make([]byte, 0, len(lines)*128)
	for i, line := range lines {
		raw, err := json.Marshal(line)
		require.NoError(tb, err)
		encoded = append(encoded, raw...)
		if i < len(lines)-1 {
			encoded = append(encoded, '\n')
		}
	}

	path := filepath.Join(rawDir, name)
	require.NoError(tb, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(tb, os.WriteFile(path, encoded, 0o644))
}

func replaceClaudeAssistantText(t *testing.T, path, oldText, newText string) {
	t.Helper()

	rawData, err := os.ReadFile(path)
	require.NoError(t, err)
	updated := strings.ReplaceAll(string(rawData), oldText, newText)
	require.NoError(t, os.WriteFile(path, []byte(updated), 0o644))
}

type conversationSnapshot struct {
	cacheKey          string
	name              string
	firstMessage      string
	project           string
	sessionCount      int
	planCount         int
	totalMessages     int
	totalMain         int
	totalInputTokens  int
	totalOutputTokens int
}

func conversationListSnapshot(conversations []conv.Conversation) []conversationSnapshot {
	snapshots := make([]conversationSnapshot, 0, len(conversations))
	for _, conversation := range conversations {
		usage := conversation.TotalTokenUsage()
		snapshots = append(snapshots, conversationSnapshot{
			cacheKey:          conversation.CacheKey(),
			name:              conversation.Name,
			firstMessage:      conversation.FirstMessage(),
			project:           conversation.Project.DisplayName,
			sessionCount:      len(conversation.Sessions),
			planCount:         conversation.PlanCount,
			totalMessages:     conversation.TotalMessageCount(),
			totalMain:         conversation.MainMessageCount(),
			totalInputTokens:  usage.InputTokens,
			totalOutputTokens: usage.OutputTokens,
		})
	}
	return snapshots
}

type sessionStateSnapshot struct {
	id               string
	slug             string
	project          string
	messageCount     int
	mainMessageCount int
	messages         []conv.Message
}

func snapshotSession(session conv.Session) sessionStateSnapshot {
	return sessionStateSnapshot{
		id:               session.Meta.ID,
		slug:             session.Meta.Slug,
		project:          session.Meta.Project.DisplayName,
		messageCount:     session.Meta.MessageCount,
		mainMessageCount: session.Meta.MainMessageCount,
		messages:         session.Messages,
	}
}

type searchResultStateSnapshot struct {
	cacheKey string
	preview  string
}

func snapshotSearchResults(results []conv.Conversation) []searchResultStateSnapshot {
	snapshots := make([]searchResultStateSnapshot, 0, len(results))
	for _, result := range results {
		snapshots = append(snapshots, searchResultStateSnapshot{
			cacheKey: result.CacheKey(),
			preview:  result.SearchPreview,
		})
	}
	return snapshots
}
