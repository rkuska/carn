package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractSessionSlugHandlesLongUserRecord(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	line := makeTestUserRecord(t, "session-1", "long-slug", strings.Repeat("long message ", 600))
	require.NoError(t, os.WriteFile(path, []byte(line+"\n"), 0o644))

	slug, err := extractSessionSlug(path)
	require.NoError(t, err)
	assert.Equal(t, "long-slug", slug)
}

func TestExtractSessionSlugReturnsEmptyWhenMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	line := makeTestUserRecord(t, "session-1", "", "hello")
	require.NoError(t, os.WriteFile(path, []byte(line+"\n"), 0o644))

	slug, err := extractSessionSlug(path)
	require.NoError(t, err)
	assert.Empty(t, slug)
}

func TestClassifyProjectFileUsesPathForSubagents(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	rawDir := t.TempDir()
	path := filepath.Join(sourceDir, "project-a", "session-1", "subagents", "agent-1.jsonl")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(makeTestUserRecord(t, "session-1", "slug-ignored", "hello")+"\n"), 0o644))

	classified, ok := classifyProjectFile(sessionFile{
		path:       path,
		isSubagent: true,
	}, sourceDir, rawDir, "project-a")
	require.True(t, ok)
	assert.Equal(t, path, classified.gk.slug)
	assert.False(t, classified.dstExists)
	assert.True(t, classified.needsSync)
}
