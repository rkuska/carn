package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeSessionStatisticModesUseNearestRankPercentiles(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"s1",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 5, 9, 10, 0, 0, time.UTC)),
			withMainMessages(2),
		),
		testMeta(
			"s2",
			time.Date(2026, 1, 6, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 6, 9, 20, 0, 0, time.UTC)),
			withMainMessages(4),
		),
		testMeta(
			"s3",
			time.Date(2026, 1, 7, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 7, 9, 30, 0, 0, time.UTC)),
			withMainMessages(6),
		),
		testMeta(
			"s4",
			time.Date(2026, 1, 8, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 8, 9, 40, 0, 0, time.UTC)),
			withMainMessages(8),
		),
	}

	assert.Equal(t, 100*time.Minute, ComputeSessionDurationStatistic(sessions, StatisticModeTotal))
	assert.Equal(t, 25*time.Minute, ComputeSessionDurationStatistic(sessions, StatisticModeAverage))
	assert.Equal(t, 20*time.Minute, ComputeSessionDurationStatistic(sessions, StatisticModeP50))
	assert.Equal(t, 40*time.Minute, ComputeSessionDurationStatistic(sessions, StatisticModeP95))
	assert.Equal(t, 40*time.Minute, ComputeSessionDurationStatistic(sessions, StatisticModeP99))
	assert.Equal(t, 40*time.Minute, ComputeSessionDurationStatistic(sessions, StatisticModeMax))

	assert.InDelta(t, 20.0, ComputeSessionMessageStatistic(sessions, StatisticModeTotal), 0.0001)
	assert.InDelta(t, 5.0, ComputeSessionMessageStatistic(sessions, StatisticModeAverage), 0.0001)
	assert.InDelta(t, 4.0, ComputeSessionMessageStatistic(sessions, StatisticModeP50), 0.0001)
	assert.InDelta(t, 8.0, ComputeSessionMessageStatistic(sessions, StatisticModeP95), 0.0001)
	assert.InDelta(t, 8.0, ComputeSessionMessageStatistic(sessions, StatisticModeP99), 0.0001)
	assert.InDelta(t, 8.0, ComputeSessionMessageStatistic(sessions, StatisticModeMax), 0.0001)
}

func TestComputeTurnTokenMetricsForRangeWithModeUsesSelectedStatistic(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	sessions := []SessionTurnMetrics{
		{
			Timestamp: now,
			Turns: []TurnTokens{
				{PromptTokens: 10, TurnTokens: 15},
				{PromptTokens: 50, TurnTokens: 55},
			},
		},
		{
			Timestamp: now.Add(-time.Hour),
			Turns: []TurnTokens{
				{PromptTokens: 20, TurnTokens: 25},
				{PromptTokens: 60, TurnTokens: 65},
			},
		},
		{
			Timestamp: now.Add(-2 * time.Hour),
			Turns: []TurnTokens{
				{PromptTokens: 30, TurnTokens: 35},
				{PromptTokens: 70, TurnTokens: 75},
			},
		},
		{
			Timestamp: now.Add(-3 * time.Hour),
			Turns: []TurnTokens{
				{PromptTokens: 40, TurnTokens: 45},
				{PromptTokens: 80, TurnTokens: 85},
			},
		},
	}

	average := ComputeTurnTokenMetricsForRangeWithMode(sessions, TimeRange{}, StatisticModeAverage)
	require.Len(t, average, 2)
	assert.InDelta(t, 25.0, average[0].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 65.0, average[1].AveragePromptTokens, 0.0001)

	p50 := ComputeTurnTokenMetricsForRangeWithMode(sessions, TimeRange{}, StatisticModeP50)
	require.Len(t, p50, 2)
	assert.Equal(t, 4, p50[0].SampleCount)
	assert.InDelta(t, 20.0, p50[0].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 25.0, p50[0].AverageTurnTokens, 0.0001)
	assert.InDelta(t, 60.0, p50[1].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 65.0, p50[1].AverageTurnTokens, 0.0001)

	p95 := ComputeTurnTokenMetricsForRangeWithMode(sessions, TimeRange{}, StatisticModeP95)
	require.Len(t, p95, 2)
	assert.InDelta(t, 40.0, p95[0].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 45.0, p95[0].AverageTurnTokens, 0.0001)
	assert.InDelta(t, 80.0, p95[1].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 85.0, p95[1].AverageTurnTokens, 0.0001)
}

func TestComputeToolCallsPerSessionStatisticModes(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"s1",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Read": 1}),
		),
		testMeta(
			"s2",
			time.Date(2026, 1, 6, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Read": 3}),
		),
		testMeta(
			"s3",
			time.Date(2026, 1, 7, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Read": 8}),
		),
		testMeta(
			"s4",
			time.Date(2026, 1, 8, 9, 0, 0, 0, time.UTC),
			withToolCounts(map[string]int{"Read": 20}),
		),
	}

	assert.InDelta(t, 8.0, ComputeToolCallsPerSessionStatistic(sessions, StatisticModeAverage), 0.0001)
	assert.InDelta(t, 3.0, ComputeToolCallsPerSessionStatistic(sessions, StatisticModeP50), 0.0001)
	assert.InDelta(t, 20.0, ComputeToolCallsPerSessionStatistic(sessions, StatisticModeP95), 0.0001)
	assert.InDelta(t, 20.0, ComputeToolCallsPerSessionStatistic(sessions, StatisticModeP99), 0.0001)
	assert.InDelta(t, 20.0, ComputeToolCallsPerSessionStatistic(sessions, StatisticModeMax), 0.0001)
}
