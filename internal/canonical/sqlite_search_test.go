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

func TestBuildSearchPreviewQueryUsesPerConversationRowLimit(t *testing.T) {
	t.Parallel()

	query, args := buildSearchPreviewQuery([]int64{11, 22})

	assert.Contains(t, query, "ROW_NUMBER() OVER (PARTITION BY conversation_id ORDER BY ordinal ASC)")
	assert.Contains(t, query, "match_row <= ?")
	require.Len(t, args, 3)
	assert.EqualValues(t, 11, args[0])
	assert.EqualValues(t, 22, args[1])
	assert.Equal(t, searchPreviewFetchRowsPerConversation, args[2])
}

func TestReadSearchPreviewsReturnsThreeUniquePreviewsPerConversation(t *testing.T) {
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
	corpus := searchCorpus{units: []searchUnit{
		{conversationID: conversationValue.CacheKey(), ordinal: 0, text: repeated},
		{conversationID: conversationValue.CacheKey(), ordinal: 1, text: repeated},
		{conversationID: conversationValue.CacheKey(), ordinal: 2, text: repeated},
		{conversationID: conversationValue.CacheKey(), ordinal: 3, text: repeated},
		{conversationID: conversationValue.CacheKey(), ordinal: 4, text: repeated},
		{
			conversationID: conversationValue.CacheKey(),
			ordinal:        5,
			text:           strings.Repeat("before ", 8) + "important alpha unique line" + strings.Repeat(" after", 8),
		},
		{
			conversationID: conversationValue.CacheKey(),
			ordinal:        6,
			text:           strings.Repeat("before ", 8) + "important beta unique line" + strings.Repeat(" after", 8),
		},
		{
			conversationID: conversationValue.CacheKey(),
			ordinal:        7,
			text:           strings.Repeat("before ", 8) + "important gamma unique line" + strings.Repeat(" after", 8),
		},
	}}

	writeSQLiteTestStore(t, archiveDir, []conversation{conversationValue}, map[string]sessionFull{
		conversationValue.CacheKey(): {
			Meta: conversationValue.Sessions[0],
			Messages: []message{
				{Role: role("assistant"), Text: "important alpha unique line"},
			},
		},
	}, corpus)

	db, err := openSQLiteDB(canonicalStorePath(archiveDir), true)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	matches, err := readRankedConversationMatches(
		context.Background(),
		db,
		buildFTSQuery("important"),
	)
	require.NoError(t, err)
	require.Len(t, matches, 1)

	conversationIDs := []int64{matches[0].id}
	previews, err := readSearchPreviews(
		context.Background(),
		db,
		conversationIDs,
		matches,
		searchTerms("important"),
	)
	require.NoError(t, err)
	require.Len(t, previews[conversationValue.CacheKey()], 3)
	assert.Contains(t, previews[conversationValue.CacheKey()][0], "important repeated line")
	assert.Contains(t, previews[conversationValue.CacheKey()][1], "important alpha unique line")
	assert.Contains(t, previews[conversationValue.CacheKey()][2], "important beta unique line")
}
