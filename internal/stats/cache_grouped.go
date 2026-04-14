package stats

import (
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

var cacheGroupedRowLabels = []string{
	"Main cache-rd",
	"Sub  cache-rd",
	"Main cache-wr",
	"Sub  cache-wr",
	"Main miss",
	"Sub  miss",
}

func ComputeCacheByVersion(
	sessions []conv.SessionMeta,
	timeRange TimeRange,
	provider conv.Provider,
	versions map[string]bool,
) CacheByVersion {
	grouped := CacheByVersion{
		SegmentRows:   make([]GroupedNamedStat, 0, len(cacheGroupedRowLabels)),
		ReadDuration:  make([]GroupedHistogramBucket, len(cacheDurationLabels)),
		WriteDuration: make([]GroupedHistogramBucket, len(cacheDurationLabels)),
	}
	for i, label := range cacheDurationLabels {
		grouped.ReadDuration[i] = GroupedHistogramBucket{Label: label}
		grouped.WriteDuration[i] = GroupedHistogramBucket{Label: label}
	}
	if len(sessions) == 0 || provider == "" {
		return groupedRowsWithLabels(grouped)
	}

	filtered := filterSessionsForCacheGrouping(sessions, timeRange, provider, versions)
	if len(filtered) == 0 {
		return groupedRowsWithLabels(grouped)
	}

	location := activityLocation(filtered, timeRange)
	start, _, startDayKey, dayCount, ok := resolveCacheBounds(filtered, timeRange, location)
	if !ok {
		return groupedRowsWithLabels(grouped)
	}

	readTotals := make([]int, dayCount)
	writeTotals := make([]int, dayCount)
	promptTotals := make([]int, dayCount)
	activeDays := make([]bool, dayCount)
	readVersions := make([]map[string]int, dayCount)
	writeVersions := make([]map[string]int, dayCount)
	rowVersions := make([]map[string]int, len(cacheGroupedRowLabels))
	readDurationVersions := make([]map[string]int, len(cacheDurationLabels))
	writeDurationVersions := make([]map[string]int, len(cacheDurationLabels))

	for _, session := range filtered {
		versionLabel := NormalizeVersionLabel(session.Version)
		prompt := sessionPromptTokens(session)
		readTokens := session.TotalUsage.CacheReadInputTokens
		writeProxy := cacheWriteProxy(session)
		missTokens := prompt - readTokens - writeProxy

		dayIndex := activityDayKey(
			normalizeActivityTime(session.Timestamp, location),
		) - startDayKey
		recordGroupedCacheDay(
			dayIndex,
			dayCount,
			activeDays,
			promptTotals,
			readTotals,
			writeTotals,
			readVersions,
			writeVersions,
			versionLabel,
			prompt,
			readTokens,
			writeProxy,
		)

		rowBase := 0
		if session.IsSubagent {
			rowBase = 1
		}
		accumulateGroupedCacheRow(rowVersions, rowBase, versionLabel, readTokens)
		accumulateGroupedCacheRow(rowVersions, rowBase+2, versionLabel, writeProxy)
		accumulateGroupedCacheRow(rowVersions, rowBase+4, versionLabel, missTokens)

		durationIndex := durationBucket(session.Duration())
		accumulateGroupedCacheRow(readDurationVersions, durationIndex, versionLabel, readTokens)
		accumulateGroupedCacheRow(writeDurationVersions, durationIndex, versionLabel, writeProxy)
		grouped.ReadDuration[durationIndex].Total += readTokens
		grouped.WriteDuration[durationIndex].Total += writeProxy
	}

	grouped.DailyReadShare = buildGroupedDailyShares(start, dayCount, promptTotals, readTotals, activeDays, readVersions)
	grouped.DailyWriteShare = buildGroupedDailyShares(
		start,
		dayCount,
		promptTotals,
		writeTotals,
		activeDays,
		writeVersions,
	)
	grouped.SegmentRows = buildGroupedCacheRows(rowVersions)
	for i := range grouped.ReadDuration {
		grouped.ReadDuration[i].Versions = sortVersionValues(readDurationVersions[i])
		grouped.WriteDuration[i].Versions = sortVersionValues(writeDurationVersions[i])
	}
	return grouped
}

func filterSessionsForCacheGrouping(
	sessions []conv.SessionMeta,
	timeRange TimeRange,
	provider conv.Provider,
	versions map[string]bool,
) []conv.SessionMeta {
	filtered := make([]conv.SessionMeta, 0, len(sessions))
	for _, session := range sessions {
		if _, ok := matchSessionVersionScope(session, timeRange, provider, versions); !ok {
			continue
		}
		filtered = append(filtered, session)
	}
	return filtered
}

func groupedRowsWithLabels(grouped CacheByVersion) CacheByVersion {
	grouped.SegmentRows = buildGroupedCacheRows(nil)
	return grouped
}

func recordGroupedCacheDay(
	dayIndex int,
	dayCount int,
	activeDays []bool,
	promptTotals []int,
	readTotals []int,
	writeTotals []int,
	readVersions []map[string]int,
	writeVersions []map[string]int,
	version string,
	prompt int,
	readTokens int,
	writeProxy int,
) {
	if dayIndex < 0 || dayIndex >= dayCount {
		return
	}
	activeDays[dayIndex] = true
	promptTotals[dayIndex] += prompt
	readTotals[dayIndex] += readTokens
	writeTotals[dayIndex] += writeProxy
	accumulateGroupedCacheRow(readVersions, dayIndex, version, readTokens)
	accumulateGroupedCacheRow(writeVersions, dayIndex, version, writeProxy)
}

func accumulateGroupedCacheRow(target []map[string]int, index int, version string, value int) {
	if value <= 0 {
		return
	}
	if target[index] == nil {
		target[index] = make(map[string]int)
	}
	target[index][version] += value
}

func buildGroupedDailyShares(
	start time.Time,
	dayCount int,
	promptTotals []int,
	valueTotals []int,
	activeDays []bool,
	versionTotals []map[string]int,
) []GroupedDailyShare {
	daily := make([]GroupedDailyShare, 0, dayCount)
	for i := range dayCount {
		daily = append(daily, GroupedDailyShare{
			Date:        start.AddDate(0, 0, i),
			Prompt:      promptTotals[i],
			Total:       valueTotals[i],
			HasActivity: activeDays[i],
			Versions:    sortVersionValues(versionTotals[i]),
		})
	}
	return daily
}

func buildGroupedCacheRows(rowVersions []map[string]int) []GroupedNamedStat {
	rows := make([]GroupedNamedStat, 0, len(cacheGroupedRowLabels))
	for i, label := range cacheGroupedRowLabels {
		versions := map[string]int(nil)
		if i < len(rowVersions) {
			versions = rowVersions[i]
		}
		total := 0
		for _, value := range versions {
			total += value
		}
		rows = append(rows, groupedNamedStatFromCounts(label, total, versions))
	}
	return rows
}
