package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestAddPerformanceSessionUsesProviderTokenAccounting(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		provider         conv.Provider
		usage            conv.TokenUsage
		wantTotalTokens  int
		wantPromptTokens int
	}{
		{
			name:     "claude adds cache and reasoning tokens",
			provider: conv.ProviderClaude,
			usage: conv.TokenUsage{
				InputTokens:              100,
				CacheCreationInputTokens: 7,
				CacheReadInputTokens:     90,
				OutputTokens:             20,
				ReasoningOutputTokens:    10,
			},
			wantTotalTokens:  227,
			wantPromptTokens: 197,
		},
		{
			name:     "codex uses provider total and raw input tokens",
			provider: conv.ProviderCodex,
			usage: conv.TokenUsage{
				InputTokens:              100,
				CacheCreationInputTokens: 7,
				CacheReadInputTokens:     90,
				OutputTokens:             20,
				ReasoningOutputTokens:    10,
			},
			wantTotalTokens:  120,
			wantPromptTokens: 100,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var aggregate performanceAggregate
			addPerformanceSession(&aggregate, performanceSession{
				provider: testCase.provider,
				meta: &conv.SessionMeta{
					ID:               "session",
					Timestamp:        time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC),
					UserMessageCount: 2,
					TotalUsage:       testCase.usage,
				},
			})

			assert.Equal(t, testCase.wantTotalTokens, aggregate.totalTokens)
			assert.Equal(t, testCase.wantPromptTokens, aggregate.cachePromptTokens)
		})
	}
}

func TestComputePerformanceUsesCodexProviderTokenTotals(t *testing.T) {
	t.Parallel()

	timeRange := TimeRange{
		Start: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC),
	}
	conversations := makePerformanceConversations(
		conv.ProviderCodex,
		"current",
		time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
		1,
		func(meta *conv.SessionMeta) {
			meta.Model = "gpt-5.4"
			meta.UserMessageCount = 2
			meta.TotalUsage = conv.TokenUsage{
				InputTokens:           100,
				CacheReadInputTokens:  90,
				OutputTokens:          20,
				ReasoningOutputTokens: 10,
			}
		},
	)

	got := ComputePerformance(conversations, timeRange, nil)

	tokensPerTurn := findPerformanceMetric(t, got.Efficiency.Metrics, perfMetricTokensPerTurn)
	assert.InDelta(t, 60.0, tokensPerTurn.Current, 0.0001)

	reasoningShare := findPerformanceMetric(t, got.Efficiency.Metrics, perfMetricReasoningShare)
	assert.InDelta(t, 10.0/120.0, reasoningShare.Current, 0.0001)
}
