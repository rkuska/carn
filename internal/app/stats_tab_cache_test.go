package app

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestFormatReuseInfinite(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "inf", formatReuse(math.Inf(1)))
}

func TestNormalizeReuseDailyRatesClampsInfinitePoints(t *testing.T) {
	t.Parallel()

	rates := []statspkg.DailyRate{
		{Rate: math.Inf(1)},
		{Rate: 3},
	}

	normalized, cap, hasInfinite := normalizeReuseDailyRates(rates)

	assert.True(t, hasInfinite)
	assert.InDelta(t, 3.0, cap, 0.001)
	assert.InDelta(t, 3.0, normalized[0].Rate, 0.001)
	assert.InDelta(t, 3.0, normalized[1].Rate, 0.001)
}

func TestCacheReuseBucketsUseInfiniteDisplay(t *testing.T) {
	t.Parallel()

	buckets := cacheReuseBuckets([]statspkg.CacheDurationBucket{
		{Label: "<5m", ReuseRatio: math.Inf(1), Sessions: 1},
		{Label: "5-15", ReuseRatio: 7, Sessions: 1},
	})

	assert.Equal(t, "inf", buckets[0].Display)
	assert.Equal(t, 700, buckets[0].Count)
}
