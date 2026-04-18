package claude

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanSessionFilesParallelReturnsScannedSessions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(makeTestUserRecord(t, "s1", "demo", "hello")), 0o644))

	sessions, _, _, err := scanSessionFilesParallel(context.Background(), []sessionFile{{
		path:         path,
		project:      project{DisplayName: "demo"},
		groupDirName: "project-a",
	}})
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "s1", sessions[0].meta.ID)
	assert.Equal(t, groupKey{dirName: "project-a", slug: "demo"}, sessions[0].groupKey)
}

func TestScanSessionFilesParallelLogsMalformedSkipAtWarn(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.InfoLevel)
	ctx := logger.WithContext(context.Background())

	missing := filepath.Join(t.TempDir(), "missing.jsonl")

	sessions, _, report, err := scanSessionFilesParallel(ctx, []sessionFile{{
		path:         missing,
		project:      project{DisplayName: "demo"},
		groupDirName: "project-a",
	}})
	require.NoError(t, err)
	require.Empty(t, sessions)
	assert.Equal(t, 1, report.Count())

	out := buf.String()
	assert.Contains(t, out, `"level":"warn"`)
	assert.Contains(t, out, `"provider":"claude"`)
	assert.Contains(t, out, `"path":"`+missing+`"`)
	assert.Contains(t, out, "skipping malformed raw data")
}

func TestScanSessionFilesParallelLogsNoMetadataAtInfoWithoutRecording(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.InfoLevel)
	ctx := logger.WithContext(context.Background())

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	require.NoError(t, os.WriteFile(path, nil, 0o644))

	sessions, _, report, err := scanSessionFilesParallel(ctx, []sessionFile{{
		path:         path,
		project:      project{DisplayName: "demo"},
		groupDirName: "project-a",
	}})
	require.NoError(t, err)
	require.Empty(t, sessions)
	assert.True(t, report.Empty(), "no-metadata sessions must not be reported as malformed")

	out := buf.String()
	assert.Contains(t, out, `"level":"info"`)
	assert.Contains(t, out, `"provider":"claude"`)
	assert.Contains(t, out, `"path":"`+path+`"`)
	assert.Contains(t, out, "skipping session without metadata")
	assert.NotContains(t, out, "skipping malformed raw data")
}
