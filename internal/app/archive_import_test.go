package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractSessionSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		wantSlug string
		wantErr  bool
	}{
		{
			name:     "extracts slug from first user record",
			content:  makeJSONLRecord("user", "test-slug", "session-1"),
			wantSlug: "test-slug",
		},
		{
			name: "skips assistant records to find user",
			content: strings.Join([]string{
				makeJSONLRecord("assistant", "", ""),
				makeJSONLRecord("user", "my-slug", "session-2"),
			}, "\n"),
			wantSlug: "my-slug",
		},
		{
			name:     "returns empty slug for empty file",
			content:  "",
			wantSlug: "",
		},
		{
			name:     "returns empty slug when no user record",
			content:  makeJSONLRecord("assistant", "", ""),
			wantSlug: "",
		},
		{
			name:     "returns empty slug for user record without slug",
			content:  makeJSONLRecord("user", "", "session-3"),
			wantSlug: "",
		},
		{
			name: "returns later non-empty slug",
			content: strings.Join([]string{
				makeJSONLRecord("user", "", "session-4"),
				makeJSONLRecord("user", "late-slug", "session-4"),
			}, "\n"),
			wantSlug: "late-slug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "test.jsonl")
			writeTestFile(t, path, tt.content)

			slug, err := extractSessionSlug(path)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantSlug, slug)
		})
	}
}

func TestExtractSessionSlugNonExistent(t *testing.T) {
	t.Parallel()

	_, err := extractSessionSlug("/nonexistent/file.jsonl")
	require.Error(t, err)
}

func TestAnalyzeProjectDir(t *testing.T) {
	t.Parallel()

	t.Run("new conversations have no archive", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		srcDir := filepath.Join(dir, "source")
		archDir := filepath.Join(dir, "archive")
		projDir := filepath.Join(srcDir, "proj1")

		writeTestFile(t, filepath.Join(projDir, "s1.jsonl"),
			makeJSONLRecord("user", "feat-a", "id1"))
		writeTestFile(t, filepath.Join(projDir, "s2.jsonl"),
			makeJSONLRecord("user", "feat-b", "id2"))

		cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
		seen := make(map[groupKey]*conversationState)
		var syncCandidates []string

		inspected, err := analyzeProjectDir(projDir, cfg, seen, &syncCandidates)
		require.NoError(t, err)
		assert.Equal(t, 2, inspected)
		assert.Len(t, seen, 2)
		assert.Len(t, syncCandidates, 2)

		newConvs, toUpdate, upToDate := classifyConversations(seen)
		assert.Equal(t, 2, newConvs)
		assert.Zero(t, toUpdate)
		assert.Zero(t, upToDate)
	})

	t.Run("up-to-date conversations have matching archive", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		srcDir := filepath.Join(dir, "source")
		archDir := filepath.Join(dir, "archive")
		projDir := filepath.Join(srcDir, "proj1")

		content := makeJSONLRecord("user", "feat-a", "id1")
		writeTestFile(t, filepath.Join(projDir, "s1.jsonl"), content)

		// Create matching archive file with same size and mod time
		archPath := filepath.Join(
			providerRawDir(archDir, conversationProviderClaude),
			"proj1",
			"s1.jsonl",
		)
		writeTestFile(t, archPath, content)
		srcInfo, _ := os.Stat(filepath.Join(projDir, "s1.jsonl"))
		_ = os.Chtimes(archPath, srcInfo.ModTime(), srcInfo.ModTime())

		cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
		seen := make(map[groupKey]*conversationState)
		var syncCandidates []string

		_, err := analyzeProjectDir(projDir, cfg, seen, &syncCandidates)
		require.NoError(t, err)
		assert.Empty(t, syncCandidates)

		_, _, upToDate := classifyConversations(seen)
		assert.Equal(t, 1, upToDate)
	})

	t.Run("stale archive classifies as to-update", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		srcDir := filepath.Join(dir, "source")
		archDir := filepath.Join(dir, "archive")
		projDir := filepath.Join(srcDir, "proj1")

		content := makeJSONLRecord("user", "feat-a", "id1")
		writeTestFile(t, filepath.Join(projDir, "s1.jsonl"), content)

		// Create archive file with older mod time
		archPath := filepath.Join(
			providerRawDir(archDir, conversationProviderClaude),
			"proj1",
			"s1.jsonl",
		)
		writeTestFile(t, archPath, content)
		past := time.Now().Add(-1 * time.Hour)
		_ = os.Chtimes(archPath, past, past)

		cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
		seen := make(map[groupKey]*conversationState)
		var syncCandidates []string

		_, err := analyzeProjectDir(projDir, cfg, seen, &syncCandidates)
		require.NoError(t, err)
		assert.Len(t, syncCandidates, 1)

		_, toUpdate, _ := classifyConversations(seen)
		assert.Equal(t, 1, toUpdate)
	})

	t.Run("same slug groups into one conversation", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		srcDir := filepath.Join(dir, "source")
		archDir := filepath.Join(dir, "archive")
		projDir := filepath.Join(srcDir, "proj1")

		writeTestFile(t, filepath.Join(projDir, "s1.jsonl"),
			makeJSONLRecord("user", "same-slug", "id1"))
		writeTestFile(t, filepath.Join(projDir, "s2.jsonl"),
			makeJSONLRecord("user", "same-slug", "id2"))

		cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
		seen := make(map[groupKey]*conversationState)
		var syncCandidates []string

		_, err := analyzeProjectDir(projDir, cfg, seen, &syncCandidates)
		require.NoError(t, err)
		assert.Len(t, seen, 1)
	})

	t.Run("later slug matches browser grouping", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		srcDir := filepath.Join(dir, "source")
		archDir := filepath.Join(dir, "archive")
		projDir := filepath.Join(srcDir, "proj1")

		firstFile := strings.Join([]string{
			makeJSONLRecord("user", "", "id1"),
			makeJSONLRecord("user", "shared-slug", "id1"),
		}, "\n")
		writeTestFile(t, filepath.Join(projDir, "s1.jsonl"), firstFile)
		writeTestFile(t, filepath.Join(projDir, "s2.jsonl"),
			makeJSONLRecord("user", "shared-slug", "id2"))

		cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
		seen := make(map[groupKey]*conversationState)
		var syncCandidates []string

		_, err := analyzeProjectDir(projDir, cfg, seen, &syncCandidates)
		require.NoError(t, err)
		assert.Len(t, seen, 1)
	})

	t.Run("subagent files get unique keys", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		srcDir := filepath.Join(dir, "source")
		archDir := filepath.Join(dir, "archive")
		projDir := filepath.Join(srcDir, "proj1")

		writeTestFile(t, filepath.Join(projDir, "s1.jsonl"),
			makeJSONLRecord("user", "feat-a", "id1"))
		writeTestFile(t, filepath.Join(projDir, "uuid1", "subagents", "agent-1.jsonl"),
			makeJSONLRecord("user", "feat-a", "sub-id1"))

		cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
		seen := make(map[groupKey]*conversationState)
		var syncCandidates []string

		_, err := analyzeProjectDir(projDir, cfg, seen, &syncCandidates)
		require.NoError(t, err)

		// Subagent should be separate even with same slug
		assert.Len(t, seen, 2)
	})

	t.Run("empty source directory returns zero", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		srcDir := filepath.Join(dir, "source")
		archDir := filepath.Join(dir, "archive")
		projDir := filepath.Join(srcDir, "proj1")
		require.NoError(t, os.MkdirAll(projDir, 0o755))

		cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
		seen := make(map[groupKey]*conversationState)
		var syncCandidates []string

		inspected, err := analyzeProjectDir(projDir, cfg, seen, &syncCandidates)
		require.NoError(t, err)
		assert.Zero(t, inspected)
		assert.Empty(t, seen)
	})
}

