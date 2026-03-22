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

func TestComputeMessageTokenGrowthAveragesByPositionAndSkipsSparsePositions(t *testing.T) {
	t.Parallel()

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessage(10, 5),
			userMessage(),
			assistantUsageMessage(30, 10),
		}),
		testSession("s2", []message{
			assistantUsageMessage(20, 10),
			assistantUsageMessage(50, 20),
			assistantUsageMessage(60, 20),
		}),
		testSession("s3", []message{
			assistantUsageMessage(25, 5),
			assistantUsageMessage(70, 10),
		}),
		testSession("zero", []message{
			userMessage(),
			userMessage(),
		}),
	}

	got := ComputeMessageTokenGrowth(sessions)
	require.Len(t, got, 2)
	assert.Equal(t, 1, got[0].Position)
	assert.Equal(t, 3, got[0].SampleCount)
	assert.InDelta(t, 25, got[0].AverageTokens, 0.0001)
	assert.Equal(t, 2, got[1].Position)
	assert.Equal(t, 3, got[1].SampleCount)
	assert.InDelta(t, 63.3333333333, got[1].AverageTokens, 0.0001)
}
