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
		filepath.Join(archiveDir, "claude", "store", "v1"),
		providerStoreDir(archiveDir, conversationProviderClaude),
	)
}

func TestCollectSyncCandidatesSkipsProviderNamespaceForLegacyMigration(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")

	writeTestFile(t, filepath.Join(archiveDir, "project-a", "session-1.jsonl"), "legacy")
	writeTestFile(
		t,
		filepath.Join(providerRawDir(archiveDir, conversationProviderClaude), "project-b", "session-2.jsonl"),
		"raw",
	)

	candidates, err := collectSyncCandidates(syncRootsConfig{
		sourceDir:          archiveDir,
		destDir:            providerRawDir(archiveDir, conversationProviderClaude),
		excludeRelPrefixes: []string{"claude"},
	})
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(
		t,
		filepath.Join(archiveDir, "project-a", "session-1.jsonl"),
		candidates[0].sourcePath,
	)
	assert.Equal(
		t,
		filepath.Join(providerRawDir(archiveDir, conversationProviderClaude), "project-a", "session-1.jsonl"),
		candidates[0].destPath,
	)
	assert.Equal(t, syncStatusNew, candidates[0].status)
}

func TestRunImportPipelineMigratesLegacyThenOverridesWithSourceAndBuildsStore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	archiveDir := filepath.Join(dir, "archive")

	legacyContent := strings.Join([]string{
		makeJSONLRecord("user", "shared-session", "session-1"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T10:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"legacy reply"}]}}`,
		}, ""),
	}, "\n")
	sourceContent := strings.Join([]string{
		makeJSONLRecord("user", "shared-session", "session-1"),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2026-03-08T10:00:00Z",`,
			`"message":{"role":"assistant","model":"claude","content":[`,
			`{"type":"text","text":"source reply"}]}}`,
		}, ""),
	}, "\n")

	writeTestFile(t, filepath.Join(archiveDir, "project-a", "session-1.jsonl"), legacyContent)
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

	var statuses []syncFileStatus
	for _, file := range result.files {
		if file.destPath == rawPath {
			statuses = append(statuses, file.status)
		}
	}
	assert.Contains(t, statuses, syncStatusNew)
	assert.Contains(t, statuses, syncStatusUpdated)

	repo := newDefaultConversationRepository()
	conversations, err := repo.scan(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	assert.NotEqual(t, "session-1", conversations[0].id())

	session, err := repo.load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	assert.Contains(t, renderTranscript(session, transcriptOptions{}), "source reply")

	corpus, err := repo.searchCorpus(context.Background(), archiveDir)
	require.NoError(t, err)
	assert.NotEmpty(t, corpus.units)
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

	require.NoError(t, rebuildCanonicalStore(context.Background(), archiveDir, conversationProviderClaude))

	storeDir := providerStoreDir(archiveDir, conversationProviderClaude)
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

	require.NoError(t, rebuildCanonicalStore(context.Background(), archiveDir, conversationProviderClaude))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := rebuildCanonicalStore(ctx, archiveDir, conversationProviderClaude)
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

	require.NoError(t, rebuildCanonicalStore(context.Background(), archiveDir, conversationProviderClaude))
	require.NoError(
		t,
		os.Remove(filepath.Join(providerStoreDir(archiveDir, conversationProviderClaude), "search.bin")),
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

	require.NoError(t, rebuildCanonicalStore(context.Background(), archiveDir, conversationProviderClaude))
	searchPath := filepath.Join(providerStoreDir(archiveDir, conversationProviderClaude), "search.bin")
	require.NoError(t, os.WriteFile(searchPath, []byte("not-a-valid-search-index"), 0o644))

	msg := loadSessionsCmd(context.Background(), archiveDir)()
	loaded := requireMsgType[conversationsLoadedMsg](t, msg)
	require.Len(t, loaded.conversations, 1)
	assert.False(t, loaded.deepSearchAvailable)
}
