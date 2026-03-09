package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderArchivePaths(t *testing.T) {
	t.Parallel()

	archiveDir := "/tmp/archive"

	assert.Equal(
		t,
		filepath.Join(archiveDir, "claude", "raw"),
		providerRawDir(archiveDir, conversationProviderClaude),
	)
	assert.Equal(
		t,
		filepath.Join(archiveDir, "store", "v1"),
		canonicalStoreDir(archiveDir),
	)
}

func TestRunImportPipelineSyncsSourceAndBuildsStore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	archiveDir := filepath.Join(dir, "archive")

	sourceContent := strings.Join([]string{
		makeJSONLRecord("user", "shared-session", "session-1"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T10:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"source reply"}]}}`,
		}, ""),
	}, "\n")

	writeTestFile(t, filepath.Join(sourceDir, "project-a", "session-1.jsonl"), sourceContent)

	cfg := archiveConfig{
		sourceDir:  sourceDir,
		archiveDir: archiveDir,
	}

	result, err := runImportPipeline(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.True(t, result.storeBuilt)

	rawPath := filepath.Join(providerRawDir(archiveDir, conversationProviderClaude), "project-a", "session-1.jsonl")
	rawBytes, err := os.ReadFile(rawPath)
	require.NoError(t, err)
	assert.Equal(t, sourceContent, string(rawBytes))

	repo := newDefaultConversationRepository()
	conversations, err := repo.scan(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)

	session, err := repo.load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	assert.Contains(t, renderTranscript(session, transcriptOptions{}), "source reply")

	corpus, err := repo.searchCorpus(context.Background(), archiveDir)
	require.NoError(t, err)
	assert.NotEmpty(t, corpus.units)
}

func TestRunImportPipelineWithFixtureCorpus(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	archiveDir := filepath.Join(dir, "archive")
	copyFixtureCorpusToSource(t, sourceDir)

	cfg := archiveConfig{
		sourceDir:  sourceDir,
		archiveDir: archiveDir,
	}

	result, err := runImportPipeline(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.True(t, result.storeBuilt)

	repo := newDefaultConversationRepository()
	conversations, err := repo.scan(context.Background(), archiveDir)
	require.NoError(t, err)

	names := conversationNames(conversations)
	assert.ElementsMatch(
		t,
		[]string{
			"fixture-basic",
			"legacy-format",
			"plan-session",
			"subagent-helper",
			"subagent-parent",
			"tool-runbook",
			"usage-summary",
		},
		names,
	)

	toolConv := requireConversationByName(t, conversations, "tool-runbook")
	toolSession, err := repo.load(context.Background(), archiveDir, toolConv)
	require.NoError(t, err)
	toolTranscript := renderTranscript(toolSession, transcriptOptions{})
	assertContainsAll(
		t,
		toolTranscript,
		"Inspect the main package and run the tests.",
		"Main package inspected and tests passed.",
	)
	assert.True(t, sessionHasToolCallSummary(toolSession, "go test ./..."))
	assert.True(
		t,
		sessionHasToolResultContent(toolSession, "github.com/example/carn/internal/app"),
	)

	subagentConv := requireConversationByName(t, conversations, "subagent-parent")
	subagentSession, err := repo.load(context.Background(), archiveDir, subagentConv)
	require.NoError(t, err)
	subagentTranscript := renderTranscript(subagentSession, transcriptOptions{})
	assertContainsAll(
		t,
		subagentTranscript,
		"Investigate flaky search results.",
		"Check tokenizer edge cases.",
		"Tokenizer edge case report",
		"Tokenizer investigation completed.",
	)

	corpus, err := repo.searchCorpus(context.Background(), archiveDir)
	require.NoError(t, err)
	assert.True(t, corpusContains(corpus, "Tokenizer edge case report"))
}

func TestRebuildCanonicalStoreBuildsStoreFromRawArchive(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	rawDir := providerRawDir(archiveDir, conversationProviderClaude)

	writeTestFile(t, filepath.Join(rawDir, "project-a", "session-1.jsonl"), strings.Join([]string{
		makeJSONLRecord("user", "store-session", "session-1"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T10:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"store reply"},`,
			`{"type":"tool_use","id":"tool-1","name":"Read",`,
			`"input":{"file_path":"/tmp/main.go"}}]}}`,
		}, ""),
	}, "\n"))

	require.NoError(t, rebuildCanonicalStore(context.Background(), archiveDir, conversationProviderClaude, nil))

	storeDir := canonicalStoreDir(archiveDir)
	_, err := os.Stat(filepath.Join(storeDir, "manifest.json"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(storeDir, "catalog.bin"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(storeDir, "search.bin"))
	require.NoError(t, err)

	repo := newDefaultConversationRepository()
	conversations, err := repo.scan(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)

	session, err := repo.load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	assert.Contains(t, renderTranscript(session, transcriptOptions{}), "store reply")

	corpus, err := repo.searchCorpus(context.Background(), archiveDir)
	require.NoError(t, err)
	assert.NotEmpty(t, corpus.units)
}

func TestRebuildCanonicalStoreWithFixtureCorpus(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	copyFixtureCorpusToArchive(t, archiveDir)

	require.NoError(t, rebuildCanonicalStore(context.Background(), archiveDir, conversationProviderClaude, nil))

	msg := loadSessionsCmd(context.Background(), archiveDir)()
	loaded := requireMsgType[conversationsLoadedMsg](t, msg)

	names := conversationNames(loaded.conversations)
	assert.ElementsMatch(
		t,
		[]string{
			"fixture-basic",
			"legacy-format",
			"plan-session",
			"subagent-helper",
			"subagent-parent",
			"tool-runbook",
			"usage-summary",
		},
		names,
	)
	assert.NotContains(t, names, "command-only")
	assert.True(t, loaded.deepSearchAvailable)
	assert.True(t, corpusContains(loaded.searchCorpus, "Deployment checklist summary"))

	repo := newDefaultConversationRepository()
	subagentConv := requireConversationByName(t, loaded.conversations, "subagent-parent")
	session, err := repo.load(context.Background(), archiveDir, subagentConv)
	require.NoError(t, err)
	assert.Contains(t, renderTranscript(session, transcriptOptions{}), "Tokenizer edge case report")
}

func TestRebuildCanonicalStoreKeepsExistingStoreWhenRebuildFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	rawPath := filepath.Join(
		providerRawDir(archiveDir, conversationProviderClaude),
		"project-a",
		"session-1.jsonl",
	)

	writeTestFile(t, rawPath, strings.Join([]string{
		makeJSONLRecord("user", "stable-session", "session-1"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T10:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"stable reply"}]}}`,
		}, ""),
	}, "\n"))

	require.NoError(t, rebuildCanonicalStore(context.Background(), archiveDir, conversationProviderClaude, nil))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := rebuildCanonicalStore(ctx, archiveDir, conversationProviderClaude, nil)
	require.Error(t, err)

	repo := newDefaultConversationRepository()
	conversations, err := repo.scan(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)

	session, err := repo.load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	assert.Contains(t, renderTranscript(session, transcriptOptions{}), "stable reply")
}

func TestLoadSessionsCmdUsesCatalogWhenSearchIndexIsMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	rawPath := filepath.Join(
		providerRawDir(archiveDir, conversationProviderClaude),
		"project-a",
		"session-1.jsonl",
	)

	writeTestFile(t, rawPath, strings.Join([]string{
		makeJSONLRecord("user", "missing-search", "session-1"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T10:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"search fallback"}]}}`,
		}, ""),
	}, "\n"))

	require.NoError(t, rebuildCanonicalStore(context.Background(), archiveDir, conversationProviderClaude, nil))
	require.NoError(
		t,
		os.Remove(filepath.Join(canonicalStoreDir(archiveDir), "search.bin")),
	)

	msg := loadSessionsCmd(context.Background(), archiveDir)()
	loaded := requireMsgType[conversationsLoadedMsg](t, msg)
	require.Len(t, loaded.conversations, 1)
	assert.False(t, loaded.deepSearchAvailable)
}

