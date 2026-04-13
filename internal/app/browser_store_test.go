package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rkuska/carn/internal/canonical"
	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
	"github.com/rkuska/carn/internal/source/claude"
)

func TestCanonicalBrowserStoreListClonesTopLevelConversations(t *testing.T) {
	t.Parallel()

	archiveDir, store := makeTestBrowserArchive(t)
	browserStore := newBrowserStore(store)

	first, err := browserStore.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, first, 1)

	first[0].SetSearchPreview("mutated preview")

	shared, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, shared, 1)
	assert.Empty(t, shared[0].SearchPreview)

	second, err := browserStore.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, second, 1)
	assert.Empty(t, second[0].SearchPreview)
}

func makeTestBrowserArchive(t *testing.T) (string, *canonical.Store) {
	t.Helper()

	archiveDir := t.TempDir()
	rawDir := src.ProviderRawDir(archiveDir, conv.ProviderClaude)
	projectDir := filepath.Join(rawDir, "project-a")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	sessionPath := filepath.Join(projectDir, "session-1.jsonl")
	session := "" +
		`{"type":"user","sessionId":"session-1","slug":"demo","timestamp":"2026-03-22T12:00:00Z",` +
		`"cwd":"/tmp/demo","message":{"role":"user","content":"inspect the parser"}}` +
		"\n" +
		`{"type":"assistant","timestamp":"2026-03-22T12:00:01Z","message":{"role":"assistant",` +
		`"model":"claude-sonnet-4","content":[{"type":"text","text":"done"}]}}`
	require.NoError(t, os.WriteFile(sessionPath, []byte(session), 0o644))

	store := canonical.New(nil, claude.New())
	_, err := store.RebuildAll(context.Background(), archiveDir, nil)
	require.NoError(t, err)
	return archiveDir, store
}
