package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

const testCodexModel = "gpt-5.4"

func TestComputePerformanceUsesMetricDenominatorForSampleThresholds(t *testing.T) {
	t.Parallel()

	timeRange := TimeRange{
		Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	}
	conversations := append(
		makePerformanceConversations(
			conv.ProviderClaude,
			"baseline",
			time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 2
				if meta.Timestamp.Day() != 4 {
					return
				}
				meta.ActionCounts = map[string]int{
					string(conv.NormalizedActionMutate): 1,
					string(conv.NormalizedActionTest):   1,
				}
			},
		),
		makePerformanceConversations(
			conv.ProviderClaude,
			"current",
			time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 2
				if meta.Timestamp.Day() != 1 {
					return
				}
				meta.ActionCounts = map[string]int{
					string(conv.NormalizedActionMutate): 1,
					string(conv.NormalizedActionTest):   1,
				}
			},
		)...,
	)

	got := ComputePerformance(conversations, timeRange, nil)

	verification := findPerformanceMetric(t, got.Outcome.Metrics, perfMetricVerificationPass)
	assert.Equal(t, 1, verification.SampleCount)
	assert.False(t, verification.HasScore)
	assert.Equal(t, PerformanceMetricStatusLowSample, verification.Status)
	assert.False(t, got.Outcome.HasScore)
}

func TestComputePerformanceUsesSequenceMetricDenominatorForSampleThresholds(t *testing.T) {
	t.Parallel()

	timeRange := TimeRange{
		Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	}
	conversations := append(
		makePerformanceConversations(
			conv.ProviderClaude,
			"baseline",
			time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 2
			},
		),
		makePerformanceConversations(
			conv.ProviderClaude,
			"current",
			time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 2
			},
		)...,
	)
	sequence := append(
		make([]PerformanceSequenceSession, 0, 24),
		PerformanceSequenceSession{
			Timestamp:          time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC),
			Mutated:            true,
			FirstPassResolved:  true,
			VerificationPassed: true,
			ActionCount:        2,
			AssistantTurns:     1,
		},
	)
	for i := 1; i < 12; i++ {
		sequence = append(sequence, PerformanceSequenceSession{
			Timestamp:      time.Date(2026, 1, 4+i, 12, 0, 0, 0, time.UTC),
			ActionCount:    1,
			AssistantTurns: 1,
		})
	}
	sequence = append(sequence, PerformanceSequenceSession{
		Timestamp:          time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
		Mutated:            true,
		FirstPassResolved:  true,
		VerificationPassed: true,
		ActionCount:        2,
		AssistantTurns:     1,
	})
	for i := 1; i < 12; i++ {
		sequence = append(sequence, PerformanceSequenceSession{
			Timestamp:      time.Date(2026, 2, 1+i, 12, 0, 0, 0, time.UTC),
			ActionCount:    1,
			AssistantTurns: 1,
		})
	}

	got := ComputePerformance(conversations, timeRange, sequence)

	firstPass := findPerformanceMetric(t, got.Outcome.Metrics, perfMetricFirstPassResolution)
	assert.Equal(t, 1, firstPass.SampleCount)
	assert.False(t, firstPass.HasScore)
	assert.Equal(t, PerformanceMetricStatusLowSample, firstPass.Status)
}

func TestBuildPerformanceLaneUsesMetricWeights(t *testing.T) {
	t.Parallel()

	lane := buildPerformanceLane(
		"Efficiency",
		"Tracks token and action cost per unit of user direction.",
		[]PerformanceMetric{
			{ID: perfMetricTokensPerTurn, HasScore: true, Score: 80, Trend: TrendDirectionUp},
			{ID: perfMetricVisibleThinking, HasScore: true, Score: 20, Trend: TrendDirectionDown},
			{ID: perfMetricTimeToMutation, HasScore: true, Score: 0, Trend: TrendDirectionDown},
		},
	)

	require.True(t, lane.HasScore)
	assert.Equal(t, 68, lane.Score)
	assert.Equal(t, TrendDirectionUp, lane.Trend)
}

func TestComputePerformanceGatesProviderSpecificRobustnessMetrics(t *testing.T) {
	t.Parallel()

	timeRange := TimeRange{
		Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	}

	claude := append(
		makePerformanceConversations(
			conv.ProviderClaude,
			"baseline-claude",
			time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 2
				meta.Performance.RetryAttemptCount = 1
				meta.Performance.APIErrorCounts = map[string]int{"api_error": 1}
				meta.Performance.AbortCount = 1
				meta.Performance.TaskStartedCount = 2
			},
		),
		makePerformanceConversations(
			conv.ProviderClaude,
			"current-claude",
			time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 2
				meta.Performance.RetryAttemptCount = 1
				meta.Performance.APIErrorCounts = map[string]int{"api_error": 1}
				meta.Performance.AbortCount = 1
				meta.Performance.TaskStartedCount = 2
			},
		)...,
	)
	got := ComputePerformance(claude, timeRange, nil)
	assert.Nil(t, tryFindPerformanceMetric(got.Robustness.Metrics, perfMetricAbortRate))
	assert.NotNil(t, tryFindPerformanceMetric(got.Robustness.Metrics, perfMetricRetryBurden))

	codex := append(
		makePerformanceConversations(
			conv.ProviderCodex,
			"baseline-codex",
			time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.Model = testCodexModel
				meta.UserMessageCount = 2
				meta.Performance.AbortCount = 1
				meta.Performance.TaskStartedCount = 2
				meta.Performance.RetryAttemptCount = 1
				meta.Performance.APIErrorCounts = map[string]int{"api_error": 1}
			},
		),
		makePerformanceConversations(
			conv.ProviderCodex,
			"current-codex",
			time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.Model = testCodexModel
				meta.UserMessageCount = 2
				meta.Performance.AbortCount = 1
				meta.Performance.TaskStartedCount = 2
				meta.Performance.RetryAttemptCount = 1
				meta.Performance.APIErrorCounts = map[string]int{"api_error": 1}
			},
		)...,
	)
	got = ComputePerformance(codex, timeRange, nil)
	assert.NotNil(t, tryFindPerformanceMetric(got.Robustness.Metrics, perfMetricAbortRate))
	assert.Nil(t, tryFindPerformanceMetric(got.Robustness.Metrics, perfMetricRetryBurden))
}

func TestPerformanceTimeWindowUsesImmediatelyPreviousEqualDurationBaseline(t *testing.T) {
	t.Parallel()

	current := TimeRange{
		Start: time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 14, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC),
	}

	window := performanceTimeWindow[performanceSession](current, nil)

	assert.Equal(t, current, window.current)
	assert.Equal(t, TimeRange{
		Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 7, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC),
	}, window.baseline)
}

func TestPerformanceMetricDeltaUsesCurrentMinusBaseline(t *testing.T) {
	t.Parallel()

	metric := performanceMetricFromRatio(
		perfMetricErrorRate,
		"tool error rate",
		1,
		10,
		2,
		10,
		false,
		performanceMinSessionSamples,
		0.05,
		"Errored action results / action calls.",
		performanceMetricContext[performanceAggregate]{
			currentSampleCount:  10,
			baselineSampleCount: 10,
		},
		func(performanceAggregate) (float64, float64) {
			return 0, 0
		},
	)

	metric = enrichPerformanceMetric(metric)

	assert.InDelta(t, 0.1, metric.Current, 0.0001)
	assert.InDelta(t, 0.2, metric.Baseline, 0.0001)
	assert.Equal(t, TrendDirectionUp, metric.Trend)
	assert.Equal(t, "-10.0 pts", metric.DeltaText)
}
