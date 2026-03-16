package canonical

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFTSQuery(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "single token",
			query: "cache",
			want:  `"cache"*`,
		},
		{
			name:  "multiple tokens join with and",
			query: "cache reload",
			want:  `"cache"* AND "reload"*`,
		},
		{
			name:  "quotes are escaped",
			query: `say "hi"`,
			want:  `"say"* AND """hi"""*`,
		},
		{
			name:  "blank query",
			query: "   ",
			want:  "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, buildFTSQuery(testCase.query))
		})
	}
}

func TestBuildDeepSearchQueryUsesSingleMatchPass(t *testing.T) {
	t.Parallel()

	query, args := buildDeepSearchQuery(`"GENERATE_UUID"*`)

	assert.Equal(t, 1, strings.Count(query, "search_fts MATCH ?"))
	assert.Contains(t, query, "snippet(search_fts, 0, '', '', '...',")
	assert.Contains(t, query, "ranked_conversations AS")
	assert.Contains(t, query, "LEFT JOIN ranked_previews")
	assert.Contains(t, query, "GROUP BY conversation_id, preview")
	assert.Contains(t, query, "ROW_NUMBER() OVER (PARTITION BY conversation_id ORDER BY first_ordinal ASC)")
	assert.Contains(t, query, "rp.preview_row <= ?")
	require.Len(t, args, 2)
	assert.Equal(t, `"GENERATE_UUID"*`, args[0])
	assert.Equal(t, searchPreviewMaxPerConversation, args[1])
}

func TestReadSearchPreviewsDeduplicatesBeforeApplyingConversationLimit(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	conversationValue := conversation{
		Ref:       conversationRef{Provider: conversationProvider("claude"), ID: "s1"},
		Name:      "search",
		Project:   project{DisplayName: "claude"},
		PlanCount: 0,
		Sessions: []sessionMeta{{
			ID:            "s1",
			Project:       project{DisplayName: "claude"},
			Timestamp:     time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC),
			LastTimestamp: time.Date(2026, 3, 13, 10, 5, 0, 0, time.UTC),
			FilePath:      "/raw/s1.jsonl",
		}},
	}

	repeated := strings.Repeat("important repeated line ", 8)
	units := make([]searchUnit, 0, 15)
	for ordinal := range 12 {
		units = append(units, searchUnit{
			conversationID: conversationValue.CacheKey(),
			ordinal:        ordinal,
			text:           repeated,
		})
	}
	units = append(units,
		searchUnit{
			conversationID: conversationValue.CacheKey(),
			ordinal:        12,
			text:           strings.Repeat("before ", 8) + "important alpha unique line" + strings.Repeat(" after", 8),
		},
		searchUnit{
			conversationID: conversationValue.CacheKey(),
			ordinal:        13,
			text:           strings.Repeat("before ", 8) + "important beta unique line" + strings.Repeat(" after", 8),
		},
		searchUnit{
			conversationID: conversationValue.CacheKey(),
			ordinal:        14,
			text:           strings.Repeat("before ", 8) + "important gamma unique line" + strings.Repeat(" after", 8),
		},
	)
	corpus := searchCorpus{units: units}

	writeSQLiteTestStore(t, archiveDir, []conversation{conversationValue}, map[string]sessionFull{
		conversationValue.CacheKey(): {
			Meta: conversationValue.Sessions[0],
			Messages: []message{
				{Role: role("assistant"), Text: "important alpha unique line"},
			},
		},
	}, corpus)

	db, err := openSQLiteDB(context.Background(), canonicalStorePath(archiveDir), true)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	matches, err := readRankedConversationMatches(
		context.Background(),
		db,
		buildFTSQuery("important"),
	)
	require.NoError(t, err)
	require.Len(t, matches, 1)

	require.Len(t, matches[0].previews, 3)
	assert.Contains(t, matches[0].previews[0], "important repeated line")
	assert.Contains(t, matches[0].previews[1], "important alpha unique line")
	assert.Contains(t, matches[0].previews[2], "important beta unique line")
}

func TestReadSearchPreviewsGroupsDedupedPreviewsPerConversation(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	first := testSQLiteConversation("s1")
	second := testSQLiteConversation("s2")
	second.Sessions[0].Timestamp = second.Sessions[0].Timestamp.Add(time.Minute)
	second.Sessions[0].LastTimestamp = second.Sessions[0].LastTimestamp.Add(time.Minute)

	corpus := searchCorpus{units: []searchUnit{
		{conversationID: first.CacheKey(), ordinal: 0, text: strings.Repeat("common first ", 8) + "important repeated alpha"},
		{conversationID: first.CacheKey(), ordinal: 1, text: strings.Repeat("common first ", 8) + "important repeated alpha"},
		{conversationID: first.CacheKey(), ordinal: 2, text: strings.Repeat("before ", 8) + "important unique first"},
		{
			conversationID: second.CacheKey(),
			ordinal:        0,
			text:           strings.Repeat("common second ", 8) + "important repeated beta",
		},
		{
			conversationID: second.CacheKey(),
			ordinal:        1,
			text:           strings.Repeat("common second ", 8) + "important repeated beta",
		},
		{conversationID: second.CacheKey(), ordinal: 2, text: strings.Repeat("before ", 8) + "important unique second"},
	}}

	writeSQLiteTestStore(t, archiveDir, []conversation{first, second}, map[string]sessionFull{
		first.CacheKey():  {Meta: first.Sessions[0]},
		second.CacheKey(): {Meta: second.Sessions[0]},
	}, corpus)

	db, err := openSQLiteDB(context.Background(), canonicalStorePath(archiveDir), true)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	matches, err := readRankedConversationMatches(context.Background(), db, buildFTSQuery("important"))
	require.NoError(t, err)
	require.Len(t, matches, 2)

	previewsByCacheKey := make(map[string][]string, len(matches))
	for _, match := range matches {
		previewsByCacheKey[match.cacheKey] = match.previews
	}

	require.Len(t, previewsByCacheKey[first.CacheKey()], 2)
	assert.Contains(t, previewsByCacheKey[first.CacheKey()][0], "important repeated alpha")
	assert.Contains(t, previewsByCacheKey[first.CacheKey()][1], "important unique first")

	require.Len(t, previewsByCacheKey[second.CacheKey()], 2)
	assert.Contains(t, previewsByCacheKey[second.CacheKey()][0], "important repeated beta")
	assert.Contains(t, previewsByCacheKey[second.CacheKey()][1], "important unique second")
}

func TestReadSearchPreviewsUsesStableSnippetEllipses(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	conversationValue := testSQLiteConversation("s1")
	longChunk := strings.Join([]string{
		"alpha bravo charlie delta echo foxtrot golf hotel india juliet kilo lima mike november",
		"important needle",
		"oscar papa quebec romeo sierra tango uniform victor whiskey xray yankee zulu",
	}, " ")

	writeSQLiteTestStore(t, archiveDir, []conversation{conversationValue}, map[string]sessionFull{
		conversationValue.CacheKey(): {Meta: conversationValue.Sessions[0]},
	}, searchCorpus{units: []searchUnit{{
		conversationID: conversationValue.CacheKey(),
		ordinal:        0,
		text:           longChunk,
	}}})

	db, err := openSQLiteDB(context.Background(), canonicalStorePath(archiveDir), true)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	matches, err := readRankedConversationMatches(context.Background(), db, buildFTSQuery("needle"))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	require.Len(t, matches[0].previews, 1)
	assert.Contains(t, matches[0].previews[0], "important needle")
	assert.Contains(t, matches[0].previews[0], "...")
}
