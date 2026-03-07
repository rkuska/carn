package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileNeedsSync(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, srcPath, dstPath string)
		want  bool
	}{
		{
			name: "dst missing",
			setup: func(t *testing.T, srcPath, _ string) {
				t.Helper()
				writeTestFile(t, srcPath, "content")
			},
			want: true,
		},
		{
			name: "size differs",
			setup: func(t *testing.T, srcPath, dstPath string) {
				t.Helper()
				writeTestFile(t, srcPath, "longer content")
				writeTestFile(t, dstPath, "short")
			},
			want: true,
		},
		{
			name: "src newer",
			setup: func(t *testing.T, srcPath, dstPath string) {
				t.Helper()
				writeTestFile(t, srcPath, "content")
				writeTestFile(t, dstPath, "content")
				past := time.Now().Add(-1 * time.Hour)
				if err := os.Chtimes(dstPath, past, past); err != nil {
					t.Fatalf("os.Chtimes: %v", err)
				}
			},
			want: true,
		},
		{
			name: "identical",
			setup: func(t *testing.T, srcPath, dstPath string) {
				t.Helper()
				writeTestFile(t, srcPath, "content")
				writeTestFile(t, dstPath, "content")
				// Set same mod time
				now := time.Now()
				if err := os.Chtimes(srcPath, now, now); err != nil {
					t.Fatalf("os.Chtimes src: %v", err)
				}
				if err := os.Chtimes(dstPath, now, now); err != nil {
					t.Fatalf("os.Chtimes dst: %v", err)
				}
			},
			want: false,
		},
		{
			name: "dst newer",
			setup: func(t *testing.T, srcPath, dstPath string) {
				t.Helper()
				writeTestFile(t, srcPath, "content")
				writeTestFile(t, dstPath, "content")
				past := time.Now().Add(-1 * time.Hour)
				if err := os.Chtimes(srcPath, past, past); err != nil {
					t.Fatalf("os.Chtimes: %v", err)
				}
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "src", "file.jsonl")
			dstPath := filepath.Join(dir, "dst", "file.jsonl")

			tt.setup(t, srcPath, dstPath)

			srcInfo, err := os.Stat(srcPath)
			if err != nil {
				t.Fatalf("os.Stat src: %v", err)
			}

			got := fileNeedsSync(srcInfo, dstPath)
			if got != tt.want {
				t.Errorf("fileNeedsSync() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	t.Parallel()

	t.Run("content correctness and mod time", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "src", "file.jsonl")
		dstPath := filepath.Join(dir, "dst", "nested", "file.jsonl")

		content := `{"type":"user","message":"hello"}`
		writeTestFile(t, srcPath, content)

		// Set a specific mod time
		modTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		if err := os.Chtimes(srcPath, modTime, modTime); err != nil {
			t.Fatalf("os.Chtimes: %v", err)
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			t.Fatalf("copyFile: %v", err)
		}

		// Verify content
		got, err := os.ReadFile(dstPath)
		if err != nil {
			t.Fatalf("os.ReadFile: %v", err)
		}
		if string(got) != content {
			t.Errorf("content = %q, want %q", string(got), content)
		}

		// Verify mod time preserved
		dstInfo, err := os.Stat(dstPath)
		if err != nil {
			t.Fatalf("os.Stat: %v", err)
		}
		if !dstInfo.ModTime().Equal(modTime) {
			t.Errorf("mod time = %v, want %v", dstInfo.ModTime(), modTime)
		}
	})

	t.Run("non-existent source error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		err := copyFile(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dst"))
		if err == nil {
			t.Error("expected error for non-existent source")
		}
	})
}

func TestDefaultArchiveConfigDefaults(t *testing.T) {
	t.Setenv("CLDSRCH_SOURCE_DIR", "")
	t.Setenv("CLDSRCH_ARCHIVE_DIR", "")

	cfg, err := defaultArchiveConfig()
	if err != nil {
		t.Fatalf("defaultArchiveConfig: %v", err)
	}

	home, _ := os.UserHomeDir()
	wantSource := filepath.Join(home, ".claude", "projects")
	wantArchive := filepath.Join(home, ".local", "share", "cldrsrch")

	if cfg.sourceDir != wantSource {
		t.Errorf("sourceDir = %q, want %q", cfg.sourceDir, wantSource)
	}
	if cfg.archiveDir != wantArchive {
		t.Errorf("archiveDir = %q, want %q", cfg.archiveDir, wantArchive)
	}
}

func TestDefaultArchiveConfigEnvOverrides(t *testing.T) {
	t.Setenv("CLDSRCH_SOURCE_DIR", "/custom/source")
	t.Setenv("CLDSRCH_ARCHIVE_DIR", "/custom/archive")

	cfg, err := defaultArchiveConfig()
	if err != nil {
		t.Fatalf("defaultArchiveConfig: %v", err)
	}

	if cfg.sourceDir != "/custom/source" {
		t.Errorf("sourceDir = %q, want %q", cfg.sourceDir, "/custom/source")
	}
	if cfg.archiveDir != "/custom/archive" {
		t.Errorf("archiveDir = %q, want %q", cfg.archiveDir, "/custom/archive")
	}
}

// Scenario tests — sequential, use t.TempDir()

