package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestComputeCacheByVersionAggregatesDailySegmentsRowsAndDurations(t *testing.T) {
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
		testMeta(
			"codex",
			time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderCodex),
			withVersion("9.9.9"),
			withUsage(100, 100, 100, 10),
		),
	}

	got := ComputeCacheByVersion(sessions, TimeRange{}, conv.ProviderClaude, nil)

	require.Len(t, got.DailyReadShare, 3)
	assert.Equal(t, time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), got.DailyReadShare[0].Date)
	assert.Equal(t, 1500, got.DailyReadShare[0].Prompt)
	assert.Equal(t, 1000, got.DailyReadShare[0].Total)
	assert.Equal(t, []VersionValue{
		{Version: "1.0.0", Value: 800},
		{Version: "2.0.0", Value: 200},
	}, got.DailyReadShare[0].Versions)
	assert.False(t, got.DailyReadShare[1].HasActivity)
	assert.Equal(t, 0, got.DailyReadShare[1].Prompt)
	assert.Equal(t, 0, got.DailyReadShare[1].Total)

	require.Len(t, got.DailyWriteShare, 3)
	assert.Equal(t, 150, got.DailyWriteShare[0].Total)
	assert.Equal(t, []VersionValue{
		{Version: "1.0.0", Value: 100},
		{Version: "2.0.0", Value: 50},
	}, got.DailyWriteShare[0].Versions)

	require.Len(t, got.SegmentRows, 6)
	assert.Equal(t, GroupedNamedStat{
		Name:  "Main cache-rd",
		Total: 1100,
		Versions: []VersionValue{
			{Version: "1.0.0", Value: 1100},
		},
	}, got.SegmentRows[0])
	assert.Equal(t, GroupedNamedStat{
		Name:  "Sub  miss",
		Total: 250,
		Versions: []VersionValue{
			{Version: "2.0.0", Value: 250},
		},
	}, got.SegmentRows[5])

	require.Len(t, got.ReadDuration, 6)
	assert.Equal(t, GroupedHistogramBucket{
		Label: "<5m",
		Total: 1000,
		Versions: []VersionValue{
			{Version: "1.0.0", Value: 800},
			{Version: "2.0.0", Value: 200},
		},
	}, got.ReadDuration[0])
	assert.Equal(t, GroupedHistogramBucket{
		Label: "2h+",
		Total: 300,
		Versions: []VersionValue{
			{Version: "1.0.0", Value: 300},
		},
	}, got.ReadDuration[5])

	require.Len(t, got.WriteDuration, 6)
	assert.Equal(t, GroupedHistogramBucket{
		Label: "<5m",
		Total: 150,
		Versions: []VersionValue{
			{Version: "1.0.0", Value: 100},
			{Version: "2.0.0", Value: 50},
		},
	}, got.WriteDuration[0])
	assert.Equal(t, GroupedHistogramBucket{
		Label: "2h+",
		Total: 100,
		Versions: []VersionValue{
			{Version: "1.0.0", Value: 100},
		},
	}, got.WriteDuration[5])
}

func TestComputeCacheByVersionHonorsSelectedVersions(t *testing.T) {
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

	got := ComputeCacheByVersion(
		sessions,
		TimeRange{},
		conv.ProviderClaude,
		map[string]bool{"2.0.0": true},
	)

	require.Len(t, got.DailyReadShare, 1)
	assert.Equal(t, 400, got.DailyReadShare[0].Prompt)
	assert.Equal(t, 150, got.DailyReadShare[0].Total)
	assert.Equal(t, []VersionValue{{Version: "2.0.0", Value: 150}}, got.DailyReadShare[0].Versions)
	require.Len(t, got.SegmentRows, 6)
	assert.Equal(t, 150, got.SegmentRows[0].Total)
	assert.Equal(t, []VersionValue{{Version: "2.0.0", Value: 150}}, got.SegmentRows[0].Versions)
}
