package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeSessionsAggregatesRatiosAndHistograms(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"s1",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 5, 9, 2, 0, 0, time.UTC)),
			withMainMessages(2),
			withRoleCounts(1, 1),
		),
		testMeta(
			"s2",
			time.Date(2026, 1, 6, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 6, 9, 20, 0, 0, time.UTC)),
			withMainMessages(10),
			withRoleCounts(4, 6),
		),
		testMeta(
			"s3",
			time.Date(2026, 1, 7, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 7, 11, 10, 0, 0, time.UTC)),
			withMainMessages(70),
			withRoleCounts(20, 50),
		),
	}

	got := ComputeSessions(sessions)

	assert.Equal(t, 50*time.Minute+40*time.Second, got.AverageDuration)
	assert.InDelta(t, 27.3333333333, got.AverageMessages, 0.0001)
	assert.Equal(t, 25, got.UserMessageCount)
	assert.Equal(t, 57, got.AssistantMessageCount)
	assert.InDelta(t, 25.0/57.0, got.UserAssistantRatio, 0.0001)
	assert.Equal(t, 1, got.AbandonedCount)
	assert.InDelta(t, 33.3333333333, got.AbandonedRate, 0.0001)
	assert.Equal(t, HistogramBucket{Label: "<5m", Count: 1}, got.DurationHistogram[0])
	assert.Equal(t, HistogramBucket{Label: "15-30", Count: 1}, got.DurationHistogram[2])
	assert.Equal(t, HistogramBucket{Label: "2h+", Count: 1}, got.DurationHistogram[5])
	assert.Equal(t, HistogramBucket{Label: "1-5", Count: 1}, got.MessageHistogram[0])
	assert.Equal(t, HistogramBucket{Label: "5-15", Count: 1}, got.MessageHistogram[1])
	assert.Equal(t, HistogramBucket{Label: "60+", Count: 1}, got.MessageHistogram[4])
}

func TestComputeSessionsBucketBoundariesStayInclusiveAtUpperEdges(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"m5",
			time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 1, 9, 5, 0, 0, time.UTC)),
			withMainMessages(5),
		),
		testMeta(
			"m15",
			time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 1, 10, 15, 0, 0, time.UTC)),
			withMainMessages(15),
		),
		testMeta(
			"m30",
			time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 1, 11, 30, 0, 0, time.UTC)),
			withMainMessages(30),
		),
		testMeta(
			"m60",
			time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC)),
			withMainMessages(60),
		),
		testMeta(
			"m120",
			time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 1, 16, 0, 0, 0, time.UTC)),
			withMainMessages(61),
		),
	}

	got := ComputeSessions(sessions)

	assert.Equal(t, []HistogramBucket{
		{Label: "<5m", Count: 0},
		{Label: "5-15", Count: 1},
		{Label: "15-30", Count: 1},
		{Label: "30-60", Count: 1},
		{Label: "1-2h", Count: 1},
		{Label: "2h+", Count: 1},
	}, got.DurationHistogram)
	assert.Equal(t, []HistogramBucket{
		{Label: "1-5", Count: 1},
		{Label: "5-15", Count: 1},
		{Label: "15-30", Count: 1},
		{Label: "30-60", Count: 1},
		{Label: "60+", Count: 1},
	}, got.MessageHistogram)
}

func TestComputeSessionsDoesNotMarkExactAbandonThresholdsAsAbandoned(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"threshold",
			time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 2, 9, 1, 0, 0, time.UTC)),
			withMainMessages(3),
		),
		testMeta(
			"short",
			time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 2, 10, 0, 59, 0, time.UTC)),
			withMainMessages(3),
		),
		testMeta(
			"few",
			time.Date(2026, 1, 2, 11, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 2, 11, 1, 0, 0, time.UTC)),
			withMainMessages(2),
		),
	}

	got := ComputeSessions(sessions)

	assert.Equal(t, 2, got.AbandonedCount)
	assert.InDelta(t, 66.6667, got.AbandonedRate, 0.0001)
}

func TestComputeTurnTokenMetricsAveragesInputAndTurnCostByPosition(t *testing.T) {
	t.Parallel()

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessage(10, 5),
			assistantUsageMessage(30, 10),
			userMessage(),
			assistantUsageMessage(40, 10),
		}),
		testSession("s2", []message{
			userMessage(),
			assistantUsageMessage(20, 10),
			userMessage(),
			assistantUsageMessage(50, 20),
			assistantUsageMessage(60, 20),
		}),
		testSession("s3", []message{
			assistantUsageMessage(25, 5),
			assistantUsageMessage(35, 10),
			userMessage(),
			assistantUsageMessage(70, 10),
		}),
		testSession("zero", []message{
			userMessage(),
			userMessage(),
		}),
	}

	got := ComputeTurnTokenMetrics(sessions)
	require.Len(t, got, 2)
	assert.Equal(t, 1, got[0].Position)
	assert.Equal(t, 3, got[0].SampleCount)
	assert.InDelta(t, 28.3333333333, got[0].AverageInputTokens, 0.0001)
	assert.InDelta(t, 53.3333333333, got[0].AverageTurnTokens, 0.0001)
	assert.Equal(t, 2, got[1].Position)
	assert.Equal(t, 3, got[1].SampleCount)
	assert.InDelta(t, 56.6666666667, got[1].AverageInputTokens, 0.0001)
	assert.InDelta(t, 93.3333333333, got[1].AverageTurnTokens, 0.0001)
}

