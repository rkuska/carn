package stats

import (
	"slices"

	conv "github.com/rkuska/carn/internal/conversation"
)

func ComputeToolsByVersion(
	sessions []conv.SessionMeta,
	timeRange TimeRange,
	provider conv.Provider,
	versions map[string]bool,
) ToolsByVersion {
	grouped := ToolsByVersion{
		CallsPerSession: make([]GroupedHistogramBucket, 0, 5),
	}
	bucketVersionCounts := make([]map[string]int, 5)
	for i, label := range []string{"0-20", "21-50", "51-100", "101-200", "201+"} {
		grouped.CallsPerSession = append(grouped.CallsPerSession, GroupedHistogramBucket{Label: label})
		bucketVersionCounts[i] = make(map[string]int)
	}
	if len(sessions) == 0 || provider == "" {
		return grouped
	}

	toolTotals := make(map[string]int)
	toolVersions := make(map[string]map[string]int)
	errorCounts := make(map[string]int)
	errorVersions := make(map[string]map[string]int)
	rejectCounts := make(map[string]int)
	rejectVersions := make(map[string]map[string]int)

	for _, session := range sessions {
		versionLabel, ok := matchSessionVersionScope(session, timeRange, provider, versions)
		if !ok {
			continue
		}

		sessionCalls, _ := accumulateToolCounts(toolTotals, session.ToolCounts, session.ActionCounts)
		bucket := toolCallsBucket(sessionCalls)
		grouped.CallsPerSession[bucket].Total++
		bucketVersionCounts[bucket][versionLabel]++

		accumulateVersionCounts(toolVersions, versionLabel, session.ToolCounts)
		accumulateVersionCounts(errorVersions, versionLabel, session.ToolErrorCounts)
		accumulateVersionCounts(rejectVersions, versionLabel, session.ToolRejectCounts)
		accumulateNamedCounts(errorCounts, session.ToolErrorCounts)
		accumulateNamedCounts(rejectCounts, session.ToolRejectCounts)
	}

	for i := range grouped.CallsPerSession {
		grouped.CallsPerSession[i].Versions = sortVersionValues(bucketVersionCounts[i])
	}
	grouped.TopTools = buildGroupedToolStats(toolTotals, toolVersions)
	grouped.ToolErrorRates = buildGroupedToolRates(toolTotals, errorCounts, errorVersions, minToolErrorCount)
	grouped.ToolRejectRates = buildGroupedToolRates(toolTotals, rejectCounts, rejectVersions, 1)
	return grouped
}

func accumulateVersionCounts(target map[string]map[string]int, version string, counts map[string]int) {
	for name, count := range counts {
		byVersion := target[name]
		if byVersion == nil {
			byVersion = make(map[string]int)
			target[name] = byVersion
		}
		byVersion[version] += count
	}
}

func buildGroupedToolStats(
	totals map[string]int,
	versionCounts map[string]map[string]int,
) []GroupedNamedStat {
	items := make([]GroupedNamedStat, 0, len(totals))
	for name, total := range totals {
		items = append(items, groupedNamedStatFromCounts(name, total, versionCounts[name]))
	}
	slices.SortFunc(items, func(left, right GroupedNamedStat) int {
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

func buildGroupedToolRates(
	totalCounts map[string]int,
	countMap map[string]int,
	versionCounts map[string]map[string]int,
	minCount int,
) []GroupedRateStat {
	rates := make([]GroupedRateStat, 0, len(countMap))
	for name, count := range countMap {
		total := totalCounts[name]
		if total < minToolRateCalls || count < minCount {
			continue
		}
		rates = append(rates, GroupedRateStat{
			Name:     name,
			Count:    count,
			Total:    total,
			Rate:     float64(count) / float64(total) * 100,
			Versions: sortVersionValues(versionCounts[name]),
		})
	}
	slices.SortFunc(rates, func(left, right GroupedRateStat) int {
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
