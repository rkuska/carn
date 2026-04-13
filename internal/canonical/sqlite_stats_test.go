package canonical

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/internal/source/claude"
)

func TestStoreQueryStatsFiltersByCacheKeyAndAggregatesDailyTokens(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	store := New(nil, claude.New())

	first := testSQLiteConversation("s1")
	second := testSQLiteConversation("s2")
	day := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)

	require.NoError(t, writeCanonicalStoreAtomically(
		context.Background(),
		archiveDir,
		[]conversation{first, second},
		map[string]sessionFull{
			first.CacheKey():  {Meta: first.Sessions[0]},
			second.CacheKey(): {Meta: second.Sessions[0]},
		},
		searchCorpus{},
		map[string][]conv.SessionStatsData{
			first.CacheKey(): {{
				PerformanceSequence: conv.PerformanceSequenceSession{
					Timestamp:     first.Sessions[0].Timestamp,
					Mutated:       true,
					MutationCount: 2,
				},
				TurnMetrics: conv.SessionTurnMetrics{
					Timestamp: first.Sessions[0].Timestamp,
					Turns: []conv.TurnTokens{{
						InputTokens: 120,
						TurnTokens:  180,
					}},
				},
			}},
			second.CacheKey(): {{
				PerformanceSequence: conv.PerformanceSequenceSession{
					Timestamp:     second.Sessions[0].Timestamp.Add(time.Minute),
					MutationCount: 1,
				},
				TurnMetrics: conv.SessionTurnMetrics{
					Timestamp: second.Sessions[0].Timestamp.Add(time.Minute),
					Turns: []conv.TurnTokens{{
						InputTokens: 90,
						TurnTokens:  140,
					}},
				},
			}},
		},
		map[string][]conv.ActivityBucketRow{
			first.CacheKey(): {{
				BucketStart:           day,
				Provider:              "claude",
				Model:                 "claude-sonnet-4",
				Project:               "claude",
				SessionCount:          1,
				MessageCount:          4,
				UserMessageCount:      2,
				AssistantMessageCount: 2,
				InputTokens:           10,
				OutputTokens:          3,
			}},
			second.CacheKey(): {{
				BucketStart:           day,
				Provider:              "claude",
				Model:                 "claude-sonnet-4",
				Project:               "claude",
				SessionCount:          1,
				MessageCount:          6,
				UserMessageCount:      3,
				AssistantMessageCount: 3,
				InputTokens:           20,
				OutputTokens:          5,
			}},
		},
	))

	sequence, err := store.QueryPerformanceSequence(context.Background(), archiveDir, []string{first.CacheKey()})
	require.NoError(t, err)
	require.Len(t, sequence, 1)
	assert.Equal(t, 2, sequence[0].MutationCount)
	assert.True(t, sequence[0].Mutated)

	turnMetrics, err := store.QueryTurnMetrics(context.Background(), archiveDir, []string{first.CacheKey()})
	require.NoError(t, err)
	require.Len(t, turnMetrics, 1)
	assert.Equal(t, []conv.TurnTokens{{InputTokens: 120, TurnTokens: 180}}, turnMetrics[0].Turns)

	daily, err := store.QueryActivityBuckets(
		context.Background(),
		archiveDir,
		[]string{first.CacheKey(), second.CacheKey()},
	)
	require.NoError(t, err)
	require.Len(t, daily, 1)
	assert.Equal(t, 2, daily[0].SessionCount)
	assert.Equal(t, 10, daily[0].MessageCount)
	assert.Equal(t, 30, daily[0].InputTokens)
	assert.Equal(t, 8, daily[0].OutputTokens)

	filteredDaily, err := store.QueryActivityBuckets(context.Background(), archiveDir, []string{first.CacheKey()})
	require.NoError(t, err)
	require.Len(t, filteredDaily, 1)
	assert.Equal(t, 1, filteredDaily[0].SessionCount)
	assert.Equal(t, 4, filteredDaily[0].MessageCount)
	assert.Equal(t, 10, filteredDaily[0].InputTokens)
}
