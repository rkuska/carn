package canonical

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/rkuska/carn/internal/source/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreDeepSearchAvailabilityFollowsSQLitePresence(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(claude.New())

	conversations := []conversation{testSQLiteConversation("s1")}

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "", conversations)
	require.NoError(t, err)
	assert.False(t, available)
	assert.Equal(t, conversations, results)

	writeSQLiteTestStore(t, archiveDir, conversations, map[string]sessionFull{
		conversations[0].CacheKey(): {
			Meta: conversations[0].Sessions[0],
		},
	}, searchCorpus{})

	results, available, err = store.DeepSearch(context.Background(), archiveDir, "", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	assert.Equal(t, conversations, results)
}

func TestStoreNeedsRebuildCachesSQLiteHandle(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(claude.New())

	conversations := []conversation{testSQLiteConversation("s1")}
	writeSQLiteTestStore(t, archiveDir, conversations, map[string]sessionFull{
		conversations[0].CacheKey(): {
			Meta: conversations[0].Sessions[0],
			Messages: []message{
				{Role: role("assistant"), Text: "needle"},
			},
		},
	}, searchCorpus{units: []searchUnit{{
		conversationID: conversations[0].CacheKey(),
		ordinal:        0,
		text:           "needle",
	}}})

	needsRebuild, err := store.NeedsRebuild(context.Background(), archiveDir)
	require.NoError(t, err)
	assert.False(t, needsRebuild)

	cached, ok := store.cachedDB(canonicalStorePath(archiveDir))
	require.True(t, ok)

	db, err := store.loadDB(context.Background(), archiveDir)
	require.NoError(t, err)

	assert.Same(t, cached, db)
}

func TestStoreNeedsRebuildWhenSearchCorpusVersionIsStale(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(claude.New())

	conversations := []conversation{testSQLiteConversation("s1")}
	writeSQLiteTestStore(t, archiveDir, conversations, map[string]sessionFull{
		conversations[0].CacheKey(): {
			Meta: conversations[0].Sessions[0],
		},
	}, searchCorpus{})
	setSQLiteMetaValue(t, archiveDir, metaSearchKey, strconv.Itoa(storeSearchCorpusVersion-1))

	needsRebuild, err := store.NeedsRebuild(context.Background(), archiveDir)
	require.NoError(t, err)
	assert.True(t, needsRebuild)
}

func TestStoreDeepSearchAvailabilityIsFalseWhenSearchCorpusVersionIsStale(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(claude.New())

	conversations := []conversation{testSQLiteConversation("s1")}
	writeSQLiteTestStore(t, archiveDir, conversations, map[string]sessionFull{
		conversations[0].CacheKey(): {
			Meta: conversations[0].Sessions[0],
		},
	}, searchCorpus{})
	setSQLiteMetaValue(t, archiveDir, metaSearchKey, strconv.Itoa(storeSearchCorpusVersion-1))

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "", conversations)
	require.NoError(t, err)
	assert.False(t, available)
	assert.Equal(t, conversations, results)
}

func TestStoreDeepSearchQueryIsUnavailableWhenSearchCorpusVersionIsStale(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(claude.New())

	conversations := []conversation{testSQLiteConversation("s1")}
	writeSQLiteTestStore(t, archiveDir, conversations, map[string]sessionFull{
		conversations[0].CacheKey(): {
			Meta: conversations[0].Sessions[0],
			Messages: []message{
				{Role: role("assistant"), Text: "use generate uuid for ids"},
			},
		},
	}, searchCorpus{units: []searchUnit{{
		conversationID: conversations[0].CacheKey(),
		ordinal:        0,
		text:           "use generate uuid for ids",
	}}})
	setSQLiteMetaValue(t, archiveDir, metaSearchKey, strconv.Itoa(storeSearchCorpusVersion-1))

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "GENERATE_UUID", conversations)
	require.NoError(t, err)
	assert.False(t, available)
	assert.Nil(t, results)
}

func TestStoreDeepSearchUsesSQLiteIndex(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(claude.New())

	conversations := []conversation{testSQLiteConversation("s1")}
	writeSQLiteTestStore(t, archiveDir, conversations, map[string]sessionFull{
		conversations[0].CacheKey(): {
			Meta: conversations[0].Sessions[0],
			Messages: []message{
				{Role: role("assistant"), Text: "needle"},
			},
		},
	}, searchCorpus{units: []searchUnit{{
		conversationID: conversations[0].CacheKey(),
		ordinal:        0,
		text:           "needle",
	}}})

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "needle", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.Len(t, results, 1)
	assert.Equal(t, conversations[0].CacheKey(), results[0].CacheKey())
}

