package stats

import (
	"slices"

	conv "github.com/rkuska/carn/internal/conversation"
)

func ComputeToolsBySplit(
	sessions []conv.SessionMeta,
	timeRange TimeRange,
	dim SplitDimension,
	allowed map[string]bool,
) ToolsBySplit {
	grouped := ToolsBySplit{
		CallsPerSession: make([]SplitHistogramBucket, 0, 5),
	}
	bucketSplitCounts := make([]map[string]int, 5)
	for i, label := range []string{"0-20", "21-50", "51-100", "101-200", "201+"} {
		grouped.CallsPerSession = append(grouped.CallsPerSession, SplitHistogramBucket{Label: label})
		bucketSplitCounts[i] = make(map[string]int)
	}
	if len(sessions) == 0 || !dim.IsActive() {
		return grouped
	}

	toolTotals := make(map[string]int)
	toolSplits := make(map[string]map[string]int)
	errorCounts := make(map[string]int)
	errorSplits := make(map[string]map[string]int)
	rejectCounts := make(map[string]int)
	rejectSplits := make(map[string]map[string]int)

	for _, session := range sessions {
		key, ok := matchSessionSplitScope(session, timeRange, dim, allowed)
		if !ok {
			continue
		}

		sessionCalls, _ := accumulateToolCounts(toolTotals, session.ToolCounts, session.ActionCounts)
		bucket := toolCallsBucket(sessionCalls)
		grouped.CallsPerSession[bucket].Total++
		bucketSplitCounts[bucket][key]++

		accumulateSplitCounts(toolSplits, key, session.ToolCounts)
		accumulateSplitCounts(errorSplits, key, session.ToolErrorCounts)
		accumulateSplitCounts(rejectSplits, key, session.ToolRejectCounts)
		accumulateNamedCounts(errorCounts, session.ToolErrorCounts)
		accumulateNamedCounts(rejectCounts, session.ToolRejectCounts)
	}

	for i := range grouped.CallsPerSession {
		grouped.CallsPerSession[i].Splits = sortSplitValues(bucketSplitCounts[i])
	}
	grouped.TopTools = buildSplitToolStats(toolTotals, toolSplits)
	grouped.ToolErrorRates = buildSplitToolRates(toolTotals, errorCounts, errorSplits, minToolErrorCount)
	grouped.ToolRejectRates = buildSplitToolRates(toolTotals, rejectCounts, rejectSplits, 1)
	return grouped
}

func accumulateSplitCounts(target map[string]map[string]int, key string, counts map[string]int) {
	for name, count := range counts {
		bySplit := target[name]
		if bySplit == nil {
			bySplit = make(map[string]int)
			target[name] = bySplit
		}
		bySplit[key] += count
	}
}

func buildSplitToolStats(
	totals map[string]int,
	splitCounts map[string]map[string]int,
) []SplitNamedStat {
	items := make([]SplitNamedStat, 0, len(totals))
	for name, total := range totals {
		items = append(items, splitNamedStatFromCounts(name, total, splitCounts[name]))
	}
	slices.SortFunc(items, func(left, right SplitNamedStat) int {
		switch {
		case left.Total != right.Total:
			return right.Total - left.Total
		case left.Name < right.Name:
			return -1
		case left.Name > right.Name:
			return 1
		default:
			return 0
		}
	})
	return items
}

func buildSplitToolRates(
	totalCounts map[string]int,
	countMap map[string]int,
	splitCounts map[string]map[string]int,
	minCount int,
) []SplitRateStat {
	rates := make([]SplitRateStat, 0, len(countMap))
	for name, count := range countMap {
		total := totalCounts[name]
		if total < minToolRateCalls || count < minCount {
			continue
		}
		rates = append(rates, SplitRateStat{
			Name:   name,
			Count:  count,
			Total:  total,
			Rate:   float64(count) / float64(total) * 100,
			Splits: sortSplitValues(splitCounts[name]),
		})
	}
	slices.SortFunc(rates, func(left, right SplitRateStat) int {
		switch {
		case left.Rate != right.Rate:
			if left.Rate > right.Rate {
				return -1
			}
			return 1
		case left.Count != right.Count:
			return right.Count - left.Count
		case left.Name < right.Name:
			return -1
		case left.Name > right.Name:
			return 1
		default:
			return 0
		}
	})
	return rates
}
