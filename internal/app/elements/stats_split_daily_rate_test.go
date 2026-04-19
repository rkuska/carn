package elements

import (
	"fmt"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestGroupedDailyValueBarSlotsKeepDayGapsAndConsistentBarWidths(t *testing.T) {
	t.Parallel()

	buckets := []groupedDailyValueBucket{
		{Series: []DailyValueBucket{{HasValue: true}, {HasValue: true}, {HasValue: false}}},
		{Series: []DailyValueBucket{{HasValue: true}, {HasValue: false}, {HasValue: false}}},
		{Series: []DailyValueBucket{{HasValue: false}, {HasValue: true}, {HasValue: true}}},
		{Series: []DailyValueBucket{{HasValue: false}, {HasValue: false}, {HasValue: false}}},
		{Series: []DailyValueBucket{{HasValue: true}, {HasValue: true}, {HasValue: true}}},
	}

	slots := GroupedDailyValueBarSlots(buckets, 41)

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
	series := sparseSplitDailyValueSeries(start, "a", "b", "c", "d")

	assert.Equal(t, 3, groupedDailyValueBucketCount(series, 6))
}

func TestGroupedDailyRateBucketCountUsesActiveSeriesNotLegendSize(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	series := sparseSplitDailyValueSeries(
		start,
		"2.1.100",
		"2.1.101",
		"2.1.102",
		"2.1.103",
	)

	assert.Equal(t, 4, groupedDailyValueBucketCount(series, 8))
}

func TestDailyRateBucketCountMergesSparseDaysBeforeBreakingGroupSpacing(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)
	rates := []statspkg.DailyRate{
		{Date: start, Rate: 0.8, HasActivity: true},
		{Date: start.AddDate(0, 0, 1), Rate: 0, HasActivity: false},
		{Date: start.AddDate(0, 0, 2), Rate: 0, HasActivity: false},
		{Date: start.AddDate(0, 0, 3), Rate: 0.4, HasActivity: true},
		{Date: start.AddDate(0, 0, 4), Rate: 0.2, HasActivity: true},
	}

	assert.Equal(t, 3, dailyRateBucketCount(rates, 6))
}

func TestGroupedDailyValueBarSlotsAllowInternalGapForSingleBucket(t *testing.T) {
	t.Parallel()

	buckets := []groupedDailyValueBucket{
		{Series: []DailyValueBucket{{HasValue: true}, {HasValue: true}, {HasValue: true}}},
	}

	slots := GroupedDailyValueBarSlots(buckets, 20)

	require.Len(t, slots, 1)
	require.Len(t, slots[0].Bars, 3)
	assert.Greater(t, slots[0].Bars[1].Start-slots[0].Bars[0].End, 0)
}

func TestBucketSplitDailyValuesAveragesOnlyVisibleDays(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)

	got := bucketSplitDailyValues([]statspkg.SplitDailyValueSeries{
		{
			Key: "Claude",
			Values: []statspkg.DailyValue{
				{Date: start, Value: 2, HasValue: true},
				{Date: start.AddDate(0, 0, 1), Value: 0, HasValue: false},
				{Date: start.AddDate(0, 0, 2), Value: 4, HasValue: true},
				{Date: start.AddDate(0, 0, 3), Value: 0, HasValue: false},
			},
		},
	}, 1)

	require.Len(t, got, 1)
	require.Len(t, got[0].Series, 1)
	assert.Equal(t, 3.0, got[0].Series[0].Value)
}

func TestRenderSplitDailyValueChartBodyShowsDotOnlyForFullyEmptyDayBuckets(t *testing.T) {
	t.Parallel()

	dayOne := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)
	theme := NewTheme(true)

	got := ansi.Strip(theme.RenderSplitDailyValueChartBody(
		[]statspkg.SplitDailyValueSeries{
			{
				Key: "Claude",
				Values: []statspkg.DailyValue{
					{Date: dayOne, Value: 8, HasValue: true},
					{Date: dayOne.AddDate(0, 0, 1), Value: 0, HasValue: false},
				},
			},
			{
				Key: "Codex",
				Values: []statspkg.DailyValue{
					{Date: dayOne, Value: 0, HasValue: false},
					{Date: dayOne.AddDate(0, 0, 1), Value: 0, HasValue: false},
				},
			},
		},
		24,
		6,
		map[string]color.Color{
			"Claude": theme.ColorPrimary,
			"Codex":  theme.ColorChartBar,
		},
		func(_ int, v float64) string {
			return strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(assertionFloat(v)), ".0"), ".")
		},
		1,
	))

	assert.Contains(t, got, "█")
	assert.Equal(t, 1, strings.Count(got, "·"))
}

func sparseSplitDailyValueSeries(
	start time.Time,
	keys ...string,
) []statspkg.SplitDailyValueSeries {
	series := make([]statspkg.SplitDailyValueSeries, 0, len(keys))
	for i, key := range keys {
		values := make([]statspkg.DailyValue, 0, len(keys))
		for day := range len(keys) {
			value := statspkg.DailyValue{
				Date: start.AddDate(0, 0, day),
			}
			if day == i {
				value.Value = 9
				value.HasValue = true
			}
			values = append(values, value)
		}
		series = append(series, statspkg.SplitDailyValueSeries{
			Key:    key,
			Values: values,
		})
	}
	return series
}

func assertionFloat(v float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", v), "0"), ".")
}
