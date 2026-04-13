package stats

import (
	"math"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

var cacheDurationLabels = [6]string{"<5m", "5-15", "15-30", "30-60", "1-2h", "2h+"}

type cacheDurationAccum struct {
	hitRateSum float64
	readSum    int
	writeSum   int
	count      int
}

func ComputeCache(sessions []conv.SessionMeta, timeRange TimeRange) Cache {
	if len(sessions) == 0 {
		return Cache{}
	}

	location := activityLocation(sessions, timeRange)
	start, _, startDayKey, dayCount, ok := resolveCacheBounds(sessions, timeRange, location)
	if !ok {
		return Cache{}
	}

	cache := Cache{
		DailyHitRate:    make([]DailyRate, dayCount),
		DailyReuseRatio: make([]DailyRate, dayCount),
		DurationBuckets: make([]CacheDurationBucket, len(cacheDurationLabels)),
	}
	readByDay := make([]int, dayCount)
	writeByDay := make([]int, dayCount)
	promptByDay := make([]int, dayCount)
	activeByDay := make([]bool, dayCount)
	buckets := [6]cacheDurationAccum{}

	for _, session := range sessions {
		prompt := sessionPromptTokens(session)
		readTokens := session.TotalUsage.CacheReadInputTokens
		writeProxy := cacheWriteProxy(session)
		accumulateCacheSession(&cache, session, prompt, readTokens, writeProxy)

		dayIndex := activityDayKey(
			normalizeActivityTime(session.Timestamp, location),
		) - startDayKey
		recordCacheDay(dayIndex, dayCount, activeByDay, readByDay, writeByDay, promptByDay, readTokens, writeProxy, prompt)
		accumulateDurationBucket(&buckets[durationBucket(session.Duration())], readTokens, writeProxy, prompt)
	}

	finalizeCacheRates(&cache)
	finalizeCacheDays(&cache, start, readByDay, writeByDay, promptByDay, activeByDay)
	finalizeCacheDurationBuckets(&cache, buckets)

	return cache
}

func accumulateCacheSession(
	cache *Cache,
	session conv.SessionMeta,
	prompt int,
	readTokens int,
	writeProxy int,
) {
	cache.TotalCacheRead += readTokens
	cache.TotalCacheWrite += writeProxy
	cache.TotalPrompt += prompt

	seg := &cache.Main
	if session.IsSubagent {
		seg = &cache.Subagent
	}
	seg.SessionCount++
	seg.CacheRead += readTokens
	seg.CacheWrite += writeProxy
	seg.Prompt += prompt
	seg.MissTokens += prompt - readTokens - writeProxy
}

func finalizeCacheRates(cache *Cache) {
	if cache.TotalPrompt > 0 {
		prompt := float64(cache.TotalPrompt)
		cache.HitRate = float64(cache.TotalCacheRead) / prompt
		cache.WriteRate = float64(cache.TotalCacheWrite) / prompt
		cache.MissRate = float64(cache.TotalPrompt-cache.TotalCacheRead-cache.TotalCacheWrite) / prompt
	}
	cache.ReuseRatio = cacheReuseRatio(cache.TotalCacheRead, cache.TotalCacheWrite)

	finalizeSegmentRates(&cache.Main)
	finalizeSegmentRates(&cache.Subagent)
}

func finalizeSegmentRates(seg *CacheSegment) {
	if seg.Prompt > 0 {
		seg.HitRate = float64(seg.CacheRead) / float64(seg.Prompt)
	}
}

func recordCacheDay(
	dayIndex int,
	dayCount int,
	activeByDay []bool,
	readByDay []int,
	writeByDay []int,
	promptByDay []int,
	readTokens int,
	writeProxy int,
	prompt int,
) {
	if dayIndex < 0 || dayIndex >= dayCount {
		return
	}
	activeByDay[dayIndex] = true
	readByDay[dayIndex] += readTokens
	writeByDay[dayIndex] += writeProxy
	promptByDay[dayIndex] += prompt
}

func accumulateDurationBucket(bucket *cacheDurationAccum, readTokens, writeProxy, prompt int) {
	var hitRate float64
	if prompt > 0 {
		hitRate = float64(readTokens) / float64(prompt)
	}
	bucket.hitRateSum += hitRate
	bucket.readSum += readTokens
	bucket.writeSum += writeProxy
	bucket.count++
}

func finalizeCacheDays(
	cache *Cache,
	start time.Time,
	readByDay []int,
	writeByDay []int,
	promptByDay []int,
	activeByDay []bool,
) {
	day := start
	for i := range len(readByDay) {
		var hitRate float64
		if p := promptByDay[i]; p > 0 {
			hitRate = float64(readByDay[i]) / float64(p)
		}
		cache.DailyHitRate[i] = DailyRate{
			Date: day, Rate: hitRate, HasActivity: activeByDay[i],
		}
		cache.DailyReuseRatio[i] = DailyRate{
			Date: day, Rate: cacheReuseRatio(readByDay[i], writeByDay[i]), HasActivity: activeByDay[i],
		}
		day = day.AddDate(0, 0, 1)
	}
}

func finalizeCacheDurationBuckets(cache *Cache, buckets [6]cacheDurationAccum) {
	for i, label := range cacheDurationLabels {
		b := buckets[i]
		bucket := CacheDurationBucket{Label: label, Sessions: b.count}
		if b.count > 0 {
			bucket.HitRate = b.hitRateSum / float64(b.count)
			bucket.ReuseRatio = cacheReuseRatio(b.readSum, b.writeSum)
		}
		cache.DurationBuckets[i] = bucket
	}
}

// cacheWriteProxy returns the best available cache-write token count.
// Source-side normalization already fills CacheCreationInputTokens for current
// providers. Keep the InputTokens fallback only for legacy Codex/OpenAI
// sessions that predate that normalization.
func cacheWriteProxy(session conv.SessionMeta) int {
	usage := session.TotalUsage
	if usage.CacheCreationInputTokens > 0 {
		return usage.CacheCreationInputTokens
	}
	if session.Provider == conv.ProviderCodex && usage.CacheReadInputTokens > 0 {
		return usage.InputTokens
	}
	return 0
}

func cacheReuseRatio(read, write int) float64 {
	switch {
	case write > 0:
		return float64(read) / float64(write)
	case read > 0:
		return math.Inf(1)
	default:
		return 0
	}
}

func sessionPromptTokens(session conv.SessionMeta) int {
	u := session.TotalUsage
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
}

func resolveCacheBounds(
	sessions []conv.SessionMeta,
	timeRange TimeRange,
	location *time.Location,
) (time.Time, time.Time, int, int, bool) {
	start, end, ok := resolveActivityBounds(sessions, timeRange, location)
	if !ok {
		return time.Time{}, time.Time{}, 0, 0, false
	}
	startDayKey := activityDayKey(start)
	return start, end, startDayKey, activityDayKey(end) - startDayKey + 1, true
}

func activityDayKey(ts time.Time) int {
	year, month, day := ts.Date()
	return civilDayNumber(year, month, day)
}

func civilDayNumber(year int, month time.Month, day int) int {
	y := year
	m := int(month)
	if m <= 2 {
		y--
		m += 12
	}
	era := floorDiv(y, 400)
	yoe := y - era*400
	doy := (153*(m-3)+2)/5 + day - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe
}

func floorDiv(n, d int) int {
	if n >= 0 {
		return n / d
	}
	return -((-n + d - 1) / d)
}