func TestStoreDeepSearchMatchesQueriesAcrossTokenizerSeparators(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		query       string
		text        string
		wantPreview string
	}{
		{
			name:        "underscore",
			query:       "GENERATE_UUID",
			text:        "use generate uuid for ids",
			wantPreview: "generate uuid",
		},
		{
			name:        "slash",
			query:       "foo/bar",
			text:        "use foo bar for route parsing",
			wantPreview: "foo bar",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			archiveDir := t.TempDir()
			store := New(claude.New())

			conversations := []conversation{testSQLiteConversation("s1")}
			writeSQLiteTestStore(t, archiveDir, conversations, map[string]sessionFull{
				conversations[0].CacheKey(): {
					Meta: conversations[0].Sessions[0],
					Messages: []message{
						{Role: role("assistant"), Text: testCase.text},
					},
				},
			}, searchCorpus{units: []searchUnit{{
				conversationID: conversations[0].CacheKey(),
				ordinal:        0,
				text:           testCase.text,
			}}})

			results, available, err := store.DeepSearch(
				context.Background(),
				archiveDir,
				testCase.query,
				conversations,
			)
			require.NoError(t, err)
			assert.True(t, available)
			require.Len(t, results, 1)
			assert.Equal(t, conversations[0].CacheKey(), results[0].CacheKey())
			assert.Contains(t, results[0].SearchPreview, testCase.wantPreview)
		})
	}
}

func TestStoreListReturnsIndependentCachedCopies(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(claude.New())

	conversations := []conversation{{
		Ref:       conversationRef{Provider: conversationProvider("claude"), ID: "s1"},
		Name:      "cached",
		Project:   project{DisplayName: "claude"},
		PlanCount: 0,
		Sessions: []sessionMeta{{
			ID:            "s1",
			Timestamp:     time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC),
			LastTimestamp: time.Date(2026, 3, 13, 10, 5, 0, 0, time.UTC),
			FilePath:      "/raw/s1.jsonl",
			Project:       project{DisplayName: "claude"},
			ToolCounts:    map[string]int{"Read": 1},
		}},
	}}
	writeSQLiteTestStore(t, archiveDir, conversations, map[string]sessionFull{
		conversations[0].CacheKey(): {Meta: conversations[0].Sessions[0]},
	}, searchCorpus{})

	first, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, first, 1)

	first[0].SetSearchPreview("mutated preview")
	first[0].Sessions[0].ToolCounts["Read"] = 99

	second, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, second, 1)
	assert.Empty(t, second[0].SearchPreview)
	assert.Equal(t, 1, second[0].Sessions[0].ToolCounts["Read"])
}

func writeSQLiteTestStore(
	tb testing.TB,
	archiveDir string,
	conversations []conversation,
	transcripts map[string]sessionFull,
	corpus searchCorpus,
) {
	tb.Helper()
	ctx := context.Background()
	require.NoError(tb, writeCanonicalStoreAtomically(ctx, archiveDir, conversations, transcripts, corpus))
}

func setSQLiteMetaValue(tb testing.TB, archiveDir, key, value string) {
	tb.Helper()

	db, err := openSQLiteDB(context.Background(), canonicalStorePath(archiveDir), true)
	require.NoError(tb, err)
	defer func() { _ = db.Close() }()

	_, err = db.ExecContext(
		context.Background(),
		`UPDATE meta SET value = ? WHERE key = ?`,
		value,
		key,
	)
	require.NoError(tb, err)
}

func testSQLiteConversation(id string) conversation {
	return conversation{
		Ref: conversationRef{Provider: conversationProvider("claude"), ID: id},
		Sessions: []sessionMeta{{
			ID:            id,
			Timestamp:     time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC),
			LastTimestamp: time.Date(2026, 3, 13, 10, 5, 0, 0, time.UTC),
			FilePath:      "/raw/" + id + ".jsonl",
			Project:       project{DisplayName: "claude"},
		}},
		Project: project{DisplayName: "claude"},
	}
}
