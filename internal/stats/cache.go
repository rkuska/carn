package stats

import (
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

var cacheDurationLabels = [6]string{"<5m", "5-15", "15-30", "30-60", "1-2h", "2h+"}

func ComputeCache(sessions []conv.SessionMeta, timeRange TimeRange) Cache {
	if len(sessions) == 0 {
		return Cache{}
	}

	location := activityLocation(sessions, timeRange)
	cache := Cache{
		DailyCacheRead:  make([]DailyCount, 0),
		DailyCacheWrite: make([]DailyCount, 0),
	}

	readByDay := make(map[time.Time]int)
	writeByDay := make(map[time.Time]int)

	type durationAccum struct {
		hitRateSum float64
		missSum    int
		count      int
	}
	buckets := [6]durationAccum{}

	for _, session := range sessions {
		accumulateCacheSession(&cache, session)

		day := startOfDayInLocation(
			normalizeActivityTime(session.Timestamp, location),
			location,
		)
		readByDay[day] += session.TotalUsage.CacheReadInputTokens
		writeByDay[day] += session.TotalUsage.CacheCreationInputTokens

		prompt := sessionPromptTokens(session)
		var hitRate float64
		if prompt > 0 {
			hitRate = float64(session.TotalUsage.CacheReadInputTokens) / float64(prompt)
		}

		idx := durationBucket(session.Duration())
		buckets[idx].hitRateSum += hitRate
		buckets[idx].missSum += session.TotalUsage.InputTokens
		buckets[idx].count++
	}

	finalizeCacheRates(&cache)

	start, end := cacheDayBounds(sessions, timeRange, location)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		cache.DailyCacheRead = append(cache.DailyCacheRead, DailyCount{
			Date: day, Count: readByDay[day],
		})
		cache.DailyCacheWrite = append(cache.DailyCacheWrite, DailyCount{
			Date: day, Count: writeByDay[day],
		})
	}

	cache.DurationBuckets = make([]CacheDurationBucket, 6)
	for i, label := range cacheDurationLabels {
		b := buckets[i]
		bucket := CacheDurationBucket{Label: label, Sessions: b.count}
		if b.count > 0 {
			bucket.HitRate = b.hitRateSum / float64(b.count)
			bucket.MissTokens = b.missSum / b.count
		}
		cache.DurationBuckets[i] = bucket
	}

	return cache
}

func accumulateCacheSession(cache *Cache, session conv.SessionMeta) {
	usage := session.TotalUsage
	prompt := sessionPromptTokens(session)

	cache.TotalCacheRead += usage.CacheReadInputTokens
	cache.TotalCacheWrite += usage.CacheCreationInputTokens
	cache.TotalPrompt += prompt

	seg := &cache.Main
	if session.IsSubagent {
		seg = &cache.Subagent
	}
	seg.SessionCount++
	seg.CacheRead += usage.CacheReadInputTokens
	seg.CacheWrite += usage.CacheCreationInputTokens
	seg.Prompt += prompt
	seg.MissTokens += usage.InputTokens
}

func finalizeCacheRates(cache *Cache) {
	if cache.TotalPrompt > 0 {
		prompt := float64(cache.TotalPrompt)
		cache.HitRate = float64(cache.TotalCacheRead) / prompt
		cache.WriteRate = float64(cache.TotalCacheWrite) / prompt
		cache.MissRate = float64(cache.TotalPrompt-cache.TotalCacheRead-cache.TotalCacheWrite) / prompt
	}
	if cache.TotalCacheWrite > 0 {
		cache.ReuseRatio = float64(cache.TotalCacheRead) / float64(cache.TotalCacheWrite)
	}

	finalizeSegmentRates(&cache.Main)
	finalizeSegmentRates(&cache.Subagent)
}

func finalizeSegmentRates(seg *CacheSegment) {
	if seg.Prompt > 0 {
		seg.HitRate = float64(seg.CacheRead) / float64(seg.Prompt)
	}
}

func sessionPromptTokens(session conv.SessionMeta) int {
	u := session.TotalUsage
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
}

func cacheDayBounds(
	sessions []conv.SessionMeta,
	timeRange TimeRange,
	location *time.Location,
) (time.Time, time.Time) {
	start, end, ok := resolveActivityBounds(sessions, timeRange, location)
	if !ok {
		return time.Time{}, time.Time{}
	}
	return start, end
}
