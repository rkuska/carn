package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestAggregateActivityBucketsSingleSessionSingleDay(t *testing.T) {
	t.Parallel()

	inputSession := conv.Session{
		Meta: testMeta(
			"s1",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withUsage(999, 0, 0, 999),
			withMainMessages(4),
			withRoleCounts(1, 2),
		),
		Messages: []conv.Message{
			{
				Role:      conv.RoleAssistant,
				Timestamp: time.Date(2026, 1, 5, 9, 5, 0, 0, time.UTC),
				Usage: conv.TokenUsage{
					InputTokens:              120,
					CacheCreationInputTokens: 30,
					CacheReadInputTokens:     10,
					OutputTokens:             40,
					ReasoningOutputTokens:    5,
				},
			},
		},
	}

	got := AggregateActivityBuckets([]conv.Session{inputSession})
	require.Len(t, got, 2)
	assert.Equal(t, []conv.ActivityBucketRow{
		{
			BucketStart:           time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			Provider:              string(conv.ProviderClaude),
			Model:                 "claude-sonnet-4",
			Project:               "proj",
			SessionCount:          1,
			MessageCount:          4,
			UserMessageCount:      1,
			AssistantMessageCount: 2,
		},
		{
			BucketStart:           time.Date(2026, 1, 5, 9, 5, 0, 0, time.UTC),
			Provider:              string(conv.ProviderClaude),
			Model:                 "claude-sonnet-4",
			Project:               "proj",
			InputTokens:           120,
			CacheCreationTokens:   30,
			CacheReadTokens:       10,
			OutputTokens:          40,
			ReasoningOutputTokens: 5,
		},
	}, got)
}

func TestAggregateActivityBucketsSplitsTokensAcrossMidnight(t *testing.T) {
	t.Parallel()

	inputSession := conv.Session{
		Meta: testMeta(
			"s1",
			time.Date(2026, 1, 5, 23, 55, 0, 0, time.UTC),
			withMainMessages(6),
			withRoleCounts(2, 3),
		),
		Messages: []conv.Message{
			{
				Role:      conv.RoleAssistant,
				Timestamp: time.Date(2026, 1, 5, 23, 58, 0, 0, time.UTC),
				Usage:     conv.TokenUsage{InputTokens: 80, OutputTokens: 20},
			},
			{
				Role:      conv.RoleAssistant,
				Timestamp: time.Date(2026, 1, 6, 0, 2, 0, 0, time.UTC),
				Usage:     conv.TokenUsage{InputTokens: 50, OutputTokens: 10},
			},
		},
	}

	got := AggregateActivityBuckets([]conv.Session{inputSession})
	require.Len(t, got, 3)
	assert.Equal(t, []conv.ActivityBucketRow{
		{
			BucketStart:           time.Date(2026, 1, 5, 23, 55, 0, 0, time.UTC),
			Provider:              string(conv.ProviderClaude),
			Model:                 "claude-sonnet-4",
			Project:               "proj",
			SessionCount:          1,
			MessageCount:          6,
			UserMessageCount:      2,
			AssistantMessageCount: 3,
		},
		{
			BucketStart:  time.Date(2026, 1, 5, 23, 58, 0, 0, time.UTC),
			Provider:     string(conv.ProviderClaude),
			Model:        "claude-sonnet-4",
			Project:      "proj",
			InputTokens:  80,
			OutputTokens: 20,
		},
		{
			BucketStart:  time.Date(2026, 1, 6, 0, 2, 0, 0, time.UTC),
			Provider:     string(conv.ProviderClaude),
			Model:        "claude-sonnet-4",
			Project:      "proj",
			InputTokens:  50,
			OutputTokens: 10,
		},
	}, got)
}

func TestAggregateActivityBucketsFallsBackToSessionStartForZeroMessageTimestamp(t *testing.T) {
	t.Parallel()

	inputSession := conv.Session{
		Meta: testMeta("s1", time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC), withMainMessages(3)),
		Messages: []conv.Message{{
			Role:  conv.RoleAssistant,
			Usage: conv.TokenUsage{InputTokens: 40, OutputTokens: 12},
		}},
	}

	got := AggregateActivityBuckets([]conv.Session{inputSession})
	require.Len(t, got, 1)
	assert.Equal(t, time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC), got[0].BucketStart)
	assert.Equal(t, 1, got[0].SessionCount)
	assert.Equal(t, 3, got[0].MessageCount)
	assert.Equal(t, 40, got[0].InputTokens)
	assert.Equal(t, 12, got[0].OutputTokens)
}

func TestAggregateActivityBucketsUsesSessionMessageCountsNotTranscriptLength(t *testing.T) {
	t.Parallel()

	inputSession := conv.Session{
		Meta: testMeta(
			"subagent",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			func(meta *conv.SessionMeta) {
				meta.IsSubagent = true
				meta.MessageCount = 12
				meta.MainMessageCount = 0
				meta.UserMessageCount = 4
				meta.AssistantMessageCount = 8
			},
		),
		Messages: []conv.Message{{
			Role:      conv.RoleAssistant,
			Timestamp: time.Date(2026, 1, 5, 9, 1, 0, 0, time.UTC),
			Usage:     conv.TokenUsage{InputTokens: 20, OutputTokens: 5},
		}},
	}

	got := AggregateActivityBuckets([]conv.Session{inputSession})
	require.Len(t, got, 2)
	assert.Equal(t, 12, got[0].MessageCount)
	assert.Equal(t, 4, got[0].UserMessageCount)
	assert.Equal(t, 8, got[0].AssistantMessageCount)
	assert.Equal(t, 20, got[1].InputTokens)
	assert.Equal(t, 5, got[1].OutputTokens)
}

func TestAggregateActivityBucketsKeepsRowsSeparateByModelAndProject(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	sessions := []conv.Session{
		{
			Meta: testMeta("a", day, withModel("claude-opus"), withProject("proj-a")),
			Messages: []conv.Message{{
				Role:      conv.RoleAssistant,
				Timestamp: day.Add(time.Minute),
				Usage:     conv.TokenUsage{InputTokens: 10, OutputTokens: 2},
			}},
		},
		{
			Meta: testMeta("b", day, withModel("claude-sonnet-4"), withProject("proj-b")),
			Messages: []conv.Message{{
				Role:      conv.RoleAssistant,
				Timestamp: day.Add(2 * time.Minute),
				Usage:     conv.TokenUsage{InputTokens: 20, OutputTokens: 4},
			}},
		},
	}

	got := AggregateActivityBuckets(sessions)
	require.Len(t, got, 4)
	assert.Equal(t, "claude-opus", got[0].Model)
	assert.Equal(t, "proj-a", got[0].Project)
	assert.Equal(t, "claude-sonnet-4", got[1].Model)
	assert.Equal(t, "proj-b", got[1].Project)
	assert.Equal(t, "claude-opus", got[2].Model)
	assert.Equal(t, "proj-a", got[2].Project)
	assert.Equal(t, "claude-sonnet-4", got[3].Model)
	assert.Equal(t, "proj-b", got[3].Project)
}
