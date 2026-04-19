package elements

import (
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestGroupedDailyRateBarSlotsKeepDayGapsAndConsistentBarWidths(t *testing.T) {
	t.Parallel()

	buckets := []groupedDailyRateBucket{
		{Series: []DailyRateBucket{{HasActivity: true}, {HasActivity: true}, {HasActivity: false}}},
		{Series: []DailyRateBucket{{HasActivity: true}, {HasActivity: false}, {HasActivity: false}}},
		{Series: []DailyRateBucket{{HasActivity: false}, {HasActivity: true}, {HasActivity: true}}},
		{Series: []DailyRateBucket{{HasActivity: false}, {HasActivity: false}, {HasActivity: false}}},
		{Series: []DailyRateBucket{{HasActivity: true}, {HasActivity: true}, {HasActivity: true}}},
	}

	slots := GroupedDailyRateBarSlots(buckets, 41)

	require.Len(t, slots, len(buckets))

	dayGap := slots[1].Start - slots[0].End
	barWidth := slots[0].Bars[0].End - slots[0].Bars[0].Start
	barGap := slots[0].Bars[1].Start - slots[0].Bars[0].End
	assert.GreaterOrEqual(t, dayGap, 1)
	assert.Equal(t, 0, barGap)
	assert.Greater(t, dayGap, barGap)

	for i, slot := range slots {
		if i == 0 {
			require.Len(t, slot.Bars, 2)
		} else {
			assert.Equal(t, dayGap, slot.Start-slots[i-1].End)
		}
		for j, bar := range slot.Bars {
			assert.Equal(t, barWidth, bar.End-bar.Start)
			assert.GreaterOrEqual(t, bar.Start, slot.Start)
			assert.LessOrEqual(t, bar.End, slot.End)
			if j == 0 {
				continue
			}
			assert.Equal(t, barGap, bar.Start-slot.Bars[j-1].End)
		}
	}

	assert.LessOrEqual(t, slots[len(slots)-1].End, 41)
}

func TestGroupedDailyRateBucketCountReducesToPreserveDayGaps(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	series := sparseSplitDailyRateSeries(start, "a", "b", "c", "d")

	assert.Equal(t, 3, groupedDailyRateBucketCount(series, 6))
}

func TestGroupedDailyRateBucketCountUsesActiveSeriesNotLegendSize(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	series := sparseSplitDailyRateSeries(
		start,
		"2.1.100",
		"2.1.101",
		"2.1.102",
		"2.1.103",
	)

	assert.Equal(t, 4, groupedDailyRateBucketCount(series, 8))
}

func TestRenderSplitDailyRateChartBodyShowsDotOnlyForFullyEmptyDayBuckets(t *testing.T) {
	t.Parallel()

	dayOne := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)
	theme := NewTheme(true)

	got := ansi.Strip(theme.RenderSplitDailyRateChartBody(
		[]statspkg.SplitDailyRateSeries{
			{
				Key: "Claude",
				Rates: []statspkg.DailyRate{
					{Date: dayOne, Rate: 0.8, HasActivity: true},
					{Date: dayOne.AddDate(0, 0, 1), Rate: 0.0, HasActivity: false},
				},
			},
			{
				Key: "Codex",
				Rates: []statspkg.DailyRate{
					{Date: dayOne, Rate: 0.0, HasActivity: false},
					{Date: dayOne.AddDate(0, 0, 1), Rate: 0.0, HasActivity: false},
				},
			},
		},
		24,
		6,
		map[string]color.Color{
			"Claude": theme.ColorPrimary,
			"Codex":  theme.ColorChartBar,
		},
	))

	assert.Contains(t, got, "█")
	assert.Equal(t, 1, strings.Count(got, "·"))
}

func sparseSplitDailyRateSeries(
	start time.Time,
	keys ...string,
) []statspkg.SplitDailyRateSeries {
	series := make([]statspkg.SplitDailyRateSeries, 0, len(keys))
	for i, key := range keys {
		rates := make([]statspkg.DailyRate, 0, len(keys))
		for day := range len(keys) {
			rate := statspkg.DailyRate{
				Date: start.AddDate(0, 0, day),
			}
			if day == i {
				rate.Rate = 0.9
				rate.HasActivity = true
			}
			rates = append(rates, rate)
		}
		series = append(series, statspkg.SplitDailyRateSeries{
			Key:   key,
			Rates: rates,
		})
	}
	return series
}
