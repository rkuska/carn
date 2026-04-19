package elements

type verticalBarGroupSlot struct {
	Start  int
	End    int
	Anchor int
	Bars   []DailyRateBarSlot
}

type verticalBarLayout struct {
	BarWidth int
	BarGap   int
	GroupGap int
	LeftPad  int
}

type verticalBarLayoutTotals struct {
	TotalBars     int
	TotalBarGaps  int
	EmptyGroups   int
	GroupGapCount int
}

func groupedVerticalBarBucketCount(
	maxGroups, plotWidth int,
	activeCounts func(bucketCount int) []int,
) int {
	if maxGroups <= 0 || plotWidth <= 0 {
		return 0
	}
	if maxGroups == 1 {
		return 1
	}

	for bucketCount := min(maxGroups, plotWidth); bucketCount >= 1; bucketCount-- {
		if _, ok := resolveVerticalBarLayout(activeCounts(bucketCount), plotWidth); ok {
			return bucketCount
		}
	}
	return 1
}

func resolveVerticalBarGroupSlots(
	activeCounts []int,
	plotWidth int,
) []verticalBarGroupSlot {
	if len(activeCounts) == 0 || plotWidth <= 0 {
		return nil
	}

	layout, ok := resolveVerticalBarLayout(activeCounts, plotWidth)
	if !ok {
		return nil
	}

	slots := make([]verticalBarGroupSlot, 0, len(activeCounts))
	start := layout.LeftPad
	for i, count := range activeCounts {
		groupWidth := verticalBarGroupWidth(layout, count)
		end := start + groupWidth
		slots = append(slots, verticalBarGroupSlot{
			Start:  start,
			End:    end,
			Anchor: start + max(groupWidth-1, 0)/2,
			Bars:   verticalBarGroupBars(layout, start, count),
		})
		if i < len(activeCounts)-1 {
			start = end + layout.GroupGap
		}
	}
	return slots
}

func resolveVerticalBarLayout(
	activeCounts []int,
	plotWidth int,
) (verticalBarLayout, bool) {
	if len(activeCounts) == 0 || plotWidth <= 0 {
		return verticalBarLayout{}, false
	}

	totals := verticalBarLayoutTotalsForCounts(activeCounts)
	best := verticalBarLayout{}
	bestUsed := -1
	found := false
	for _, candidate := range verticalBarLayoutCandidates(totals) {
		layout, used, ok := verticalBarLayoutUsage(candidate, totals, plotWidth)
		if !ok {
			continue
		}
		if verticalBarLayoutBetter(layout, used, best, bestUsed, found) {
			best = layout
			bestUsed = used
			found = true
		}
	}
	if !found {
		return verticalBarLayout{}, false
	}

	verticalBarDistributeRemainder(&best, plotWidth-bestUsed, totals)
	return best, true
}

func verticalBarLayoutTotalsForCounts(
	activeCounts []int,
) verticalBarLayoutTotals {
	totals := verticalBarLayoutTotals{
		GroupGapCount: max(len(activeCounts)-1, 0),
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

func verticalBarLayoutCandidates(
	totals verticalBarLayoutTotals,
) []verticalBarLayout {
	if totals.GroupGapCount == 0 {
		return []verticalBarLayout{
			{GroupGap: 0, BarGap: 1},
			{GroupGap: 0, BarGap: 0},
		}
	}
	return []verticalBarLayout{{GroupGap: 1, BarGap: 0}}
}

func verticalBarLayoutUsage(
	candidate verticalBarLayout,
	totals verticalBarLayoutTotals,
	plotWidth int,
) (verticalBarLayout, int, bool) {
	candidate = verticalBarNormalizeCandidate(candidate, totals)
	baseWidth := verticalBarBaseWidth(candidate, totals)
	if totals.TotalBars == 0 {
		if baseWidth > plotWidth {
			return verticalBarLayout{}, 0, false
		}
		return candidate, baseWidth, true
	}
	if plotWidth-baseWidth < totals.TotalBars {
		return verticalBarLayout{}, 0, false
	}

	candidate.BarWidth = (plotWidth - baseWidth) / totals.TotalBars
	if candidate.BarWidth < 1 {
		return verticalBarLayout{}, 0, false
	}
	return candidate, baseWidth + candidate.BarWidth*totals.TotalBars, true
}

func verticalBarNormalizeCandidate(
	candidate verticalBarLayout,
	totals verticalBarLayoutTotals,
) verticalBarLayout {
	if totals.GroupGapCount == 0 {
		candidate.GroupGap = 0
	}
	if totals.TotalBarGaps == 0 {
		candidate.BarGap = 0
	}
	return candidate
}

func verticalBarBaseWidth(
	candidate verticalBarLayout,
	totals verticalBarLayoutTotals,
) int {
	return totals.EmptyGroups +
		candidate.GroupGap*totals.GroupGapCount +
		candidate.BarGap*totals.TotalBarGaps
}

func verticalBarLayoutBetter(
	candidate verticalBarLayout,
	used int,
	best verticalBarLayout,
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
	if candidate.GroupGap != best.GroupGap {
		return candidate.GroupGap > best.GroupGap
	}
	return candidate.BarGap > best.BarGap
}

func verticalBarDistributeRemainder(
	layout *verticalBarLayout,
	remaining int,
	totals verticalBarLayoutTotals,
) {
	if totals.GroupGapCount > 0 {
		layout.GroupGap += remaining / totals.GroupGapCount
		remaining %= totals.GroupGapCount
	} else if totals.TotalBarGaps > 0 {
		layout.BarGap += remaining / totals.TotalBarGaps
		remaining %= totals.TotalBarGaps
	}
	layout.LeftPad = remaining / 2
}

func verticalBarGroupWidth(layout verticalBarLayout, count int) int {
	if count <= 0 {
		return 1
	}
	return layout.BarWidth*count + layout.BarGap*max(count-1, 0)
}

func verticalBarGroupBars(
	layout verticalBarLayout,
	offset int,
	count int,
) []DailyRateBarSlot {
	if layout.BarWidth <= 0 || count <= 0 {
		return nil
	}

	slots := make([]DailyRateBarSlot, 0, count)
	start := offset
	for range count {
		end := start + layout.BarWidth
		slots = append(slots, buildDailyRateBarSlot(start, end))
		start = end + layout.BarGap
	}
	return slots
}
