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
		DailyHitRate:    make([]DailyRate, 0),
		DailyReuseRatio: make([]DailyRate, 0),
	}

	readByDay := make(map[time.Time]int)
	writeByDay := make(map[time.Time]int)
	promptByDay := make(map[time.Time]int)

	type durationAccum struct {
		hitRateSum float64
		readSum    int
		writeSum   int
		count      int
	}
	buckets := [6]durationAccum{}

	for _, session := range sessions {
		accumulateCacheSession(&cache, session)

		writeProxy := cacheWriteProxy(session.TotalUsage)
		day := startOfDayInLocation(
			normalizeActivityTime(session.Timestamp, location),
			location,
		)
		readByDay[day] += session.TotalUsage.CacheReadInputTokens
		writeByDay[day] += writeProxy
		promptByDay[day] += sessionPromptTokens(session)

		prompt := sessionPromptTokens(session)
		var hitRate float64
		if prompt > 0 {
			hitRate = float64(session.TotalUsage.CacheReadInputTokens) / float64(prompt)
		}

		idx := durationBucket(session.Duration())
		buckets[idx].hitRateSum += hitRate
		buckets[idx].readSum += session.TotalUsage.CacheReadInputTokens
		buckets[idx].writeSum += writeProxy
		buckets[idx].count++
	}

	finalizeCacheRates(&cache)

	start, end := cacheDayBounds(sessions, timeRange, location)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		var hitRate float64
		if p := promptByDay[day]; p > 0 {
			hitRate = float64(readByDay[day]) / float64(p)
		}
		cache.DailyHitRate = append(cache.DailyHitRate, DailyRate{
			Date: day, Rate: hitRate,
		})

		var reuseRatio float64
		if w := writeByDay[day]; w > 0 {
			reuseRatio = float64(readByDay[day]) / float64(w)
		}
		cache.DailyReuseRatio = append(cache.DailyReuseRatio, DailyRate{
			Date: day, Rate: reuseRatio,
		})
	}

	cache.DurationBuckets = make([]CacheDurationBucket, 6)
	for i, label := range cacheDurationLabels {
		b := buckets[i]
		bucket := CacheDurationBucket{Label: label, Sessions: b.count}
		if b.count > 0 {
			bucket.HitRate = b.hitRateSum / float64(b.count)
			if b.writeSum > 0 {
				bucket.ReuseRatio = float64(b.readSum) / float64(b.writeSum)
			}
		}
		cache.DurationBuckets[i] = bucket
	}

	return cache
}

func accumulateCacheSession(cache *Cache, session conv.SessionMeta) {
	usage := session.TotalUsage
	prompt := sessionPromptTokens(session)
	writeProxy := cacheWriteProxy(usage)

	cache.TotalCacheRead += usage.CacheReadInputTokens
	cache.TotalCacheWrite += writeProxy
	cache.TotalPrompt += prompt

	seg := &cache.Main
	if session.IsSubagent {
		seg = &cache.Subagent
	}
	seg.SessionCount++
	seg.CacheRead += usage.CacheReadInputTokens
	seg.CacheWrite += writeProxy
	seg.Prompt += prompt
	seg.MissTokens += prompt - usage.CacheReadInputTokens - writeProxy
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

// cacheWriteProxy returns the best available cache-write token count.
// Providers that report explicit writes (Claude) return CacheCreationInputTokens.
// Providers that only report reads (Codex/OpenAI) fall back to InputTokens
// (uncached tokens) as an approximation of cache writes.
func cacheWriteProxy(usage conv.TokenUsage) int {
	if usage.CacheCreationInputTokens > 0 {
		return usage.CacheCreationInputTokens
	}
	if usage.CacheReadInputTokens > 0 {
		return usage.InputTokens
	}
	return 0
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
