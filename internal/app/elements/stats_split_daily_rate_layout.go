package elements

import statspkg "github.com/rkuska/carn/internal/stats"

type groupedDailyRateBarSlot struct {
	Start         int
	End           int
	Anchor        int
	SeriesIndexes []int
	Bars          []DailyRateBarSlot
}

type groupedDailyRateLayout struct {
	BarWidth int
	BarGap   int
	DayGap   int
	LeftPad  int
}

type groupedDailyRateLayoutTotals struct {
	TotalBars    int
	TotalBarGaps int
	EmptyGroups  int
	DayGapCount  int
}

func groupedDailyRateBucketCount(
	series []statspkg.SplitDailyRateSeries,
	plotWidth int,
) int {
	if len(series) == 0 || len(series[0].Rates) == 0 || plotWidth <= 0 {
		return 0
	}
	totalDays := len(series[0].Rates)
	if totalDays == 1 {
		return 1
	}

	maxGroups := min(totalDays, plotWidth)
	for bucketCount := maxGroups; bucketCount >= 1; bucketCount-- {
		buckets := bucketSplitDailyRates(series, bucketCount)
		if _, ok := resolveGroupedDailyRateLayout(
			groupedDailyRateActiveCounts(buckets),
			plotWidth,
		); ok {
			return bucketCount
		}
	}
	return 1
}

func GroupedDailyRateBarSlots(
	buckets []groupedDailyRateBucket,
	plotWidth int,
) []groupedDailyRateBarSlot {
	if len(buckets) == 0 || plotWidth <= 0 {
		return nil
	}

	activeCounts := groupedDailyRateActiveCounts(buckets)
	layout, ok := resolveGroupedDailyRateLayout(activeCounts, plotWidth)
	if !ok {
		return nil
	}

	slots := make([]groupedDailyRateBarSlot, 0, len(buckets))
	start := layout.LeftPad
	for i, bucket := range buckets {
		activeSeries := groupedDailyRateActiveSeries(bucket)
		groupWidth := groupedDailyRateGroupWidth(layout, len(activeSeries))
		end := start + groupWidth
		bars := groupedDailyRateGroupBars(layout, len(activeSeries))
		offsetBars := make([]DailyRateBarSlot, 0, len(bars))
		for _, bar := range bars {
			offsetBars = append(
				offsetBars,
				groupedDailyRateOffsetBarSlot(start, bar),
			)
		}
		slots = append(slots, groupedDailyRateBarSlot{
			Start:         start,
			End:           end,
			Anchor:        start + max(groupWidth-1, 0)/2,
			SeriesIndexes: activeSeries,
			Bars:          offsetBars,
		})
		if i < len(buckets)-1 {
			start = end + layout.DayGap
		}
	}

	return slots
}

func groupedDailyRateGroupBars(
	layout groupedDailyRateLayout,
	seriesCount int,
) []DailyRateBarSlot {
	if layout.BarWidth <= 0 || seriesCount <= 0 {
		return nil
	}

	slots := make([]DailyRateBarSlot, 0, seriesCount)
	start := 0
	for range seriesCount {
		end := start + layout.BarWidth
		slots = append(slots, buildDailyRateBarSlot(start, end))
		start = end + layout.BarGap
	}
	return slots
}

func resolveGroupedDailyRateLayout(
	activeCounts []int,
	plotWidth int,
) (groupedDailyRateLayout, bool) {
	if len(activeCounts) == 0 || plotWidth <= 0 {
		return groupedDailyRateLayout{}, false
	}

	totals := groupedDailyRateLayoutTotalsForCounts(activeCounts)
	best := groupedDailyRateLayout{}
	bestUsed := -1
	found := false
	for _, candidate := range groupedDailyRateLayoutCandidates(totals) {
		layout, used, ok := groupedDailyRateLayoutUsage(
			candidate,
			totals,
			plotWidth,
		)
		if !ok {
			continue
		}
		if groupedDailyRateLayoutBetter(layout, used, best, bestUsed, found) {
			best = layout
			bestUsed = used
			found = true
		}
	}
	if !found {
		return groupedDailyRateLayout{}, false
	}

	groupedDailyRateDistributeRemainder(&best, plotWidth-bestUsed, totals)
	return best, true
}

func groupedDailyRateLayoutTotalsForCounts(
	activeCounts []int,
) groupedDailyRateLayoutTotals {
	totals := groupedDailyRateLayoutTotals{
		DayGapCount: max(len(activeCounts)-1, 0),
	}
	for _, count := range activeCounts {
		if count <= 0 {
			totals.EmptyGroups++
			continue
		}
		totals.TotalBars += count
		totals.TotalBarGaps += max(count-1, 0)
	}
	return totals
}