func TestLoadSessionsCmdUsesCatalogWhenSearchIndexIsCorrupt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	rawPath := filepath.Join(
		providerRawDir(archiveDir, conversationProviderClaude),
		"project-a",
		"session-1.jsonl",
	)

	writeTestFile(t, rawPath, strings.Join([]string{
		makeJSONLRecord("user", "corrupt-search", "session-1"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T10:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"corrupt fallback"}]}}`,
		}, ""),
	}, "\n"))

	require.NoError(t, rebuildCanonicalStore(context.Background(), archiveDir, conversationProviderClaude, nil))
	searchPath := filepath.Join(canonicalStoreDir(archiveDir), "search.bin")
	require.NoError(t, os.WriteFile(searchPath, []byte("not-a-valid-search-index"), 0o644))

	msg := loadSessionsCmd(context.Background(), archiveDir)()
	loaded := requireMsgType[conversationsLoadedMsg](t, msg)
	require.Len(t, loaded.conversations, 1)
	assert.False(t, loaded.deepSearchAvailable)
}

func TestRebuildCanonicalStoreIncrementalReusesUnchanged(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	rawDir := providerRawDir(archiveDir, conversationProviderClaude)

	pathA := filepath.Join(rawDir, "project-a", "session-a.jsonl")
	pathB := filepath.Join(rawDir, "project-b", "session-b.jsonl")

	writeTestFile(t, pathA, strings.Join([]string{
		makeJSONLRecord("user", "session-a", "session-a"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T10:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"reply A"}]}}`,
		}, ""),
	}, "\n"))
	writeTestFile(t, pathB, strings.Join([]string{
		makeJSONLRecord("user", "session-b", "session-b"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T11:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"reply B original"}]}}`,
		}, ""),
	}, "\n"))

	require.NoError(t, rebuildCanonicalStore(
		context.Background(), archiveDir, conversationProviderClaude, nil,
	))

	writeTestFile(t, pathB, strings.Join([]string{
		makeJSONLRecord("user", "session-b", "session-b"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T11:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"reply B updated"}]}}`,
		}, ""),
	}, "\n"))

	require.NoError(t, rebuildCanonicalStore(
		context.Background(), archiveDir, conversationProviderClaude, []string{pathB},
	))

	repo := newDefaultConversationRepository()
	conversations, err := repo.scan(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 2)

	for _, conv := range conversations {
		session, err := repo.load(context.Background(), archiveDir, conv)
		require.NoError(t, err)
		rendered := renderTranscript(session, transcriptOptions{})
		if conv.name == "session-a" {
			assert.Contains(t, rendered, "reply A")
		}
		if conv.name == "session-b" {
			assert.Contains(t, rendered, "reply B updated")
		}
	}

	corpus, err := repo.searchCorpus(context.Background(), archiveDir)
	require.NoError(t, err)
	assert.NotEmpty(t, corpus.units)
}

