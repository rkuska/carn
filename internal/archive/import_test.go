package archive

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rkuska/carn/internal/canonical"
	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
	"github.com/rkuska/carn/internal/source/claude"
)

func TestImportAnalysisHelpers(t *testing.T) {
	t.Parallel()

	assert.True(t, ImportAnalysis{QueuedFiles: []string{"/a.jsonl"}}.NeedsSync())
	assert.True(t, ImportAnalysis{StoreNeedsBuild: true}.NeedsSync())
	assert.False(t, ImportAnalysis{}.NeedsSync())
	assert.False(t, ImportAnalysis{Err: assert.AnError}.NeedsSync())
	assert.Equal(t, 2, ImportAnalysis{QueuedFiles: []string{"a", "b"}}.QueuedFileCount())
}

func TestPipelineAnalyze(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	archiveDir := filepath.Join(dir, "archive")
	projDir := filepath.Join(sourceDir, "proj1")

	writeTestFile(t, filepath.Join(projDir, "session-1.jsonl"), makeJSONLRecord("user", "feat-a", "id1"))
	writeTestFile(t, filepath.Join(projDir, "session-2.jsonl"), makeJSONLRecord("user", "feat-b", "id2"))

	source := claude.New()
	store := canonical.New(nil, source)
	pipeline := New(Config{
		SourceDirs: map[conv.Provider]string{conv.ProviderClaude: sourceDir},
		ArchiveDir: archiveDir,
	}, store, source)

	var progress []ImportProgress
	analysis, err := pipeline.Analyze(context.Background(), func(p ImportProgress) {
		progress = append(progress, p)
	})
	require.NoError(t, err)

	assert.Equal(t, archiveDir, analysis.ArchiveDir)
	assert.Equal(t, 1, analysis.Projects)
	assert.Equal(t, 2, analysis.FilesInspected)
	assert.Equal(t, 2, analysis.Conversations)
	assert.Equal(t, 2, analysis.NewConversations)
	assert.Zero(t, analysis.ToUpdate)
	assert.Zero(t, analysis.UpToDate)
	assert.Len(t, analysis.QueuedFiles, 2)
	assert.True(t, analysis.StoreNeedsBuild)
	assert.NoError(t, analysis.Err)

	require.NotEmpty(t, progress)
	last := progress[len(progress)-1]
	assert.Equal(t, 1, last.ProjectsCompleted)
	assert.Equal(t, 1, last.ProjectsTotal)
	assert.Equal(t, 2, last.FilesInspected)
	assert.Equal(t, 2, last.Conversations)
	assert.Equal(t, 2, last.NewConversations)
	assert.Equal(t, "claude / proj1", last.CurrentProject)
	assert.NoError(t, last.Err)
}

func TestPipelineAnalyzeDedupesQueuedFilesWithoutReordering(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pipeline := New(
		Config{
			SourceDirs: map[conv.Provider]string{
				conv.ProviderClaude: filepath.Join(dir, "claude"),
				conv.ProviderCodex:  filepath.Join(dir, "codex"),
			},
			ArchiveDir: filepath.Join(dir, "archive"),
		},
		canonical.New(nil),
		stubBackend{
			provider: conv.ProviderClaude,
			analysis: src.Analysis{
				SyncCandidates: []string{"b.jsonl", "a.jsonl", "b.jsonl"},
			},
		},
		stubBackend{
			provider: conv.ProviderCodex,
			analysis: src.Analysis{
				SyncCandidates: []string{"c.jsonl", "a.jsonl"},
			},
		},
	)

	analysis, err := pipeline.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"b.jsonl", "a.jsonl", "c.jsonl"}, analysis.QueuedFiles)
}

func TestPipelineAnalyzeMissingSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := claude.New()
	store := canonical.New(nil, source)
	pipeline := New(Config{
		SourceDirs: map[conv.Provider]string{conv.ProviderClaude: filepath.Join(dir, "missing")},
		ArchiveDir: filepath.Join(dir, "archive"),
	}, store, source)

	analysis, err := pipeline.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.Zero(t, analysis.Projects)
	assert.Zero(t, analysis.FilesInspected)
	assert.Empty(t, analysis.QueuedFiles)
	assert.True(t, analysis.StoreNeedsBuild)
}

func TestPipelineAnalyzeContextCanceled(t *testing.T) {
	t.Parallel()

	source := claude.New()
	store := canonical.New(nil, source)
	pipeline := New(Config{
		SourceDirs: map[conv.Provider]string{conv.ProviderClaude: t.TempDir()},
		ArchiveDir: t.TempDir(),
	}, store, source)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := pipeline.Analyze(ctx, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "analyze_ctx")
}

