package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeCacheEmptySessions(t *testing.T) {
	t.Parallel()

	got := ComputeCache(nil, TimeRange{})

	assert.Zero(t, got.TotalCacheRead)
	assert.Zero(t, got.TotalCacheWrite)
	assert.Zero(t, got.TotalPrompt)
	assert.Zero(t, got.HitRate)
	assert.Zero(t, got.WriteRate)
	assert.Zero(t, got.MissRate)
	assert.Zero(t, got.ReuseRatio)
	assert.Zero(t, got.Main.SessionCount)
	assert.Zero(t, got.Subagent.SessionCount)
	assert.Empty(t, got.DailyCacheRead)
	assert.Empty(t, got.DailyCacheWrite)
	assert.Empty(t, got.DurationBuckets)
}

func TestComputeCacheTotals(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(1000, 200, 800, 300)),
		testMeta("s2", time.Date(2026, 1, 11, 9, 0, 0, 0, time.UTC),
			withUsage(500, 100, 400, 200)),
	}

	got := ComputeCache(sessions, TimeRange{})

	assert.Equal(t, 1200, got.TotalCacheRead)
	assert.Equal(t, 300, got.TotalCacheWrite)
	assert.Equal(t, 3000, got.TotalPrompt)
}

func TestComputeCacheHitRate(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(200, 100, 700, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	// HitRate = 700 / (200+100+700) = 0.7
	assert.InDelta(t, 0.7, got.HitRate, 0.001)
}

func TestComputeCacheMissRate(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(300, 100, 600, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	// MissRate = 300 / (300+100+600) = 0.3
	assert.InDelta(t, 0.3, got.MissRate, 0.001)
}

func TestComputeCacheWriteRate(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(300, 200, 500, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	// WriteRate = 200 / (300+200+500) = 0.2
	assert.InDelta(t, 0.2, got.WriteRate, 0.001)
}

func TestComputeCacheReuseRatio(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(100, 200, 1000, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	// ReuseRatio = 1000 / 200 = 5.0
	assert.InDelta(t, 5.0, got.ReuseRatio, 0.001)
}

func TestComputeCacheZeroDenominators(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(0, 0, 0, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	assert.Zero(t, got.HitRate)
	assert.Zero(t, got.WriteRate)
	assert.Zero(t, got.MissRate)
	assert.Zero(t, got.ReuseRatio)
}

func TestComputeCacheZeroCacheWrite(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(500, 0, 500, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	assert.Zero(t, got.ReuseRatio)
	assert.InDelta(t, 0.5, got.HitRate, 0.001)
}

func TestComputeCacheMainSubagentSplit(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("main1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(200, 100, 700, 100)),
		testMeta("sub1", time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
			withUsage(300, 50, 150, 80),
			withSubagent()),
		testMeta("main2", time.Date(2026, 1, 11, 9, 0, 0, 0, time.UTC),
			withUsage(100, 50, 350, 50)),
	}

	got := ComputeCache(sessions, TimeRange{})

	assert.Equal(t, 2, got.Main.SessionCount)
	assert.Equal(t, 1050, got.Main.CacheRead)
	assert.Equal(t, 150, got.Main.CacheWrite)
	assert.Equal(t, 1500, got.Main.Prompt)
	assert.InDelta(t, 0.7, got.Main.HitRate, 0.001)

	assert.Equal(t, 1, got.Subagent.SessionCount)
	assert.Equal(t, 150, got.Subagent.CacheRead)
	assert.Equal(t, 50, got.Subagent.CacheWrite)
	assert.Equal(t, 500, got.Subagent.Prompt)
	assert.InDelta(t, 0.3, got.Subagent.HitRate, 0.001)
}

func TestComputeCacheSubagentOnlySessions(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("sub1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(100, 50, 350, 80),
			withSubagent()),
	}

	got := ComputeCache(sessions, TimeRange{})

	assert.Zero(t, got.Main.SessionCount)
	assert.Equal(t, 1, got.Subagent.SessionCount)
	assert.Equal(t, 350, got.Subagent.CacheRead)
	assert.InDelta(t, 0.7, got.Subagent.HitRate, 0.001)
}

func TestComputeCacheDailyReadSeries(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(100, 50, 400, 50)),
		testMeta("s2", time.Date(2026, 1, 10, 14, 0, 0, 0, time.UTC),
			withUsage(100, 50, 600, 50)),
		testMeta("s3", time.Date(2026, 1, 12, 9, 0, 0, 0, time.UTC),
			withUsage(100, 50, 200, 50)),
	}

	got := ComputeCache(sessions, TimeRange{})

	require.Len(t, got.DailyCacheRead, 3)
	assert.Equal(t, 1000, got.DailyCacheRead[0].Count)
	assert.Equal(t, 0, got.DailyCacheRead[1].Count)
	assert.Equal(t, 200, got.DailyCacheRead[2].Count)
}

func TestComputeCacheDailyWriteSeries(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(100, 300, 400, 50)),
		testMeta("s2", time.Date(2026, 1, 11, 9, 0, 0, 0, time.UTC),
			withUsage(100, 150, 400, 50)),
	}

	got := ComputeCache(sessions, TimeRange{})

	require.Len(t, got.DailyCacheWrite, 2)
	assert.Equal(t, 300, got.DailyCacheWrite[0].Count)
	assert.Equal(t, 150, got.DailyCacheWrite[1].Count)
}

func TestComputeCacheDurationBuckets(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("short", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 10, 9, 3, 0, 0, time.UTC)),
			withUsage(200, 100, 700, 100)),
		testMeta("medium", time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 10, 10, 20, 0, 0, time.UTC)),
			withUsage(400, 100, 500, 100)),
		testMeta("long", time.Date(2026, 1, 10, 11, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 10, 14, 0, 0, 0, time.UTC)),
			withUsage(800, 100, 100, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	require.Len(t, got.DurationBuckets, 6)

	// <5m bucket: short session, hitRate = 700/1000 = 0.7
	assert.Equal(t, "<5m", got.DurationBuckets[0].Label)
	assert.Equal(t, 1, got.DurationBuckets[0].Sessions)
	assert.InDelta(t, 0.7, got.DurationBuckets[0].HitRate, 0.001)
	assert.Equal(t, 200, got.DurationBuckets[0].MissTokens)

	// 15-30 bucket: medium session, hitRate = 500/1000 = 0.5
	assert.Equal(t, "15-30", got.DurationBuckets[2].Label)
	assert.Equal(t, 1, got.DurationBuckets[2].Sessions)
	assert.InDelta(t, 0.5, got.DurationBuckets[2].HitRate, 0.001)
	assert.Equal(t, 400, got.DurationBuckets[2].MissTokens)

	// 2h+ bucket: long session, hitRate = 100/1000 = 0.1
	assert.Equal(t, "2h+", got.DurationBuckets[5].Label)
	assert.Equal(t, 1, got.DurationBuckets[5].Sessions)
	assert.InDelta(t, 0.1, got.DurationBuckets[5].HitRate, 0.001)
	assert.Equal(t, 800, got.DurationBuckets[5].MissTokens)
}