func TestRebuildCanonicalStoreIncrementalHandlesNewConversation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	rawDir := providerRawDir(archiveDir, conversationProviderClaude)

	pathA := filepath.Join(rawDir, "project-a", "session-a.jsonl")
	writeTestFile(t, pathA, strings.Join([]string{
		makeJSONLRecord("user", "session-a", "session-a"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T10:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"reply A"}]}}`,
		}, ""),
	}, "\n"))

	require.NoError(t, rebuildCanonicalStore(
		context.Background(), archiveDir, conversationProviderClaude, nil,
	))

	pathB := filepath.Join(rawDir, "project-b", "session-b.jsonl")
	writeTestFile(t, pathB, strings.Join([]string{
		makeJSONLRecord("user", "session-b", "session-b"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T11:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"reply B"}]}}`,
		}, ""),
	}, "\n"))

	require.NoError(t, rebuildCanonicalStore(
		context.Background(), archiveDir, conversationProviderClaude, []string{pathB},
	))

	repo := newDefaultConversationRepository()
	conversations, err := repo.scan(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 2)
}

func TestRebuildCanonicalStoreIncrementalFallsBackOnCorruptStore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	rawDir := providerRawDir(archiveDir, conversationProviderClaude)

	pathA := filepath.Join(rawDir, "project-a", "session-a.jsonl")
	writeTestFile(t, pathA, strings.Join([]string{
		makeJSONLRecord("user", "session-a", "session-a"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T10:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"reply A"}]}}`,
		}, ""),
	}, "\n"))

	require.NoError(t, rebuildCanonicalStore(
		context.Background(), archiveDir, conversationProviderClaude, nil,
	))

	catalogPath := filepath.Join(
		canonicalStoreDir(archiveDir), "catalog.bin",
	)
	require.NoError(t, os.WriteFile(catalogPath, []byte("corrupt"), 0o644))

	require.NoError(t, rebuildCanonicalStore(
		context.Background(), archiveDir, conversationProviderClaude, []string{pathA},
	))

	repo := newDefaultConversationRepository()
	conversations, err := repo.scan(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)

	session, err := repo.load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	assert.Contains(t, renderTranscript(session, transcriptOptions{}), "reply A")
}

func conversationNames(conversations []conversation) []string {
	names := make([]string, 0, len(conversations))
	for _, conv := range conversations {
		names = append(names, conv.name)
	}
	return names
}

func requireConversationByName(
	t testing.TB,
	conversations []conversation,
	name string,
) conversation {
	t.Helper()

	for _, conv := range conversations {
		if conv.name == name {
			return conv
		}
	}

	t.Fatalf("conversation %q not found", name)
	return conversation{}
}

func corpusContains(corpus searchCorpus, needle string) bool {
	for _, unit := range corpus.units {
		if strings.Contains(unit.text, needle) {
			return true
		}
	}
	return false
}

func sessionHasToolCallSummary(session sessionFull, want string) bool {
	for _, msg := range session.messages {
		for _, call := range msg.toolCalls {
			if call.summary == want {
				return true
			}
		}
	}
	return false
}

func sessionHasToolResultContent(session sessionFull, want string) bool {
	for _, msg := range session.messages {
		for _, result := range msg.toolResults {
			if strings.Contains(result.content, want) {
				return true
			}
		}
	}
	return false
}
