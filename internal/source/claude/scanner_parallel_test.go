package claude

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanSessionFilesParallelReturnsScannedSessions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(makeTestUserRecord(t, "s1", "demo", "hello")), 0o644))

	sessions, err := scanSessionFilesParallel(context.Background(), []sessionFile{{
		path:         path,
		project:      project{DisplayName: "demo"},
		groupDirName: "project-a",
	}})
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "s1", sessions[0].meta.ID)
	assert.Equal(t, groupKey{dirName: "project-a", slug: "demo"}, sessions[0].groupKey)
}
