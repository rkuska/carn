package stats

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rkuska/carn/internal/app/testutil"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestBucketDailyRatesPreservesInactiveAndZeroDays(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	buckets := bucketDailyRates([]statspkg.DailyRate{
		{Date: start, Rate: 0.8, HasActivity: true},
		{Date: start.AddDate(0, 0, 1), Rate: 0, HasActivity: false},
		{Date: start.AddDate(0, 0, 2), Rate: 0, HasActivity: true},
	}, 3)

	require.Len(t, buckets, 3)
	assert.True(t, buckets[0].HasValue)
	assert.InDelta(t, 0.8, buckets[0].Value, 0.001)

	assert.False(t, buckets[1].HasValue)
	assert.Zero(t, buckets[1].Value)

	assert.True(t, buckets[2].HasValue)
	assert.Zero(t, buckets[2].Value)
}

func TestBucketDailyRatesAveragesActiveDaysWithinCompressedBuckets(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	buckets := bucketDailyRates([]statspkg.DailyRate{
		{Date: start, Rate: 0.8, HasActivity: true},
		{Date: start.AddDate(0, 0, 1), Rate: 0, HasActivity: false},
		{Date: start.AddDate(0, 0, 2), Rate: 0.4, HasActivity: true},
		{Date: start.AddDate(0, 0, 3), Rate: 0, HasActivity: true},
	}, 2)

	require.Len(t, buckets, 2)
	assert.InDelta(t, 0.8, buckets[0].Value, 0.001)
	assert.True(t, buckets[0].HasValue)

	assert.InDelta(t, 0.2, buckets[1].Value, 0.001)
	assert.True(t, buckets[1].HasValue)
}

func TestRenderDailyRateChartBodyUsesDistinctMarkersForInactiveAndZeroDays(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	got := ansi.Strip(renderDailyRateChartBody(testutil.NewTestTheme(), []statspkg.DailyRate{
		{Date: start, Rate: 0.8, HasActivity: true},
		{Date: start.AddDate(0, 0, 1), Rate: 0, HasActivity: false},
		{Date: start.AddDate(0, 0, 2), Rate: 0, HasActivity: true},
	}, 24, 6, testutil.NewTestTheme().ColorChartToken, percentYLabel()))

	assert.Contains(t, got, "█")
	assert.Contains(t, got, "·")
	assert.Contains(t, got, "▁")
	assert.Contains(t, got, "03/10")
	assert.Contains(t, got, "03/12")
}

func TestRenderDailyRateChartBodyCompressesSparseEmptyDaysIntoOneMarkerBucket(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	got := ansi.Strip(renderDailyRateChartBody(testutil.NewTestTheme(), []statspkg.DailyRate{
		{Date: start, Rate: 0.8, HasActivity: true},
		{Date: start.AddDate(0, 0, 1), Rate: 0, HasActivity: false},
		{Date: start.AddDate(0, 0, 2), Rate: 0, HasActivity: false},
		{Date: start.AddDate(0, 0, 3), Rate: 0.4, HasActivity: true},
		{Date: start.AddDate(0, 0, 4), Rate: 0.2, HasActivity: true},
	}, 12, 6, testutil.NewTestTheme().ColorChartToken, percentYLabel()))

	assert.Contains(t, got, "█")
	assert.Equal(t, 1, strings.Count(got, "·"))
}

func TestRenderDailyRateChartBodyFitsWidth(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	got := renderDailyRateChartBody(testutil.NewTestTheme(), []statspkg.DailyRate{
		{Date: start, Rate: 0.97, HasActivity: true},
		{Date: start.AddDate(0, 0, 1), Rate: 0.86, HasActivity: true},
		{Date: start.AddDate(0, 0, 2), Rate: 0, HasActivity: false},
		{Date: start.AddDate(0, 0, 3), Rate: 0.24, HasActivity: true},
		{Date: start.AddDate(0, 0, 4), Rate: 0, HasActivity: true},
		{Date: start.AddDate(0, 0, 5), Rate: 0.91, HasActivity: true},
	}, 28, 7, testutil.NewTestTheme().ColorChartToken, percentYLabel())

	for line := range strings.SplitSeq(ansi.Strip(got), "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), 28)
	}
}

func TestDailyRateBarSlotsExpandAcrossWidePlots(t *testing.T) {
	t.Parallel()

	slots := dailyRateBarSlots(7, 40)

	require.Len(t, slots, 7)
	assert.Greater(t, slots[0].End-slots[0].Start, 1)
	assert.Greater(t, slots[3].End-slots[3].Start, 1)
	assert.Equal(t, 0, slots[0].Start)
	assert.Equal(t, 40, slots[len(slots)-1].End)
}

func TestDailyRateBarSlotsDropGapsOnDensePlots(t *testing.T) {
	t.Parallel()

	slots := dailyRateBarSlots(30, 30)

	require.Len(t, slots, 30)
	assert.Equal(t, 1, slots[0].End-slots[0].Start)
	assert.Equal(t, 1, slots[15].End-slots[15].Start)
}

func TestDailyRateBarSlotsKeepSingleColumnGapsWhenWidthAllows(t *testing.T) {
	t.Parallel()

	slots := dailyRateBarSlots(30, 80)

	require.Len(t, slots, 30)
	assert.Equal(t, slots[0].End-slots[0].Start, slots[15].End-slots[15].Start)
	assert.Equal(t, slots[15].End-slots[15].Start, slots[29].End-slots[29].Start)
	for i := 1; i < len(slots); i++ {
		assert.GreaterOrEqual(t, slots[i].Start-slots[i-1].End, 1)
	}
}

func TestRenderDailyRateChartBodyUsesWideBarsWhenSpaceAllows(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	rates := make([]statspkg.DailyRate, 0, 7)
	for i := range 7 {
		rates = append(rates, statspkg.DailyRate{
			Date:        start.AddDate(0, 0, i),
			Rate:        0.9,
			HasActivity: true,
		})
	}

	got := ansi.Strip(renderDailyRateChartBody(
		testutil.NewTestTheme(),
		rates,
		48,
		6,
		testutil.NewTestTheme().ColorChartToken,
		percentYLabel(),
	))

	assert.Contains(t, got, "██")
}