func TestComputeTurnTokenMetricsUsesRealTurnBoundariesInsteadOfAssistantSteps(t *testing.T) {
	t.Parallel()

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessage(100, 10),
			assistantUsageMessage(200, 20),
			userMessage(),
			assistantUsageMessage(400, 40),
		}),
		testSession("s2", []message{
			userMessage(),
			assistantUsageMessage(150, 15),
			assistantUsageMessage(300, 30),
			userMessage(),
			assistantUsageMessage(500, 50),
		}),
		testSession("s3", []message{
			assistantUsageMessage(50, 5),
			assistantUsageMessage(250, 25),
			userMessage(),
			assistantUsageMessage(600, 60),
		}),
	}

	got := ComputeTurnTokenMetrics(sessions)
	require.Len(t, got, 2)
	assert.Equal(t, 1, got[0].Position)
	assert.InDelta(t, 250, got[0].AverageInputTokens, 0.0001)
	assert.InDelta(t, 385, got[0].AverageTurnTokens, 0.0001)
	assert.Equal(t, 2, got[1].Position)
	assert.InDelta(t, 500, got[1].AverageInputTokens, 0.0001)
	assert.InDelta(t, 550, got[1].AverageTurnTokens, 0.0001)
}

func TestCollectSessionTurnMetricsCapturesUserTurnsPerSession(t *testing.T) {
	t.Parallel()

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessage(100, 10),
			userToolResultMessage(),
			assistantUsageMessage(200, 20),
			userMessage(),
			assistantUsageMessage(300, 30),
		}),
		testSession("empty", []message{
			userMessage(),
			userMessage(),
		}),
	}

	got := CollectSessionTurnMetrics(sessions)
	require.Len(t, got, 1)
	assert.Equal(t, sessions[0].Meta.Timestamp, got[0].Timestamp)
	assert.Equal(t, []TurnTokens{
		{InputTokens: 200, TurnTokens: 330},
		{InputTokens: 300, TurnTokens: 330},
	}, got[0].Turns)
}

func TestComputeTurnTokenMetricsForRangeReusesCollectedSessionsAcrossDurations(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	sessions := []SessionTurnMetrics{
		{
			Timestamp: now,
			Turns: []TurnTokens{
				{InputTokens: 10, TurnTokens: 15},
				{InputTokens: 20, TurnTokens: 30},
			},
		},
		{
			Timestamp: now.AddDate(0, 0, -10),
			Turns: []TurnTokens{
				{InputTokens: 20, TurnTokens: 25},
				{InputTokens: 30, TurnTokens: 40},
			},
		},
		{
			Timestamp: now.AddDate(0, 0, -20),
			Turns: []TurnTokens{
				{InputTokens: 30, TurnTokens: 35},
				{InputTokens: 40, TurnTokens: 50},
			},
		},
		{
			Timestamp: now.AddDate(0, 0, -50),
			Turns: []TurnTokens{
				{InputTokens: 40, TurnTokens: 45},
				{InputTokens: 50, TurnTokens: 60},
			},
		},
	}

	thirtyDay := ComputeTurnTokenMetricsForRange(sessions, TimeRange{
		Start: now.AddDate(0, 0, -29),
		End:   now,
	})
	require.Len(t, thirtyDay, 2)
	assert.InDelta(t, 20, thirtyDay[0].AverageInputTokens, 0.0001)
	assert.InDelta(t, 25, thirtyDay[0].AverageTurnTokens, 0.0001)
	assert.InDelta(t, 30, thirtyDay[1].AverageInputTokens, 0.0001)
	assert.InDelta(t, 40, thirtyDay[1].AverageTurnTokens, 0.0001)

	allTime := ComputeTurnTokenMetricsForRange(sessions, TimeRange{})
	require.Len(t, allTime, 2)
	assert.InDelta(t, 25, allTime[0].AverageInputTokens, 0.0001)
	assert.InDelta(t, 30, allTime[0].AverageTurnTokens, 0.0001)
	assert.InDelta(t, 35, allTime[1].AverageInputTokens, 0.0001)
	assert.InDelta(t, 45, allTime[1].AverageTurnTokens, 0.0001)
}

func TestComputeTurnTokenMetricsSkipsSparsePositions(t *testing.T) {
	t.Parallel()

	sessions := []session{
		testSession("s1", []message{
			assistantUsageMessage(100, 10),
			assistantUsageMessage(200, 20),
		}),
		testSession("s2", []message{
			assistantUsageMessage(110, 10),
			assistantUsageMessage(210, 20),
		}),
		testSession("s3", []message{
			assistantUsageMessage(120, 10),
		}),
	}

	got := ComputeTurnTokenMetrics(sessions)

	require.Len(t, got, 1)
	assert.Equal(t, 1, got[0].Position)
	assert.Equal(t, 3, got[0].SampleCount)
}
