package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "test.jsonl")
			writeTestFile(t, path, tt.content)

			slug, err := extractSessionSlug(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("extractSessionSlug() error = %v, wantErr %v", err, tt.wantErr)
			}
			if slug != tt.wantSlug {
				t.Errorf("extractSessionSlug() = %q, want %q", slug, tt.wantSlug)
			}
		})
	}
}

func TestExtractSessionSlugNonExistent(t *testing.T) {
	t.Parallel()

	_, err := extractSessionSlug("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
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
		if err != nil {
			t.Fatalf("analyzeProjectDir: %v", err)
		}

		if inspected != 2 {
			t.Errorf("inspected = %d, want 2", inspected)
		}
		if len(seen) != 2 {
			t.Errorf("conversations = %d, want 2", len(seen))
		}
		if len(syncCandidates) != 2 {
			t.Errorf("syncCandidates = %d, want 2", len(syncCandidates))
		}

		newConvs, toUpdate, upToDate := classifyConversations(seen)
		if newConvs != 2 {
			t.Errorf("new = %d, want 2", newConvs)
		}
		if toUpdate != 0 {
			t.Errorf("toUpdate = %d, want 0", toUpdate)
		}
		if upToDate != 0 {
			t.Errorf("upToDate = %d, want 0", upToDate)
		}
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
		archPath := filepath.Join(archDir, "proj1", "s1.jsonl")
		writeTestFile(t, archPath, content)
		srcInfo, _ := os.Stat(filepath.Join(projDir, "s1.jsonl"))
		_ = os.Chtimes(archPath, srcInfo.ModTime(), srcInfo.ModTime())

		cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
		seen := make(map[groupKey]*conversationState)
		var syncCandidates []string

		_, err := analyzeProjectDir(projDir, cfg, seen, &syncCandidates)
		if err != nil {
			t.Fatalf("analyzeProjectDir: %v", err)
		}

		if len(syncCandidates) != 0 {
			t.Errorf("syncCandidates = %d, want 0", len(syncCandidates))
		}

		_, _, upToDate := classifyConversations(seen)
		if upToDate != 1 {
			t.Errorf("upToDate = %d, want 1", upToDate)
		}
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
		archPath := filepath.Join(archDir, "proj1", "s1.jsonl")
		writeTestFile(t, archPath, content)
		past := time.Now().Add(-1 * time.Hour)
		_ = os.Chtimes(archPath, past, past)

		cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
		seen := make(map[groupKey]*conversationState)
		var syncCandidates []string

		_, err := analyzeProjectDir(projDir, cfg, seen, &syncCandidates)
		if err != nil {
			t.Fatalf("analyzeProjectDir: %v", err)
		}

		if len(syncCandidates) != 1 {
			t.Errorf("syncCandidates = %d, want 1", len(syncCandidates))
		}

		_, toUpdate, _ := classifyConversations(seen)
		if toUpdate != 1 {
			t.Errorf("toUpdate = %d, want 1", toUpdate)
		}
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
		if err != nil {
			t.Fatalf("analyzeProjectDir: %v", err)
		}

		if len(seen) != 1 {
			t.Errorf("conversations = %d, want 1 (same slug should group)", len(seen))
		}
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
		if err != nil {
			t.Fatalf("analyzeProjectDir: %v", err)
		}

		// Subagent should be separate even with same slug
		if len(seen) != 2 {
			t.Errorf("conversations = %d, want 2 (subagent should be separate)", len(seen))
		}
	})

	t.Run("empty source directory returns zero", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		srcDir := filepath.Join(dir, "source")
		archDir := filepath.Join(dir, "archive")
		projDir := filepath.Join(srcDir, "proj1")
		if err := os.MkdirAll(projDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}

		cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
		seen := make(map[groupKey]*conversationState)
		var syncCandidates []string

		inspected, err := analyzeProjectDir(projDir, cfg, seen, &syncCandidates)
		if err != nil {
			t.Fatalf("analyzeProjectDir: %v", err)
		}

		if inspected != 0 {
			t.Errorf("inspected = %d, want 0", inspected)
		}
		if len(seen) != 0 {
			t.Errorf("conversations = %d, want 0", len(seen))
		}
	})
}

func TestListProjectDirs(t *testing.T) {
	t.Parallel()

	t.Run("returns directories only", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "proj1"), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "proj2"), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		writeTestFile(t, filepath.Join(dir, "file.txt"), "not a dir")

		dirs, err := listProjectDirs(dir)
		if err != nil {
			t.Fatalf("listProjectDirs: %v", err)
		}
		if len(dirs) != 2 {
			t.Errorf("dirs = %d, want 2", len(dirs))
		}
	})

	t.Run("missing source returns nil", func(t *testing.T) {
		t.Parallel()
		dirs, err := listProjectDirs("/nonexistent/path")
		if err != nil {
			t.Fatalf("listProjectDirs: %v", err)
		}
		if dirs != nil {
			t.Errorf("dirs = %v, want nil", dirs)
		}
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
	if newConvs != 2 {
		t.Errorf("new = %d, want 2", newConvs)
	}
	if toUpdate != 1 {
		t.Errorf("toUpdate = %d, want 1", toUpdate)
	}
	if upToDate != 1 {
		t.Errorf("upToDate = %d, want 1", upToDate)
	}
}

func TestImportAnalysisNeedsSync(t *testing.T) {
	t.Parallel()

	t.Run("needs sync with files", func(t *testing.T) {
		t.Parallel()
		a := importAnalysis{filesToSync: []string{"/a.jsonl"}}
		if !a.needsSync() {
			t.Error("expected needsSync() = true")
		}
	})

	t.Run("no sync needed when empty", func(t *testing.T) {
		t.Parallel()
		a := importAnalysis{}
		if a.needsSync() {
			t.Error("expected needsSync() = false")
		}
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
