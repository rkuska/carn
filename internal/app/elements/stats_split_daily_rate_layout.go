package elements

import statspkg "github.com/rkuska/carn/internal/stats"

type groupedDailyValueBarSlot struct {
	Start         int
	End           int
	Anchor        int
	SeriesIndexes []int
	Bars          []DailyRateBarSlot
}

func groupedDailyValueBucketCount(
	series []statspkg.SplitDailyValueSeries,
	plotWidth int,
) int {
	if len(series) == 0 || len(series[0].Values) == 0 || plotWidth <= 0 {
		return 0
	}

	return groupedVerticalBarBucketCount(len(series[0].Values), plotWidth, func(bucketCount int) []int {
		return groupedDailyValueActiveCounts(bucketSplitDailyValues(series, bucketCount))
	})
}

func GroupedDailyValueBarSlots(
	buckets []groupedDailyValueBucket,
	plotWidth int,
) []groupedDailyValueBarSlot {
	if len(buckets) == 0 || plotWidth <= 0 {
		return nil
	}

	baseSlots := resolveVerticalBarGroupSlots(groupedDailyValueActiveCounts(buckets), plotWidth)
	slots := make([]groupedDailyValueBarSlot, 0, len(baseSlots))
	for i, slot := range baseSlots {
		slots = append(slots, groupedDailyValueBarSlot{
			Start:         slot.Start,
			End:           slot.End,
			Anchor:        slot.Anchor,
			SeriesIndexes: groupedDailyValueActiveSeries(buckets[i]),
			Bars:          slot.Bars,
		})
	}
	return slots
}

func groupedDailyValueActiveCounts(buckets []groupedDailyValueBucket) []int {
	counts := make([]int, 0, len(buckets))
	for _, bucket := range buckets {
		n := 0
		for _, series := range bucket.Series {
			if series.HasValue {
				n++
			}
		}
		counts = append(counts, n)
	}
	return counts
}

func groupedDailyValueActiveSeries(bucket groupedDailyValueBucket) []int {
	indexes := make([]int, 0, len(bucket.Series))
	for i, series := range bucket.Series {
		if series.HasValue {
			indexes = append(indexes, i)
		}
	}
	return indexes
}
