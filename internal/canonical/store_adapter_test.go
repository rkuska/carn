package canonical

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rkuska/carn/internal/source/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreDeepSearchAvailabilityFollowsSearchIndexPresence(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(claude.New())

	conversations := []conversation{{
		Ref:      conversationRef{Provider: conversationProvider("claude"), ID: "s1"},
		Sessions: []sessionMeta{{ID: "s1"}},
	}}

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "", conversations)
	require.NoError(t, err)
	assert.False(t, available)
	assert.Equal(t, conversations, results)

	require.NoError(t, writeSearchFile(filepath.Join(canonicalStoreDir(archiveDir), "search.bin"), searchCorpus{}))

	results, available, err = store.DeepSearch(context.Background(), archiveDir, "", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	assert.Equal(t, conversations, results)
}

func TestStoreDeepSearchCacheIsSharedAcrossStoreCopies(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(claude.New())
	storeCopy := store

	conversations := []conversation{{
		Ref:      conversationRef{Provider: conversationProvider("claude"), ID: "s1"},
		Sessions: []sessionMeta{{ID: "s1"}},
	}}

	searchPath := filepath.Join(canonicalStoreDir(archiveDir), "search.bin")
	require.NoError(t, writeSearchFile(searchPath, searchCorpus{units: []searchUnit{{
		conversationID: conversations[0].CacheKey(),
		text:           "needle",
	}}}))

	_, available, err := store.DeepSearch(context.Background(), archiveDir, "", conversations)
	require.NoError(t, err)
	assert.True(t, available)

	require.NoError(t, os.Remove(searchPath))

	results, available, err := storeCopy.DeepSearch(context.Background(), archiveDir, "needle", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.Len(t, results, 1)
	assert.Equal(t, conversations[0].CacheKey(), results[0].CacheKey())
}

func TestStoreDeepSearchUsesWarmedSearchCache(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(claude.New())

	conversations := []conversation{{
		Ref:      conversationRef{Provider: conversationProvider("claude"), ID: "s1"},
		Sessions: []sessionMeta{{ID: "s1"}},
	}}

	searchPath := filepath.Join(canonicalStoreDir(archiveDir), "search.bin")
	require.NoError(t, writeSearchFile(searchPath, searchCorpus{units: []searchUnit{{
		conversationID: conversations[0].CacheKey(),
		text:           "needle",
	}}}))

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "needle", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.Len(t, results, 1)
	assert.Equal(t, conversations[0].CacheKey(), results[0].CacheKey())

	require.NoError(t, os.Remove(searchPath))

	results, available, err = store.DeepSearch(context.Background(), archiveDir, "needle", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.Len(t, results, 1)
	assert.Equal(t, conversations[0].CacheKey(), results[0].CacheKey())
}