func groupedDailyRateLayoutCandidates(
	totals groupedDailyRateLayoutTotals,
) []groupedDailyRateLayout {
	if totals.DayGapCount == 0 {
		return []groupedDailyRateLayout{
			{DayGap: 0, BarGap: 1},
			{DayGap: 0, BarGap: 0},
		}
	}
	return []groupedDailyRateLayout{{DayGap: 1, BarGap: 0}}
}

func groupedDailyRateLayoutUsage(
	candidate groupedDailyRateLayout,
	totals groupedDailyRateLayoutTotals,
	plotWidth int,
) (groupedDailyRateLayout, int, bool) {
	candidate = groupedDailyRateNormalizeCandidate(candidate, totals)
	baseWidth := groupedDailyRateBaseWidth(candidate, totals)
	if totals.TotalBars == 0 {
		used, ok := groupedDailyRateZeroBarUsage(baseWidth, plotWidth)
		return candidate, used, ok
	}

	barWidth, used, ok := groupedDailyRateActiveBarUsage(
		baseWidth,
		totals.TotalBars,
		plotWidth,
	)
	if !ok {
		return groupedDailyRateLayout{}, 0, false
	}
	candidate.BarWidth = barWidth
	return candidate, used, true
}

func groupedDailyRateNormalizeCandidate(
	candidate groupedDailyRateLayout,
	totals groupedDailyRateLayoutTotals,
) groupedDailyRateLayout {
	if totals.DayGapCount == 0 {
		candidate.DayGap = 0
	}
	if totals.TotalBarGaps == 0 {
		candidate.BarGap = 0
	}
	return candidate
}

func groupedDailyRateBaseWidth(
	candidate groupedDailyRateLayout,
	totals groupedDailyRateLayoutTotals,
) int {
	return totals.EmptyGroups +
		candidate.DayGap*totals.DayGapCount +
		candidate.BarGap*totals.TotalBarGaps
}

func groupedDailyRateZeroBarUsage(
	baseWidth, plotWidth int,
) (int, bool) {
	if baseWidth > plotWidth {
		return 0, false
	}
	return baseWidth, true
}

func groupedDailyRateActiveBarUsage(
	baseWidth, totalBars, plotWidth int,
) (int, int, bool) {
	if plotWidth-baseWidth < totalBars {
		return 0, 0, false
	}
	barWidth := (plotWidth - baseWidth) / totalBars
	if barWidth < 1 {
		return 0, 0, false
	}
	return barWidth, baseWidth + barWidth*totalBars, true
}

func groupedDailyRateLayoutBetter(
	candidate groupedDailyRateLayout,
	used int,
	best groupedDailyRateLayout,
	bestUsed int,
	found bool,
) bool {
	if !found {
		return true
	}
	if candidate.BarWidth != best.BarWidth {
		return candidate.BarWidth > best.BarWidth
	}
	if used != bestUsed {
		return used > bestUsed
	}
	if candidate.DayGap != best.DayGap {
		return candidate.DayGap > best.DayGap
	}
	return candidate.BarGap > best.BarGap
}

func groupedDailyRateDistributeRemainder(
	layout *groupedDailyRateLayout,
	remaining int,
	totals groupedDailyRateLayoutTotals,
) {
	if totals.DayGapCount > 0 {
		layout.DayGap += remaining / totals.DayGapCount
		remaining %= totals.DayGapCount
	} else if totals.TotalBarGaps > 0 {
		layout.BarGap += remaining / totals.TotalBarGaps
		remaining %= totals.TotalBarGaps
	}
	layout.LeftPad = remaining / 2
}

func groupedDailyRateActiveCounts(
	buckets []groupedDailyRateBucket,
) []int {
	counts := make([]int, 0, len(buckets))
	for _, bucket := range buckets {
		counts = append(counts, len(groupedDailyRateActiveSeries(bucket)))
	}
	return counts
}

func groupedDailyRateActiveSeries(bucket groupedDailyRateBucket) []int {
	indexes := make([]int, 0, len(bucket.Series))
	for i, series := range bucket.Series {
		if series.HasActivity {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func groupedDailyRateGroupWidth(
	layout groupedDailyRateLayout,
	seriesCount int,
) int {
	if seriesCount <= 0 {
		return 1
	}
	return layout.BarWidth*seriesCount + layout.BarGap*max(seriesCount-1, 0)
}

func groupedDailyRateOffsetBarSlot(offset int, slot DailyRateBarSlot) DailyRateBarSlot {
	return DailyRateBarSlot{
		Start:  offset + slot.Start,
		End:    offset + slot.End,
		Anchor: offset + slot.Anchor,
	}
}