func TestListProjectDirs(t *testing.T) {
	t.Parallel()

	t.Run("returns directories only", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "proj1"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "proj2"), 0o755))
		writeTestFile(t, filepath.Join(dir, "file.txt"), "not a dir")

		dirs, err := listProjectDirs(dir)
		require.NoError(t, err)
		assert.Len(t, dirs, 2)
	})

	t.Run("missing source returns nil", func(t *testing.T) {
		t.Parallel()
		dirs, err := listProjectDirs("/nonexistent/path")
		require.NoError(t, err)
		assert.Nil(t, dirs)
	})
}

func TestClassifyConversations(t *testing.T) {
	t.Parallel()

	seen := map[groupKey]*conversationState{
		{dirName: "p1", slug: "a"}: {allNew: true},
		{dirName: "p1", slug: "b"}: {allNew: true},
		{dirName: "p1", slug: "c"}: {hasStale: true, hasUpToDate: true},
		{dirName: "p2", slug: "d"}: {hasUpToDate: true},
	}

	newConvs, toUpdate, upToDate := classifyConversations(seen)
	assert.Equal(t, 2, newConvs)
	assert.Equal(t, 1, toUpdate)
	assert.Equal(t, 1, upToDate)
}

func TestImportAnalysisNeedsSync(t *testing.T) {
	t.Parallel()

	t.Run("needs sync with files", func(t *testing.T) {
		t.Parallel()
		a := importAnalysis{filesToSync: []string{"/a.jsonl"}}
		assert.True(t, a.needsSync())
	})

	t.Run("no sync needed when empty", func(t *testing.T) {
		t.Parallel()
		a := importAnalysis{}
		assert.False(t, a.needsSync())
	})
}

// makeJSONLRecord creates a minimal JSONL record for testing.
func makeJSONLRecord(recordType, slug, sessionID string) string {
	rec := map[string]any{
		"type":      recordType,
		"sessionId": sessionID,
		"slug":      slug,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	}
	if recordType == string(roleUser) {
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
		panic(fmt.Sprintf("json.Marshal: %v", err))
	}
	return string(raw)
}
