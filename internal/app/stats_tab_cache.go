package app

import (
	"fmt"
	"image/color"
	"math"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderCacheTab(width, height int) string {
	cache := m.snapshot.Cache
	chips := renderSummaryChips(m.cacheSummaryChips(cache), width)

	chartTitle, counts, chartColor := m.cacheDailySeries()
	chartHeight := 12
	if height < 18 {
		chartHeight = max(height-6, 6)
	}

	topPair := renderStatsLanePair(
		width, 30,
		chartTitle, m.cacheLaneCursor == 0,
		func(bodyWidth int) string {
			return renderDailyActivityChartBody(counts, max(bodyWidth, 10), chartHeight, chartColor)
		},
		"Main vs Subagent", m.cacheLaneCursor == 1,
		func(bodyWidth int) string {
			return renderHorizontalBarsBody(cacheSegmentBars(cache), bodyWidth, colorChartToken)
		},
	)

	missBuckets := cacheMissBuckets(cache.DurationBuckets)
	hitBuckets := cacheHitRateBuckets(cache.DurationBuckets)
	histHeight := 8

	bottomPair := renderStatsLanePair(
		width, 30,
		"Cache Miss Cost by Duration", m.cacheLaneCursor == 2,
		func(bodyWidth int) string {
			return renderVerticalHistogramBody(missBuckets, bodyWidth, histHeight, colorChartError)
		},
		"Cache Hit Rate by Duration", m.cacheLaneCursor == 3,
		func(bodyWidth int) string {
			return renderVerticalHistogramBody(hitBuckets, bodyWidth, histHeight, colorChartBar)
		},
	)

	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", chips, topPair, bottomPair, m.renderActiveMetricDetail(width))
}

func (m statsModel) cacheSummaryChips(cache statspkg.Cache) []chip {
	chips := []chip{
		{Label: "hit rate", Value: formatRate(cache.HitRate)},
		{Label: "miss rate", Value: formatRate(cache.MissRate)},
		{Label: "reuse", Value: formatReuse(cache.ReuseRatio)},
		{Label: "cache-rd", Value: statspkg.FormatNumber(cache.TotalCacheRead)},
		{Label: "cache-wr", Value: statspkg.FormatNumber(cache.TotalCacheWrite)},
		{Label: "miss cost", Value: statspkg.FormatNumber(cache.TotalPrompt - cache.TotalCacheRead - cache.TotalCacheWrite)},
	}
	if cache.Main.SessionCount > 0 {
		chips = append(chips, chip{Label: "main hit", Value: formatRate(cache.Main.HitRate)})
	}
	if cache.Subagent.SessionCount > 0 {
		chips = append(chips, chip{Label: "sub hit", Value: formatRate(cache.Subagent.HitRate)})
	}
	return chips
}

func (m statsModel) cacheDailySeries() (string, []statspkg.DailyCount, color.Color) {
	switch m.cacheMetric {
	case cacheMetricWrite:
		return "Daily Cache Write", m.snapshot.Cache.DailyCacheWrite, colorChartBar
	case cacheMetricRead:
		return "Daily Cache Read", m.snapshot.Cache.DailyCacheRead, colorChartToken
	}
	return "Daily Cache Read", m.snapshot.Cache.DailyCacheRead, colorChartToken
}

func cacheSegmentBars(cache statspkg.Cache) []barItem {
	bars := make([]barItem, 0, 6)
	if cache.Main.SessionCount > 0 || cache.Subagent.SessionCount > 0 {
		bars = append(bars,
			barItem{Label: "Main cache-rd", Value: cache.Main.CacheRead},
			barItem{Label: "Sub  cache-rd", Value: cache.Subagent.CacheRead},
			barItem{Label: "Main cache-wr", Value: cache.Main.CacheWrite},
			barItem{Label: "Sub  cache-wr", Value: cache.Subagent.CacheWrite},
			barItem{Label: "Main miss", Value: cache.Main.MissTokens},
			barItem{Label: "Sub  miss", Value: cache.Subagent.MissTokens},
		)
	}
	return bars
}

func cacheMissBuckets(durations []statspkg.CacheDurationBucket) []histBucket {
	buckets := make([]histBucket, 0, len(durations))
	for _, d := range durations {
		buckets = append(buckets, histBucket{Label: d.Label, Count: d.MissTokens})
	}
	return buckets
}

func cacheHitRateBuckets(durations []statspkg.CacheDurationBucket) []histBucket {
	buckets := make([]histBucket, 0, len(durations))
	for _, d := range durations {
		buckets = append(buckets, histBucket{
			Label: d.Label,
			Count: int(math.Round(d.HitRate * 100)),
		})
	}
	return buckets
}

func (m statsModel) renderCacheMetricDetail(width int) string {
	lane, _, ok := m.selectedStatsLane()
	if !ok {
		return renderStatsMetricDetail("Cache", width, nil, noDataLabel)
	}

	cache := m.snapshot.Cache
	switch lane.id { //nolint:exhaustive // only cache lanes handled here
	case statsLaneCacheDaily:
		return m.renderCacheDailyMetricDetail(cache, lane.title, width)
	case statsLaneCacheSegment:
		return renderCacheSegmentMetricDetail(cache, width)
	case statsLaneCacheMiss:
		return renderCacheMissMetricDetail(cache, width)
	case statsLaneCacheHitDur:
		return renderCacheHitDurMetricDetail(cache, width)
	default:
		return renderStatsMetricDetail("Cache", width, nil, noDataLabel)
	}
}

func (m statsModel) renderCacheDailyMetricDetail(cache statspkg.Cache, title string, width int) string {
	_, counts, _ := m.cacheDailySeries()
	peakDay, peakCount := peakDailyCount(counts)
	return renderStatsMetricDetail(title, width, []chip{
		{Label: "peak day", Value: peakDay},
		{Label: "peak tokens", Value: statspkg.FormatNumber(peakCount)},
		{Label: "total read", Value: statspkg.FormatNumber(cache.TotalCacheRead)},
		{Label: "total write", Value: statspkg.FormatNumber(cache.TotalCacheWrite)},
	},
		metricDetailLine("Question", "How does cache token volume change over time?"),
		metricDetailLine("Reading", "The chart shows daily token volume. Press m to toggle read/write."),
	)
}

func renderCacheSegmentMetricDetail(cache statspkg.Cache, width int) string {
	return renderStatsMetricDetail("Main vs Subagent", width, []chip{
		{Label: "main sessions", Value: statspkg.FormatNumber(cache.Main.SessionCount)},
		{Label: "main hit rate", Value: formatRate(cache.Main.HitRate)},
		{Label: "sub sessions", Value: statspkg.FormatNumber(cache.Subagent.SessionCount)},
		{Label: "sub hit rate", Value: formatRate(cache.Subagent.HitRate)},
	},
		metricDetailLine("Question", "Does the main thread cache better than subagents?"),
		metricDetailLine(
			"Reading",
			"Main sessions use 1h cache TTL, subagents use 5min. Higher miss bars indicate more uncached prompt tokens.",
		),
	)
}

func renderCacheMissMetricDetail(cache statspkg.Cache, width int) string {
	missMetric := func(b statspkg.CacheDurationBucket) int { return b.MissTokens }
	worst := worstCacheDurationBucket(cache.DurationBuckets, missMetric)
	totalMiss := cache.TotalPrompt - cache.TotalCacheRead - cache.TotalCacheWrite
	return renderStatsMetricDetail("Cache Miss Cost by Duration", width, []chip{
		{Label: "highest miss bucket", Value: worst},
		{Label: "total miss tokens", Value: statspkg.FormatNumber(totalMiss)},
	},
		metricDetailLine("Question", "Which session durations have the highest cache miss cost?"),
		metricDetailLine(
			"Reading",
			"Taller bars mean more uncached prompt tokens per session. Long sessions may lose cache due to TTL expiry.",
		),
	)
}

func renderCacheHitDurMetricDetail(cache statspkg.Cache, width int) string {
	best := bestCacheDurationBucket(cache.DurationBuckets)
	return renderStatsMetricDetail("Cache Hit Rate by Duration", width, []chip{
		{Label: "best bucket", Value: best},
		{Label: "overall hit rate", Value: formatRate(cache.HitRate)},
	},
		metricDetailLine("Question", "Do longer sessions maintain or lose cache efficiency?"),
		metricDetailLine(
			"Reading",
			"Bar height is average cache hit rate (%). Decreasing bars across duration buckets suggest cache staleness.",
		),
	)
}

func worstCacheDurationBucket(
	buckets []statspkg.CacheDurationBucket,
	metric func(statspkg.CacheDurationBucket) int,
) string {
	if len(buckets) == 0 {
		return noDataLabel
	}
	worst := buckets[0]
	for _, b := range buckets[1:] {
		if b.Sessions > 0 && metric(b) > metric(worst) {
			worst = b
		}
	}
	if worst.Sessions == 0 {
		return noDataLabel
	}
	return worst.Label
}

func bestCacheDurationBucket(buckets []statspkg.CacheDurationBucket) string {
	if len(buckets) == 0 {
		return noDataLabel
	}
	best := statspkg.CacheDurationBucket{}
	for _, b := range buckets {
		if b.Sessions > 0 && b.HitRate > best.HitRate {
			best = b
		}
	}
	if best.Sessions == 0 {
		return noDataLabel
	}
	return best.Label
}

func formatRate(rate float64) string {
	return fmt.Sprintf("%.1f%%", rate*100)
}

func formatReuse(ratio float64) string {
	if ratio < 0.05 {
		return "0x"
	}
	return fmt.Sprintf("%.1fx", ratio)
}
