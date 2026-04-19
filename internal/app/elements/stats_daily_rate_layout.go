package elements

import statspkg "github.com/rkuska/carn/internal/stats"

func dailyRateBucketCount(rates []statspkg.DailyRate, plotWidth int) int {
	if len(rates) == 0 || plotWidth <= 0 {
		return 0
	}

	return groupedVerticalBarBucketCount(len(rates), plotWidth, func(bucketCount int) []int {
		return dailyRateActiveCounts(BucketDailyRates(rates, bucketCount))
	})
}

func dailyRateActiveCounts(buckets []DailyRateBucket) []int {
	counts := make([]int, 0, len(buckets))
	for _, bucket := range buckets {
		if bucket.HasValue {
			counts = append(counts, 1)
			continue
		}
		counts = append(counts, 0)
	}
	return counts
}

func dailyRateGroupSlots(
	buckets []DailyRateBucket,
	plotWidth int,
) []verticalBarGroupSlot {
	return resolveVerticalBarGroupSlots(dailyRateActiveCounts(buckets), plotWidth)
}

func dailyRateRenderSlot(slot verticalBarGroupSlot, hasValue bool) DailyRateBarSlot {
	if hasValue && len(slot.Bars) > 0 {
		return slot.Bars[0]
	}
	return buildDailyRateBarSlot(slot.Start, slot.End)
}

func dailyRateLabelSlots(slots []verticalBarGroupSlot) []DailyRateBarSlot {
	labelSlots := make([]DailyRateBarSlot, 0, len(slots))
	for _, slot := range slots {
		labelSlots = append(labelSlots, buildDailyRateBarSlot(slot.Start, slot.End))
	}
	return labelSlots
}

func DailyRatePlotWidth(slots []DailyRateBarSlot) int {
	if len(slots) == 0 {
		return 0
	}
	return slots[len(slots)-1].End
}

func DailyRateBarSlots(bucketCount, plotWidth int) []DailyRateBarSlot {
	if bucketCount <= 0 || plotWidth <= 0 {
		return nil
	}

	barWidth := dailyRateBarWidth(bucketCount, plotWidth)
	gaps := dailyRateBarGaps(bucketCount, plotWidth, barWidth)
	slots := make([]DailyRateBarSlot, 0, bucketCount)
	start := 0
	for i := range bucketCount {
		end := start + barWidth
		slots = append(slots, buildDailyRateBarSlot(start, end))
		if i < len(gaps) {
			start = end + gaps[i]
		}
	}
	return slots
}

func dailyRateBarGapWidth(bucketCount, plotWidth int) int {
	if bucketCount <= 1 {
		return 0
	}
	if plotWidth >= bucketCount*2-1 {
		return 1
	}
	return 0
}

func dailyRateBarWidth(bucketCount, plotWidth int) int {
	gapCount := max(bucketCount-1, 0)
	return max((plotWidth-gapCount*dailyRateBarGapWidth(bucketCount, plotWidth))/bucketCount, 1)
}

func dailyRateBarGaps(bucketCount, plotWidth, barWidth int) []int {
	gapCount := max(bucketCount-1, 0)
	if gapCount == 0 {
		return nil
	}

	baseGapWidth := dailyRateBarGapWidth(bucketCount, plotWidth)
	gaps := make([]int, gapCount)
	for i := range gaps {
		gaps[i] = baseGapWidth
	}

	extra := plotWidth - bucketCount*barWidth - gapCount*baseGapWidth
	for i := range extra {
		gaps[(i+1)*gapCount/(extra+1)]++
	}
	return gaps
}

func buildDailyRateBarSlot(start, end int) DailyRateBarSlot {
	width := end - start
	return DailyRateBarSlot{
		Start:  start,
		End:    end,
		Anchor: start + max(width-1, 0)/2,
	}
}
