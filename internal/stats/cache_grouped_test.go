package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestComputeCacheBySplitByVersionAggregatesDailySegmentsRowsAndDurations(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"v1-main-day1",
			time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderClaude),
			withVersion("1.0.0"),
			withLastTimestamp(time.Date(2026, 1, 10, 9, 4, 0, 0, time.UTC)),
			withUsage(100, 100, 800, 10),
		),
		testMeta(
			"v2-sub-day1",
			time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderClaude),
			withVersion("2.0.0"),
			withLastTimestamp(time.Date(2026, 1, 10, 10, 4, 0, 0, time.UTC)),
			withUsage(250, 50, 200, 10),
			withSubagent(),
		),
		testMeta(
			"v1-main-day3",
			time.Date(2026, 1, 12, 9, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderClaude),
			withVersion("1.0.0"),
			withLastTimestamp(time.Date(2026, 1, 12, 11, 30, 0, 0, time.UTC)),
			withUsage(100, 100, 300, 10),
		),
	}

	got := ComputeCacheBySplit(sessions, TimeRange{}, SplitDimensionVersion, nil)

	require.Len(t, got.DailyReadShare, 3)
	assert.Equal(t, time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), got.DailyReadShare[0].Date)
	assert.Equal(t, 1500, got.DailyReadShare[0].Prompt)
	assert.Equal(t, 1000, got.DailyReadShare[0].Total)
	assert.Equal(t, []SplitValue{
		{Key: "1.0.0", Value: 800},
		{Key: "2.0.0", Value: 200},
	}, got.DailyReadShare[0].Splits)
	assert.False(t, got.DailyReadShare[1].HasActivity)
	assert.Equal(t, 0, got.DailyReadShare[1].Prompt)
	assert.Equal(t, 0, got.DailyReadShare[1].Total)

	require.Len(t, got.DailyWriteShare, 3)
	assert.Equal(t, 150, got.DailyWriteShare[0].Total)
	assert.Equal(t, []SplitValue{
		{Key: "1.0.0", Value: 100},
		{Key: "2.0.0", Value: 50},
	}, got.DailyWriteShare[0].Splits)

	require.Len(t, got.SegmentRows, 6)
	assert.Equal(t, SplitNamedStat{
		Name:  "Main cache-rd",
		Total: 1100,
		Splits: []SplitValue{
			{Key: "1.0.0", Value: 1100},
		},
	}, got.SegmentRows[0])
	assert.Equal(t, SplitNamedStat{
		Name:  "Sub  miss",
		Total: 250,
		Splits: []SplitValue{
			{Key: "2.0.0", Value: 250},
		},
	}, got.SegmentRows[5])

	require.Len(t, got.ReadDuration, 6)
	assert.Equal(t, SplitHistogramBucket{
		Label: "<5m",
		Total: 1000,
		Splits: []SplitValue{
			{Key: "1.0.0", Value: 800},
			{Key: "2.0.0", Value: 200},
		},
	}, got.ReadDuration[0])
	assert.Equal(t, SplitHistogramBucket{
		Label: "2h+",
		Total: 300,
		Splits: []SplitValue{
			{Key: "1.0.0", Value: 300},
		},
	}, got.ReadDuration[5])

	require.Len(t, got.WriteDuration, 6)
	assert.Equal(t, SplitHistogramBucket{
		Label: "<5m",
		Total: 150,
		Splits: []SplitValue{
			{Key: "1.0.0", Value: 100},
			{Key: "2.0.0", Value: 50},
		},
	}, got.WriteDuration[0])
	assert.Equal(t, SplitHistogramBucket{
		Label: "2h+",
		Total: 100,
		Splits: []SplitValue{
			{Key: "1.0.0", Value: 100},
		},
	}, got.WriteDuration[5])
}

func TestComputeCacheBySplitHonorsAllowedKeys(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"v1",
			time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderClaude),
			withVersion("1.0.0"),
			withUsage(100, 100, 400, 10),
		),
		testMeta(
			"v2",
			time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderClaude),
			withVersion("2.0.0"),
			withUsage(200, 50, 150, 10),
		),
	}

	got := ComputeCacheBySplit(
		sessions,
		TimeRange{},
		SplitDimensionVersion,
		map[string]bool{"2.0.0": true},
	)

	require.Len(t, got.DailyReadShare, 1)
	assert.Equal(t, 400, got.DailyReadShare[0].Prompt)
	assert.Equal(t, 150, got.DailyReadShare[0].Total)
	assert.Equal(t, []SplitValue{{Key: "2.0.0", Value: 150}}, got.DailyReadShare[0].Splits)
	require.Len(t, got.SegmentRows, 6)
	assert.Equal(t, 150, got.SegmentRows[0].Total)
	assert.Equal(t, []SplitValue{{Key: "2.0.0", Value: 150}}, got.SegmentRows[0].Splits)
}