func TestSyncArchiveFreshCopy(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	archDir := filepath.Join(dir, "archive")

	// Create source tree with .jsonl files including subagents
	writeTestFile(t, filepath.Join(srcDir, "proj1", "session1.jsonl"), `{"type":"user"}`)
	writeTestFile(t, filepath.Join(srcDir, "proj1", "session2.jsonl"), `{"type":"user"}`)
	writeTestFile(t, filepath.Join(srcDir, "proj2", "session3.jsonl"), `{"type":"assistant"}`)
	writeTestFile(t, filepath.Join(srcDir, "proj1", "uuid1", "subagents", "agent-1.jsonl"), `{"type":"user"}`)

	cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
	result, err := syncArchive(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("syncArchive: %v", err)
	}

	if result.copied != 4 {
		t.Errorf("copied = %d, want 4", result.copied)
	}
	if result.failed != 0 {
		t.Errorf("failed = %d, want 0", result.failed)
	}

	// Verify files exist in archive
	for _, rel := range []string{
		"proj1/session1.jsonl",
		"proj1/session2.jsonl",
		"proj2/session3.jsonl",
		"proj1/uuid1/subagents/agent-1.jsonl",
	} {
		path := filepath.Join(archDir, rel)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected archived file at %s: %v", path, err)
		}
	}
}

func TestSyncArchiveIncremental(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	archDir := filepath.Join(dir, "archive")

	writeTestFile(t, filepath.Join(srcDir, "proj", "a.jsonl"), "original")
	writeTestFile(t, filepath.Join(srcDir, "proj", "b.jsonl"), "original")

	cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}

	// First sync — copies both
	result1, err := syncArchive(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("first syncArchive: %v", err)
	}
	if result1.copied != 2 {
		t.Errorf("first sync copied = %d, want 2", result1.copied)
	}

	// Modify one file
	time.Sleep(10 * time.Millisecond) // ensure mod time differs
	writeTestFile(t, filepath.Join(srcDir, "proj", "a.jsonl"), "modified content")

	// Second sync — only re-copies the modified file
	result2, err := syncArchive(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("second syncArchive: %v", err)
	}
	if result2.copied != 1 {
		t.Errorf("second sync copied = %d, want 1", result2.copied)
	}

	// Verify modified content was copied
	got, err := os.ReadFile(filepath.Join(archDir, "proj", "a.jsonl"))
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	if string(got) != "modified content" {
		t.Errorf("content = %q, want %q", string(got), "modified content")
	}
}

func TestSyncArchiveSkipsNonJSONL(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	archDir := filepath.Join(dir, "archive")

	writeTestFile(t, filepath.Join(srcDir, "proj", "session.jsonl"), "jsonl content")
	writeTestFile(t, filepath.Join(srcDir, "proj", "data.json"), "json content")
	writeTestFile(t, filepath.Join(srcDir, "proj", "notes.txt"), "text content")

	cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
	result, err := syncArchive(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("syncArchive: %v", err)
	}

	if result.copied != 1 {
		t.Errorf("copied = %d, want 1", result.copied)
	}

	// .jsonl should exist
	if _, err := os.Stat(filepath.Join(archDir, "proj", "session.jsonl")); err != nil {
		t.Errorf("expected .jsonl archived: %v", err)
	}
	// .json should NOT exist
	if _, err := os.Stat(filepath.Join(archDir, "proj", "data.json")); !os.IsNotExist(err) {
		t.Error("expected .json NOT to be archived")
	}
	// .txt should NOT exist
	if _, err := os.Stat(filepath.Join(archDir, "proj", "notes.txt")); !os.IsNotExist(err) {
		t.Error("expected .txt NOT to be archived")
	}
}

func TestSyncArchiveMissingSource(t *testing.T) {
	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "nonexistent"),
		archiveDir: filepath.Join(dir, "archive"),
	}

	result, err := syncArchive(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("syncArchive should not error for missing source: %v", err)
	}
	if result.copied != 0 {
		t.Errorf("copied = %d, want 0", result.copied)
	}
}

func TestSyncArchivePartialFailure(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	archDir := filepath.Join(dir, "archive")

	writeTestFile(t, filepath.Join(srcDir, "proj", "good.jsonl"), "good content")
	writeTestFile(t, filepath.Join(srcDir, "proj", "bad.jsonl"), "bad content")

	// Create archive dir for "bad" with read-only permissions to cause copy failure
	badDir := filepath.Join(archDir, "proj")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}
	// Create a directory where the file should go to prevent file creation
	if err := os.MkdirAll(filepath.Join(badDir, "bad.jsonl"), 0o000); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}

	cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
	result, err := syncArchive(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("syncArchive: %v", err)
	}

	// One should succeed, one should fail
	if result.copied != 1 {
		t.Errorf("copied = %d, want 1", result.copied)
	}
	if result.failed != 1 {
		t.Errorf("failed = %d, want 1", result.failed)
	}
}

func TestSyncArchiveProgressCallback(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	archDir := filepath.Join(dir, "archive")

	writeTestFile(t, filepath.Join(srcDir, "proj", "a.jsonl"), "a")
	writeTestFile(t, filepath.Join(srcDir, "proj", "b.jsonl"), "b")

	cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}

	var progressCalls []syncProgress
	result, err := syncArchive(context.Background(), cfg, func(p syncProgress) {
		progressCalls = append(progressCalls, p)
	})
	if err != nil {
		t.Fatalf("syncArchive: %v", err)
	}

	if result.copied != 2 {
		t.Errorf("copied = %d, want 2", result.copied)
	}
	if len(progressCalls) != 2 {
		t.Errorf("progress callbacks = %d, want 2", len(progressCalls))
	}
	for _, p := range progressCalls {
		if p.total != 2 {
			t.Errorf("progress total = %d, want 2", p.total)
		}
	}
}

// writeTestFile creates a file with content, creating parent dirs as needed.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
}
