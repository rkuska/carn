package elements

import (
	"slices"

	statspkg "github.com/rkuska/carn/internal/stats"
)

type groupedTurnMetricValue struct {
	Value    float64
	HasValue bool
}

type groupedTurnMetricBucket struct {
	StartPosition int
	EndPosition   int
	Series        []groupedTurnMetricValue
}

type groupedTurnMetricBarSlot struct {
	Start         int
	End           int
	Anchor        int
	SeriesIndexes []int
	Bars          []DailyRateBarSlot
}

func groupedTurnMetricBucketCount(
	positions []int,
	lookups []map[int]float64,
	graphWidth int,
	mode statspkg.StatisticMode,
) int {
	if len(positions) == 0 || graphWidth <= 0 {
		return 0
	}

	return groupedVerticalBarBucketCount(len(positions), graphWidth, func(bucketCount int) []int {
		return groupedTurnMetricActiveCounts(bucketSplitTurnMetrics(positions, lookups, bucketCount, mode))
	})
}

func bucketSplitTurnMetrics(
	positions []int,
	lookups []map[int]float64,
	bucketCount int,
	mode statspkg.StatisticMode,
) []groupedTurnMetricBucket {
	if len(positions) == 0 || bucketCount <= 0 {
		return nil
	}

	buckets := make([]groupedTurnMetricBucket, 0, bucketCount)
	for i := range bucketCount {
		start := i * len(positions) / bucketCount
		end := (i + 1) * len(positions) / bucketCount
		if end <= start {
			end = start + 1
		}

		bucket := groupedTurnMetricBucket{
			StartPosition: positions[start],
			EndPosition:   positions[end-1],
			Series:        make([]groupedTurnMetricValue, 0, len(lookups)),
		}
		for _, lookup := range lookups {
			bucket.Series = append(bucket.Series, groupedTurnMetricValueForPositions(
				positions[start:end],
				lookup,
				mode,
			))
		}
		buckets = append(buckets, bucket)
	}
	return buckets
}

func collectSplitTurnPositions(series []statspkg.SplitTurnSeries) []int {
	positionSet := make(map[int]bool)
	for _, item := range series {
		for _, metric := range item.Metrics {
			positionSet[metric.Position] = true
		}
	}

	positions := make([]int, 0, len(positionSet))
	for position := range positionSet {
		positions = append(positions, position)
	}
	slices.Sort(positions)
	return positions
}

func splitTurnMetricValueLookup(
	metrics []statspkg.PositionTokenMetrics,
	value func(statspkg.PositionTokenMetrics) float64,
) map[int]float64 {
	lookup := make(map[int]float64, len(metrics))
	for _, metric := range metrics {
		lookup[metric.Position] = value(metric)
	}
	return lookup
}

func groupedTurnMetricValueForPositions(
	positions []int,
	lookup map[int]float64,
	mode statspkg.StatisticMode,
) groupedTurnMetricValue {
	bucket := groupedTurnMetricValue{}
	total := 0.0
	count := 0
	for _, position := range positions {
		current, ok := lookup[position]
		if !ok {
			continue
		}
		if !bucket.HasValue || current > bucket.Value {
			bucket.Value = current
		}
		bucket.HasValue = true
		total += current
		count++
	}
	if !bucket.HasValue {
		return groupedTurnMetricValue{}
	}
	if mode != statspkg.StatisticModeMax {
		bucket.Value = total / float64(count)
	}
	return bucket
}

func groupedTurnMetricBarSlots(
	buckets []groupedTurnMetricBucket,
	graphWidth int,
) []groupedTurnMetricBarSlot {
	if len(buckets) == 0 || graphWidth <= 0 {
		return nil
	}

	baseSlots := resolveVerticalBarGroupSlots(groupedTurnMetricActiveCounts(buckets), graphWidth)
	slots := make([]groupedTurnMetricBarSlot, 0, len(baseSlots))
	for i, slot := range baseSlots {
		slots = append(slots, groupedTurnMetricBarSlot{
			Start:         slot.Start,
			End:           slot.End,
			Anchor:        slot.Anchor,
			SeriesIndexes: groupedTurnMetricActiveSeries(buckets[i]),
			Bars:          slot.Bars,
		})
	}
	return slots
}

func groupedTurnMetricActiveCounts(buckets []groupedTurnMetricBucket) []int {
	counts := make([]int, 0, len(buckets))
	for _, bucket := range buckets {
		n := 0
		for _, item := range bucket.Series {
			if item.HasValue {
				n++
			}
		}
		counts = append(counts, n)
	}
	return counts
}

func groupedTurnMetricActiveSeries(bucket groupedTurnMetricBucket) []int {
	indexes := make([]int, 0, len(bucket.Series))
	for i, item := range bucket.Series {
		if item.HasValue {
			indexes = append(indexes, i)
		}
	}
	return indexes
}
