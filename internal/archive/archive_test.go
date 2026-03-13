package archive

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
	"github.com/rkuska/carn/internal/source/claude"
	"github.com/rkuska/carn/internal/source/codex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubBackend struct {
	provider       conv.Provider
	envVars        []string
	defaultDir     string
	syncCandidates []src.SyncCandidate
}

func (s stubBackend) Provider() conv.Provider {
	return s.provider
}

func (s stubBackend) Scan(context.Context, string) ([]conv.Conversation, error) {
	return nil, nil
}

func (s stubBackend) Load(context.Context, conv.Conversation) (conv.Session, error) {
	return conv.Session{}, nil
}

func (s stubBackend) Analyze(context.Context, string, string, func(src.Progress)) (src.Analysis, error) {
	return src.Analysis{}, nil
}

func (s stubBackend) ResumeCommand(conv.ResumeTarget) (*exec.Cmd, error) {
	return nil, nil
}

func (s stubBackend) SourceEnvVars() []string {
	return append([]string(nil), s.envVars...)
}

func (s stubBackend) DefaultSourceDir(string) string {
	return s.defaultDir
}

func (s stubBackend) SyncCandidates(context.Context, string, string) ([]src.SyncCandidate, error) {
	return append([]src.SyncCandidate(nil), s.syncCandidates...), nil
}

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
				past := time.Now().Add(-time.Hour)
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
				now := time.Now()
				require.NoError(t, os.Chtimes(srcPath, now, now))
				require.NoError(t, os.Chtimes(dstPath, now, now))
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
			assert.Equal(t, tt.want, fileNeedsSync(srcInfo, dstPath))
		})
	}
}

func TestCopyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src", "file.jsonl")
	dstPath := filepath.Join(dir, "dst", "nested", "file.jsonl")

	writeTestFile(t, srcPath, `{"type":"user","message":"hello"}`)
	modTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(srcPath, modTime, modTime))

	require.NoError(t, copyFile(srcPath, dstPath))

	got, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, `{"type":"user","message":"hello"}`, string(got))

	dstInfo, err := os.Stat(dstPath)
	require.NoError(t, err)
	assert.True(t, dstInfo.ModTime().Equal(modTime))
}

func TestDefaultConfig(t *testing.T) {
	t.Run("uses defaults", func(t *testing.T) {
		t.Setenv("CARN_SOURCE_DIR", "")
		t.Setenv("CARN_CLAUDE_SOURCE_DIR", "")
		t.Setenv("CARN_CODEX_SOURCE_DIR", "")
		t.Setenv("CARN_ARCHIVE_DIR", "")

		cfg, err := DefaultConfig(claude.New(), codex.New())
		require.NoError(t, err)

		home, _ := os.UserHomeDir()
		assert.Equal(t, filepath.Join(home, ".claude", "projects"), cfg.SourceDirs[conv.ProviderClaude])
		assert.Equal(t, filepath.Join(home, ".codex", "sessions"), cfg.SourceDirs[conv.ProviderCodex])
		assert.Equal(t, filepath.Join(home, ".local", "share", "carn"), cfg.ArchiveDir)
	})

	t.Run("uses env overrides", func(t *testing.T) {
		t.Setenv("CARN_SOURCE_DIR", "/custom/source")
		t.Setenv("CARN_CODEX_SOURCE_DIR", "/custom/codex")
		t.Setenv("CARN_ARCHIVE_DIR", "/custom/archive")

		cfg, err := DefaultConfig(claude.New(), codex.New())
		require.NoError(t, err)
		assert.Equal(t, "/custom/source", cfg.SourceDirs[conv.ProviderClaude])
		assert.Equal(t, "/custom/codex", cfg.SourceDirs[conv.ProviderCodex])
		assert.Equal(t, "/custom/archive", cfg.ArchiveDir)
	})
}

func TestCollectSyncCandidatesUsesBackendPlan(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	destDir := filepath.Join(dir, "archive")
	require.NoError(t, os.MkdirAll(sourceDir, 0o755))

	candidates, err := collectSyncCandidates(
		context.Background(),
		stubBackend{
			provider: conv.ProviderClaude,
			syncCandidates: []src.SyncCandidate{{
				SourcePath: filepath.Join(sourceDir, "proj", "a.jsonl"),
				DestPath:   filepath.Join(destDir, "proj", "a.jsonl"),
			}},
		},
		sourceDir,
		destDir,
	)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, conv.ProviderClaude, candidates[0].provider)
	assert.Equal(t, filepath.Join(sourceDir, "proj", "a.jsonl"), candidates[0].sourcePath)
	assert.Equal(t, syncStatusNew, candidates[0].status)
}

func TestSyncCandidatesReportsProgress(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "source", "session.jsonl")
	destPath := filepath.Join(dir, "archive", "session.jsonl")
	writeTestFile(t, sourcePath, `{"type":"user"}`)

	var progress []SyncProgress
	result, err := syncCandidates(
		context.Background(),
		[]syncCandidate{{sourcePath: sourcePath, destPath: destPath, status: syncStatusNew}},
		func(p SyncProgress) {
			progress = append(progress, p)
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Copied)
	assert.Zero(t, result.Failed)
	require.Len(t, progress, 1)
	assert.Equal(t, 1, progress[0].Current)
	assert.Equal(t, 1, progress[0].Copied)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
