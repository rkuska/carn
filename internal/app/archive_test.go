package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				require.NoError(t, os.Chtimes(dstPath, past, past))
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
				require.NoError(t, os.Chtimes(srcPath, now, now))
				require.NoError(t, os.Chtimes(dstPath, now, now))
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
				require.NoError(t, os.Chtimes(srcPath, past, past))
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
			require.NoError(t, err)

			got := fileNeedsSync(srcInfo, dstPath)
			assert.Equal(t, tt.want, got)
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
		require.NoError(t, os.Chtimes(srcPath, modTime, modTime))

		require.NoError(t, copyFile(srcPath, dstPath))

		// Verify content
		got, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, content, string(got))

		// Verify mod time preserved
		dstInfo, err := os.Stat(dstPath)
		require.NoError(t, err)
		assert.True(t, dstInfo.ModTime().Equal(modTime))
	})

	t.Run("non-existent source error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		err := copyFile(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dst"))
		require.Error(t, err)
	})
}

func TestDefaultArchiveConfigDefaults(t *testing.T) {
	t.Setenv("CARN_SOURCE_DIR", "")
	t.Setenv("CARN_ARCHIVE_DIR", "")

	cfg, err := defaultArchiveConfig()
	require.NoError(t, err)

	home, _ := os.UserHomeDir()
	wantSource := filepath.Join(home, ".claude", "projects")
	wantArchive := filepath.Join(home, ".local", "share", "carn")

	assert.Equal(t, wantSource, cfg.sourceDir)
	assert.Equal(t, wantArchive, cfg.archiveDir)
}

func TestDefaultArchiveConfigEnvOverrides(t *testing.T) {
	t.Setenv("CARN_SOURCE_DIR", "/custom/source")
	t.Setenv("CARN_ARCHIVE_DIR", "/custom/archive")

	cfg, err := defaultArchiveConfig()
	require.NoError(t, err)
	assert.Equal(t, "/custom/source", cfg.sourceDir)
	assert.Equal(t, "/custom/archive", cfg.archiveDir)
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
	require.NoError(t, err)
	assert.Equal(t, 4, result.copied)
	assert.Zero(t, result.failed)

	// Verify files exist in archive
	for _, rel := range []string{
		"proj1/session1.jsonl",
		"proj1/session2.jsonl",
		"proj2/session3.jsonl",
		"proj1/uuid1/subagents/agent-1.jsonl",
	} {
		path := filepath.Join(archDir, rel)
		_, err := os.Stat(path)
		assert.NoError(t, err)
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
	require.NoError(t, err)
	assert.Equal(t, 2, result1.copied)

	// Modify one file
	time.Sleep(10 * time.Millisecond) // ensure mod time differs
	writeTestFile(t, filepath.Join(srcDir, "proj", "a.jsonl"), "modified content")

	// Second sync — only re-copies the modified file
	result2, err := syncArchive(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result2.copied)

	// Verify modified content was copied
	got, err := os.ReadFile(filepath.Join(archDir, "proj", "a.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, "modified content", string(got))
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
	require.NoError(t, err)
	assert.Equal(t, 1, result.copied)

	// .jsonl should exist
	_, err = os.Stat(filepath.Join(archDir, "proj", "session.jsonl"))
	assert.NoError(t, err)
	// .json should NOT exist
	_, err = os.Stat(filepath.Join(archDir, "proj", "data.json"))
	assert.True(t, os.IsNotExist(err))
	// .txt should NOT exist
	_, err = os.Stat(filepath.Join(archDir, "proj", "notes.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestSyncArchiveMissingSource(t *testing.T) {
	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "nonexistent"),
		archiveDir: filepath.Join(dir, "archive"),
	}

	result, err := syncArchive(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.Zero(t, result.copied)
}

func TestSyncArchivePartialFailure(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	archDir := filepath.Join(dir, "archive")

	writeTestFile(t, filepath.Join(srcDir, "proj", "good.jsonl"), "good content")
	writeTestFile(t, filepath.Join(srcDir, "proj", "bad.jsonl"), "bad content")

	// Create archive dir for "bad" with read-only permissions to cause copy failure
	badDir := filepath.Join(archDir, "proj")
	require.NoError(t, os.MkdirAll(badDir, 0o755))
	// Create a directory where the file should go to prevent file creation
	require.NoError(t, os.MkdirAll(filepath.Join(badDir, "bad.jsonl"), 0o000))

	cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
	result, err := syncArchive(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.copied)
	assert.Equal(t, 1, result.failed)
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
	require.NoError(t, err)
	assert.Equal(t, 2, result.copied)
	assert.Len(t, progressCalls, 2)
	for _, p := range progressCalls {
		assert.Equal(t, 2, p.total)
	}
}

// writeTestFile creates a file with content, creating parent dirs as needed.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