func TestPipelineRunReportsSyncActivities(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	archiveDir := filepath.Join(dir, "archive")
	projDir := filepath.Join(sourceDir, "proj1")

	writeTestFile(t, filepath.Join(projDir, "session-1.jsonl"), makeJSONLRecord("user", "feat-a", "id1"))

	source := claude.New()
	store := canonical.New(nil, source)
	pipeline := New(Config{
		SourceDirs: map[conv.Provider]string{conv.ProviderClaude: sourceDir},
		ArchiveDir: archiveDir,
	}, store, source)

	var progress []SyncProgress
	result, err := pipeline.Run(context.Background(), func(p SyncProgress) {
		progress = append(progress, p)
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Copied)
	assert.True(t, result.StoreBuilt)
	require.Len(t, progress, 2)
	assert.Equal(t, SyncActivitySyncingFiles, progress[0].Activity)
	assert.Equal(t, "session-1.jsonl", progress[0].File)
	assert.Equal(t, SyncActivityRebuildingStore, progress[1].Activity)
	assert.Empty(t, progress[1].File)
}

func TestPipelineRunBuildsStoreWhenArchiveIsEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	source := claude.New()
	store := canonical.New(nil, source)
	pipeline := New(Config{
		SourceDirs: map[conv.Provider]string{conv.ProviderClaude: filepath.Join(dir, "missing")},
		ArchiveDir: archiveDir,
	}, store, source)

	result, err := pipeline.Run(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, result.StoreBuilt)

	needsRebuild, err := store.NeedsRebuild(context.Background(), archiveDir)
	require.NoError(t, err)
	assert.False(t, needsRebuild)
}

func TestPipelineRunReturnsMalformedDataWarningsWithoutFailing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	archiveDir := filepath.Join(dir, "archive")
	source := claude.New()
	store := canonical.New(nil, source)
	pipeline := New(Config{
		SourceDirs: map[conv.Provider]string{conv.ProviderClaude: sourceDir},
		ArchiveDir: archiveDir,
	}, store, source)

	writeTestFile(
		t,
		filepath.Join(sourceDir, "proj1", "session-1.jsonl"),
		strings.Join([]string{
			makeJSONLRecord("user", "demo", "session-1"),
			`{"type":"assistant","sessionId":"session-1","timestamp":"2026-03-08T10:01:00Z",` +
				`"message":{"role":"assistant","model":"claude","content":[{"type":"text","text":"before corruption"}]}}`,
		}, "\n"),
	)

	_, err := pipeline.Run(ctx, nil)
	require.NoError(t, err)

	writeTestFile(
		t,
		filepath.Join(sourceDir, "proj1", "session-1.jsonl"),
		strings.Join([]string{
			makeJSONLRecord("user", "demo", "session-1"),
			`{"type":"assistant","sessionId":"session-1","timestamp":"2026-03-08T10:01:00Z",` +
				`"message":{"role":"assistant","model":"claude","content":[{"type":"text","text":"after corruption"}}`,
		}, "\n"),
	)

	result, err := pipeline.Run(ctx, nil)
	require.NoError(t, err)
	assert.True(t, result.StoreBuilt)
	assert.Equal(t, 1, result.MalformedData.Report(conv.ProviderClaude).Count())

	needsRebuild, err := store.NeedsRebuild(ctx, archiveDir)
	require.NoError(t, err)
	assert.False(t, needsRebuild)

	conversations, err := store.List(ctx, archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)

	session, err := store.Load(ctx, archiveDir, conversations[0])
	require.NoError(t, err)
	require.Len(t, session.Messages, 2)
	assert.Equal(t, "before corruption", session.Messages[1].Text)
}

func makeJSONLRecord(recordType, slug, sessionID string) string {
	rec := map[string]any{
		"type":      recordType,
		"sessionId": sessionID,
		"slug":      slug,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	}

	if recordType == "user" {
		rec["message"] = map[string]any{
			"role":    "user",
			"content": "test message",
		}
	} else {
		rec["message"] = map[string]any{
			"role":  "assistant",
			"model": "claude",
			"content": []map[string]any{
				{"type": "text", "text": "response"},
			},
		}
	}

	raw, err := json.Marshal(rec)
	if err != nil {
		panic(err)
	}
	return string(raw)
}
