package stats

import (
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

var cacheSplitRowLabels = []string{
	"Main cache-rd",
	"Sub  cache-rd",
	"Main cache-wr",
	"Sub  cache-wr",
	"Main miss",
	"Sub  miss",
}

func ComputeCacheBySplit(
	sessions []conv.SessionMeta,
	timeRange TimeRange,
	dim SplitDimension,
	allowed map[string]bool,
) CacheBySplit {
	grouped := CacheBySplit{
		SegmentRows:   make([]SplitNamedStat, 0, len(cacheSplitRowLabels)),
		ReadDuration:  make([]SplitHistogramBucket, len(cacheDurationLabels)),
		WriteDuration: make([]SplitHistogramBucket, len(cacheDurationLabels)),
	}
	for i, label := range cacheDurationLabels {
		grouped.ReadDuration[i] = SplitHistogramBucket{Label: label}
		grouped.WriteDuration[i] = SplitHistogramBucket{Label: label}
	}
	if len(sessions) == 0 || !dim.IsActive() {
		return splitRowsWithLabels(grouped)
	}

	filtered := filterSessionsForCacheSplit(sessions, timeRange, dim, allowed)
	if len(filtered) == 0 {
		return splitRowsWithLabels(grouped)
	}

	location := activityLocation(filtered, timeRange)
	start, _, startDayKey, dayCount, ok := resolveCacheBounds(filtered, timeRange, location)
	if !ok {
		return splitRowsWithLabels(grouped)
	}

	readTotals := make([]int, dayCount)
	writeTotals := make([]int, dayCount)
	promptTotals := make([]int, dayCount)
	activeDays := make([]bool, dayCount)
	readSplits := make([]map[string]int, dayCount)
	writeSplits := make([]map[string]int, dayCount)
	rowSplits := make([]map[string]int, len(cacheSplitRowLabels))
	readDurationSplits := make([]map[string]int, len(cacheDurationLabels))
	writeDurationSplits := make([]map[string]int, len(cacheDurationLabels))

	for _, session := range filtered {
		key := dim.SessionKey(session)
		prompt := sessionPromptTokens(session)
		readTokens := session.TotalUsage.CacheReadInputTokens
		writeProxy := cacheWriteProxy(session)
		missTokens := prompt - readTokens - writeProxy

		dayIndex := activityDayKey(
			normalizeActivityTime(session.Timestamp, location),
		) - startDayKey
		recordSplitCacheDay(
			dayIndex,
			dayCount,
			activeDays,
			promptTotals,
			readTotals,
			writeTotals,
			readSplits,
			writeSplits,
			key,
			prompt,
			readTokens,
			writeProxy,
		)

		rowBase := 0
		if session.IsSubagent {
			rowBase = 1
		}
		accumulateSplitCacheRow(rowSplits, rowBase, key, readTokens)
		accumulateSplitCacheRow(rowSplits, rowBase+2, key, writeProxy)
		accumulateSplitCacheRow(rowSplits, rowBase+4, key, missTokens)

		durationIndex := durationBucket(session.Duration())
		accumulateSplitCacheRow(readDurationSplits, durationIndex, key, readTokens)
		accumulateSplitCacheRow(writeDurationSplits, durationIndex, key, writeProxy)
		grouped.ReadDuration[durationIndex].Total += readTokens
		grouped.WriteDuration[durationIndex].Total += writeProxy
	}

	grouped.DailyReadShare = buildSplitDailyShares(start, dayCount, promptTotals, readTotals, activeDays, readSplits)
	grouped.DailyWriteShare = buildSplitDailyShares(
		start,
		dayCount,
		promptTotals,
		writeTotals,
		activeDays,
		writeSplits,
	)
	grouped.SegmentRows = buildSplitCacheRows(rowSplits)
	for i := range grouped.ReadDuration {
		grouped.ReadDuration[i].Splits = sortSplitValues(readDurationSplits[i])
		grouped.WriteDuration[i].Splits = sortSplitValues(writeDurationSplits[i])
	}
	return grouped
}

func filterSessionsForCacheSplit(
	sessions []conv.SessionMeta,
	timeRange TimeRange,
	dim SplitDimension,
	allowed map[string]bool,
) []conv.SessionMeta {
	filtered := make([]conv.SessionMeta, 0, len(sessions))
	for _, session := range sessions {
		if _, ok := matchSessionSplitScope(session, timeRange, dim, allowed); !ok {
			continue
		}
		filtered = append(filtered, session)
	}
	return filtered
}

func splitRowsWithLabels(grouped CacheBySplit) CacheBySplit {
	grouped.SegmentRows = buildSplitCacheRows(nil)
	return grouped
}

func recordSplitCacheDay(
	dayIndex int,
	dayCount int,
	activeDays []bool,
	promptTotals []int,
	readTotals []int,
	writeTotals []int,
	readSplits []map[string]int,
	writeSplits []map[string]int,
	key string,
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
	accumulateSplitCacheRow(readSplits, dayIndex, key, readTokens)
	accumulateSplitCacheRow(writeSplits, dayIndex, key, writeProxy)
}

func accumulateSplitCacheRow(target []map[string]int, index int, key string, value int) {
	if value <= 0 {
		return
	}
	if target[index] == nil {
		target[index] = make(map[string]int)
	}
	target[index][key] += value
}

func buildSplitDailyShares(
	start time.Time,
	dayCount int,
	promptTotals []int,
	valueTotals []int,
	activeDays []bool,
	splitTotals []map[string]int,
) []SplitDailyShare {
	daily := make([]SplitDailyShare, 0, dayCount)
	for i := range dayCount {
		daily = append(daily, SplitDailyShare{
			Date:        start.AddDate(0, 0, i),
			Prompt:      promptTotals[i],
			Total:       valueTotals[i],
			HasActivity: activeDays[i],
			Splits:      sortSplitValues(splitTotals[i]),
		})
	}
	return daily
}

func buildSplitCacheRows(rowSplits []map[string]int) []SplitNamedStat {
	rows := make([]SplitNamedStat, 0, len(cacheSplitRowLabels))
	for i, label := range cacheSplitRowLabels {
		splits := map[string]int(nil)
		if i < len(rowSplits) {
			splits = rowSplits[i]
		}
		total := 0
		for _, value := range splits {
			total += value
		}
		rows = append(rows, splitNamedStatFromCounts(label, total, splits))
	}
	return rows
}
