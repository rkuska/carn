package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestComputeSnapshotFiltersByTimeRangeAndLeavesClaudeTurnMetricsNil(t *testing.T) {
	t.Parallel()

	conversations := []conversation{
		testConversation(
			conv.ProviderClaude,
			"old",
			testMeta("old", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), withMainMessages(3), withUsage(100, 0, 0, 20)),
		),
		testConversation(
			conv.ProviderClaude,
			"in-range",
			testMeta("in-range", time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC), withMainMessages(5), withUsage(200, 0, 0, 50)),
		),
	}

	got := ComputeSnapshot(conversations, TimeRange{
		Start: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 15, 23, 59, 59, 0, time.UTC),
	}, nil)

	if got.Overview.SessionCount != 1 || got.Overview.MessageCount != 5 {
		t.Fatalf("Overview = %#v", got.Overview)
	}
	if got.Sessions.ClaudeTurnMetrics != nil {
		t.Fatalf("ClaudeTurnMetrics = %#v, want nil", got.Sessions.ClaudeTurnMetrics)
	}
}

func TestComputeSnapshotAppliesLoadedPerformanceSequence(t *testing.T) {
	t.Parallel()

	currentTime := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
	conversations := []conversation{
		testConversation(
			conv.ProviderClaude,
			"current",
			testMeta(
				"current",
				currentTime,
				withModel("claude-sonnet-4"),
				func(meta *conv.SessionMeta) {
					meta.UserMessageCount = 1
					meta.ActionCounts = map[string]int{
						string(conv.NormalizedActionMutate): 1,
						string(conv.NormalizedActionTest):   1,
					}
				},
			),
		),
	}
	sequence := []PerformanceSequenceSession{
		{
			Timestamp:             currentTime,
			Mutated:               true,
			FirstPassResolved:     true,
			AssistantTurns:        1,
			MutationCount:         1,
			TargetedMutationCount: 1,
			ActionCount:           1,
		},
	}

	got := ComputeSnapshot(conversations, TimeRange{
		Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	}, sequence)

	if !got.Performance.Scope.SequenceLoaded {
		t.Fatalf("SequenceLoaded = false, want true")
	}
	if got.Performance.Scope.SequenceSampleCount != 1 {
		t.Fatalf("SequenceSampleCount = %d, want 1", got.Performance.Scope.SequenceSampleCount)
	}
}

func TestComputeSnapshotWithPrecomputedUsesDailyTokensAndTurnMetrics(t *testing.T) {
	t.Parallel()

	currentTime := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
	conversations := []conversation{
		testConversation(
			conv.ProviderClaude,
			"current",
			testMeta(
				"current",
				currentTime,
				withMainMessages(5),
				withUsage(999, 0, 0, 999),
				func(meta *conv.SessionMeta) {
					meta.UserMessageCount = 2
					meta.AssistantMessageCount = 3
				},
			),
		),
	}

	got := ComputeSnapshotWithPrecomputed(
		conversations,
		TimeRange{
			Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
		},
		[]conv.PerformanceSequenceSession{{
			Timestamp:         currentTime,
			Mutated:           true,
			FirstPassResolved: true,
			MutationCount:     1,
			ActionCount:       1,
		}},
		[]conv.SessionTurnMetrics{
			{
				Timestamp: currentTime,
				Turns: []conv.TurnTokens{{
					PromptTokens: 100,
					TurnTokens:   150,
				}},
			},
			{
				Timestamp: currentTime.Add(2 * time.Hour),
				Turns: []conv.TurnTokens{{
					PromptTokens: 120,
					TurnTokens:   180,
				}},
			},
			{
				Timestamp: currentTime.Add(4 * time.Hour),
				Turns: []conv.TurnTokens{{
					PromptTokens: 140,
					TurnTokens:   210,
				}},
			},
		},
		[]conv.ActivityBucketRow{
			{
				BucketStart:           time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
				SessionCount:          1,
				MessageCount:          5,
				UserMessageCount:      2,
				AssistantMessageCount: 3,
				InputTokens:           200,
				OutputTokens:          50,
			},
			{
				BucketStart:           time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
				SessionCount:          1,
				MessageCount:          5,
				UserMessageCount:      2,
				AssistantMessageCount: 3,
				InputTokens:           200,
				OutputTokens:          50,
			},
		},
	)

	assert.Equal(t, 1, got.Overview.SessionCount)
	assert.Equal(t, TrendDirectionFlat, got.Overview.TokenTrend.Direction)
	assert.Len(t, got.Sessions.ClaudeTurnMetrics, 1)
	assert.Equal(t, 1, got.Activity.ActiveDays)
	assert.True(t, got.Performance.Scope.SequenceLoaded)
}

func TestComputeSnapshotWithPrecomputedPopulatesCacheTurnStats(t *testing.T) {
	t.Parallel()

	currentTime := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	conversations := []conversation{
		testConversation(
			conv.ProviderClaude,
			"current",
			testMeta(
				"current",
				currentTime,
				withMainMessages(5),
				withUsage(100, 0, 0, 50),
			),
		),
	}

	turnMetrics := []conv.SessionTurnMetrics{
		{
			Provider:  conv.ProviderClaude,
			Version:   "1.0.0",
			Timestamp: currentTime,
			Turns: []conv.TurnTokens{
				{CacheReadTokens: 0},
				{CacheReadTokens: 20_000},
				{CacheReadTokens: 15_000, MemoryWriteCount: 1},
				{CacheReadTokens: 200},
			},
		},
		{
			Provider:  conv.ProviderClaude,
			Version:   "1.0.0",
			Timestamp: currentTime.Add(time.Hour),
			Turns: []conv.TurnTokens{
				{CacheReadTokens: 0},
				{CacheReadTokens: 25_000},
			},
		},
		{
			Provider:  conv.ProviderClaude,
			Version:   "1.0.0",
			Timestamp: currentTime.Add(2 * time.Hour),
			Turns: []conv.TurnTokens{
				{CacheReadTokens: 1_000},
			},
		},
	}

	got := ComputeSnapshotWithPrecomputed(
		conversations,
		TimeRange{
			Start: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC),
		},
		nil,
		turnMetrics,
		nil,
	)

	require.Len(t, got.Cache.FirstTurnByVersion, 1)
	assert.Equal(t, "1.0.0", got.Cache.FirstTurnByVersion[0].Version)
	assert.Equal(t, 3, got.Cache.FirstTurnByVersion[0].SessionCount)
}
