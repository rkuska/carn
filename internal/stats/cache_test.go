package stats

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
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
	assert.Empty(t, got.DailyHitRate)
	assert.Empty(t, got.DailyReuseRatio)
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

func TestComputeCacheClaudeMissesStayOutOfCacheWrites(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderClaude),
			withUsage(500, 0, 500, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	assert.InDelta(t, 0.5, got.HitRate, 0.001)
	assert.InDelta(t, 0.5, got.MissRate, 0.001)
	assert.Zero(t, got.TotalCacheWrite)
	assert.Equal(t, 500, got.Main.MissTokens)
	assert.True(t, math.IsInf(got.ReuseRatio, 1))
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

func TestComputeCacheDailyRates(t *testing.T) {
	t.Parallel()

	// Day 1: two sessions
	//   s1: prompt=1000 (I=100, W=100, R=800)
	//   s2: prompt=1000 (I=200, W=200, R=600)
	//   Token-weighted hit rate: (800+600)/(1000+1000) = 0.7
	//   Reuse ratio: (800+600)/(100+200) = 4.667
	// Day 2 (gap day): no sessions → both rates=0
	// Day 3: one session
	//   s3: prompt=500 (I=100, W=100, R=300)
	//   Hit rate: 300/500 = 0.6
	//   Reuse ratio: 300/100 = 3.0
	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(100, 100, 800, 50)),
		testMeta("s2", time.Date(2026, 1, 10, 14, 0, 0, 0, time.UTC),
			withUsage(200, 200, 600, 50)),
		testMeta("s3", time.Date(2026, 1, 12, 9, 0, 0, 0, time.UTC),
			withUsage(100, 100, 300, 50)),
	}

	got := ComputeCache(sessions, TimeRange{})

	require.Len(t, got.DailyHitRate, 3)
	assert.InDelta(t, 0.7, got.DailyHitRate[0].Rate, 0.001)
	assert.True(t, got.DailyHitRate[0].HasActivity)
	assert.InDelta(t, 0.0, got.DailyHitRate[1].Rate, 0.001)
	assert.False(t, got.DailyHitRate[1].HasActivity)
	assert.InDelta(t, 0.6, got.DailyHitRate[2].Rate, 0.001)
	assert.True(t, got.DailyHitRate[2].HasActivity)

	require.Len(t, got.DailyReuseRatio, 3)
	assert.InDelta(t, 4.667, got.DailyReuseRatio[0].Rate, 0.001)
	assert.True(t, got.DailyReuseRatio[0].HasActivity)
	assert.InDelta(t, 0.0, got.DailyReuseRatio[1].Rate, 0.001)
	assert.False(t, got.DailyReuseRatio[1].HasActivity)
	assert.InDelta(t, 3.0, got.DailyReuseRatio[2].Rate, 0.001)
	assert.True(t, got.DailyReuseRatio[2].HasActivity)
}

func TestComputeCacheDailyHitRateZeroPromptDay(t *testing.T) {
	t.Parallel()

	// Session with zero prompt tokens should yield 0 rate
	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(0, 0, 0, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	require.Len(t, got.DailyHitRate, 1)
	assert.InDelta(t, 0.0, got.DailyHitRate[0].Rate, 0.001)
	assert.True(t, got.DailyHitRate[0].HasActivity)
}

func TestComputeCacheDailyReuseRatioIsInfiniteWithoutWrites(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(0, 0, 500, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	require.Len(t, got.DailyReuseRatio, 1)
	assert.True(t, math.IsInf(got.DailyReuseRatio[0].Rate, 1))
	assert.True(t, got.DailyReuseRatio[0].HasActivity)
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

	// <5m bucket: short session, hitRate = 700/1000 = 0.7, reuse = 700/100 = 7.0
	assert.Equal(t, "<5m", got.DurationBuckets[0].Label)
	assert.Equal(t, 1, got.DurationBuckets[0].Sessions)
	assert.InDelta(t, 0.7, got.DurationBuckets[0].HitRate, 0.001)
	assert.InDelta(t, 7.0, got.DurationBuckets[0].ReuseRatio, 0.001)

	// 15-30 bucket: medium session, hitRate = 500/1000 = 0.5, reuse = 500/100 = 5.0
	assert.Equal(t, "15-30", got.DurationBuckets[2].Label)
	assert.Equal(t, 1, got.DurationBuckets[2].Sessions)
	assert.InDelta(t, 0.5, got.DurationBuckets[2].HitRate, 0.001)
	assert.InDelta(t, 5.0, got.DurationBuckets[2].ReuseRatio, 0.001)

	// 2h+ bucket: long session, hitRate = 100/1000 = 0.1, reuse = 100/100 = 1.0
	assert.Equal(t, "2h+", got.DurationBuckets[5].Label)
	assert.Equal(t, 1, got.DurationBuckets[5].Sessions)
	assert.InDelta(t, 0.1, got.DurationBuckets[5].HitRate, 0.001)
	assert.InDelta(t, 1.0, got.DurationBuckets[5].ReuseRatio, 0.001)
}

func TestComputeCacheDurationBucketsAveraging(t *testing.T) {
	t.Parallel()

	// Two sessions in <5m bucket:
	//   s1: R=700, W=100, prompt=1000
	//   s2: R=400, W=100, prompt=1000
	// Token-weighted reuse = (700+400)/(100+100) = 1100/200 = 5.5
	// Session-averaged hit rate = (0.7 + 0.4) / 2 = 0.55
	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 10, 9, 2, 0, 0, time.UTC)),
			withUsage(200, 100, 700, 100)),
		testMeta("s2", time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 10, 10, 3, 0, 0, time.UTC)),
			withUsage(500, 100, 400, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	bucket := got.DurationBuckets[0]
	assert.Equal(t, 2, bucket.Sessions)
	assert.InDelta(t, 0.55, bucket.HitRate, 0.001)
	assert.InDelta(t, 5.5, bucket.ReuseRatio, 0.001)
}