func TestComputeCacheDurationBucketsAveraging(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 10, 9, 2, 0, 0, time.UTC)),
			withUsage(200, 100, 700, 100)),
		testMeta("s2", time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 10, 10, 3, 0, 0, time.UTC)),
			withUsage(500, 100, 400, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	// Both in <5m bucket: avg hitRate = (0.7 + 0.4)/2 = 0.55
	// avg missTokens = (200 + 500) / 2 = 350
	bucket := got.DurationBuckets[0]
	assert.Equal(t, 2, bucket.Sessions)
	assert.InDelta(t, 0.55, bucket.HitRate, 0.001)
	assert.Equal(t, 350, bucket.MissTokens)
}

func TestComputeCacheNoCacheTokens(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(500, 0, 0, 200)),
	}

	got := ComputeCache(sessions, TimeRange{})

	assert.Equal(t, 0, got.TotalCacheRead)
	assert.Equal(t, 0, got.TotalCacheWrite)
	assert.Equal(t, 500, got.TotalPrompt)
	assert.Zero(t, got.HitRate)
	assert.Zero(t, got.ReuseRatio)
	assert.InDelta(t, 1.0, got.MissRate, 0.001)
}

func withSubagent() func(*sessionMeta) {
	return func(meta *sessionMeta) {
		meta.IsSubagent = true
	}
}
