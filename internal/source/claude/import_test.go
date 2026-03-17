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

func TestExtractSessionSlugSkipsLeadingBlankAndAssistantRecords(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	content := strings.Join([]string{
		"",
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "assistant",
			"sessionId": "session-1",
			"timestamp": "2024-01-01T00:00:01Z",
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude",
				"content": []map[string]any{
					{"type": "text", "text": "hello"},
				},
			},
		}),
		makeTestUserRecord(t, "session-1", "late-slug", "hello"),
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	slug, err := extractSessionSlug(path)
	require.NoError(t, err)
	assert.Equal(t, "late-slug", slug)
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

func TestProjectSyncCandidatesSkipsUpToDateFile(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	rawDir := t.TempDir()
	projDir := filepath.Join(sourceDir, "project-a")
	require.NoError(t, os.MkdirAll(projDir, 0o755))

	relPath := filepath.Join("project-a", "session-1.jsonl")
	sourcePath := filepath.Join(sourceDir, relPath)
	destPath := filepath.Join(rawDir, relPath)
	content := makeTestUserRecord(t, "session-1", "slug-1", "hello") + "\n"

	require.NoError(t, os.WriteFile(sourcePath, []byte(content), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Dir(destPath), 0o755))
	require.NoError(t, os.WriteFile(destPath, []byte(content), 0o644))

	candidates, err := projectSyncCandidates(sourceDir, rawDir, projDir)
	require.NoError(t, err)
	assert.Empty(t, candidates)
}
