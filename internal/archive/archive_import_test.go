package archive

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rkuska/carn/internal/canonical"
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

func TestDedupeStrings(t *testing.T) {
	t.Parallel()

	assert.Nil(t, dedupeStrings(nil))
	assert.Equal(t, []string{"a", "b", "c"}, dedupeStrings([]string{"a", "b", "a", "c", "b"}))
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
	store := canonical.New(source)
	pipeline := New(Config{
		SourceDir:  sourceDir,
		ArchiveDir: archiveDir,
	}, source, store)

	var progress []ImportProgress
	analysis, err := pipeline.Analyze(context.Background(), func(p ImportProgress) {
		progress = append(progress, p)
	})
	require.NoError(t, err)

	assert.Equal(t, sourceDir, analysis.SourceDir)
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
	assert.Equal(t, "proj1", last.CurrentProject)
	assert.NoError(t, last.Err)
}

func TestPipelineAnalyzeMissingSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := claude.New()
	store := canonical.New(source)
	pipeline := New(Config{
		SourceDir:  filepath.Join(dir, "missing"),
		ArchiveDir: filepath.Join(dir, "archive"),
	}, source, store)

	analysis, err := pipeline.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.Zero(t, analysis.Projects)
	assert.Zero(t, analysis.FilesInspected)
	assert.Empty(t, analysis.QueuedFiles)
	assert.False(t, analysis.StoreNeedsBuild)
}

func TestPipelineAnalyzeContextCanceled(t *testing.T) {
	t.Parallel()

	source := claude.New()
	store := canonical.New(source)
	pipeline := New(Config{SourceDir: t.TempDir(), ArchiveDir: t.TempDir()}, source, store)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := pipeline.Analyze(ctx, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "analyze_ctx")
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