func TestComputeCacheDurationBucketReuseIsInfiniteWithoutWrites(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 10, 9, 2, 0, 0, time.UTC)),
			withUsage(0, 0, 500, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	bucket := got.DurationBuckets[0]
	assert.Equal(t, 1, bucket.Sessions)
	assert.True(t, math.IsInf(bucket.ReuseRatio, 1))
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

func TestCacheWriteProxy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		meta conv.SessionMeta
		want int
	}{
		{
			name: "explicit writes returned as-is",
			meta: conv.SessionMeta{
				Provider: conv.ProviderClaude,
				TotalUsage: conv.TokenUsage{
					InputTokens:              200,
					CacheCreationInputTokens: 100,
					CacheReadInputTokens:     700,
				},
			},
			want: 100,
		},
		{
			name: "legacy codex reads without writes uses InputTokens as proxy",
			meta: conv.SessionMeta{
				Provider: conv.ProviderCodex,
				TotalUsage: conv.TokenUsage{
					InputTokens:          450,
					CacheReadInputTokens: 50,
				},
			},
			want: 450,
		},
		{
			name: "claude reads without writes returns zero",
			meta: conv.SessionMeta{
				Provider: conv.ProviderClaude,
				TotalUsage: conv.TokenUsage{
					InputTokens:          450,
					CacheReadInputTokens: 50,
				},
			},
			want: 0,
		},
		{
			name: "no reads no writes returns zero",
			meta: conv.SessionMeta{
				Provider: conv.ProviderCodex,
				TotalUsage: conv.TokenUsage{
					InputTokens: 500,
				},
			},
			want: 0,
		},
		{
			name: "zero usage returns zero",
			meta: conv.SessionMeta{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, cacheWriteProxy(tt.meta))
		})
	}
}

func TestComputeCacheExplicitWritesUnchanged(t *testing.T) {
	t.Parallel()

	// Claude-style session with explicit cache writes — proxy must not alter.
	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withUsage(200, 100, 700, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	assert.Equal(t, 100, got.TotalCacheWrite)
	// ReuseRatio = 700 / 100 = 7.0
	assert.InDelta(t, 7.0, got.ReuseRatio, 0.001)
}

func TestComputeCacheMixedProviderWriteProxy(t *testing.T) {
	t.Parallel()

	// Claude session: input=200, write=100, read=700, output=100
	// Codex session:  input=450, write=0,   read=50,  output=100
	// Codex writeProxy = 450
	sessions := []sessionMeta{
		testMeta("claude", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderClaude),
			withUsage(200, 100, 700, 100)),
		testMeta("codex", time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderCodex),
			withUsage(450, 0, 50, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	// TotalCacheWrite = 100 (claude) + 450 (codex proxy) = 550
	assert.Equal(t, 550, got.TotalCacheWrite)
	// TotalCacheRead = 700 + 50 = 750
	assert.Equal(t, 750, got.TotalCacheRead)
	// ReuseRatio = 750 / 550
	assert.InDelta(t, 750.0/550.0, got.ReuseRatio, 0.001)
}

func TestComputeCacheWriteProxySegmentInvariant(t *testing.T) {
	t.Parallel()

	// Codex pattern: verify CacheRead + CacheWrite + MissTokens == Prompt
	sessions := []sessionMeta{
		testMeta("s1", time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC),
			withProvider(conv.ProviderCodex),
			withUsage(450, 0, 50, 100)),
	}

	got := ComputeCache(sessions, TimeRange{})

	seg := got.Main
	assert.Equal(t, seg.Prompt, seg.CacheRead+seg.CacheWrite+seg.MissTokens,
		"segment invariant: CacheRead + CacheWrite + MissTokens == Prompt")
}

func withSubagent() func(*sessionMeta) {
	return func(meta *sessionMeta) {
		meta.IsSubagent = true
	}
}
