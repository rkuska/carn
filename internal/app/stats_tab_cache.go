package app

import (
	"fmt"
	"image/color"
	"math"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderCacheTab(width, height int) string {
	cache := m.snapshot.Cache
	if m.cacheGrouped {
		return m.renderGroupedCacheTab(width, cache)
	}
	chips := renderSummaryChips(m.cacheSummaryChips(cache), width)

	chartTitle, rates, chartColor, yFmt := m.cacheDailySeries()
	chartHeight := 12
	if height < 18 {
		chartHeight = max(height-6, 6)
	}

	topPair := renderStatsLanePair(
		width, 30,
		chartTitle, m.cacheLaneCursor == 0,
		func(bodyWidth int) string {
			return renderDailyRateChartBody(rates, max(bodyWidth, 10), chartHeight, chartColor, yFmt)
		},
		"Main vs Subagent", m.cacheLaneCursor == 1,
		func(bodyWidth int) string {
			return renderHorizontalBarsBody(cacheSegmentBars(cache), bodyWidth, colorChartToken)
		},
	)

	reuseBuckets := cacheReuseBuckets(cache.DurationBuckets)
	hitBuckets := cacheHitRateBuckets(cache.DurationBuckets)
	histHeight := 8

	bottomPair := renderStatsLanePair(
		width, 30,
		"Cache Reuse by Duration", m.cacheLaneCursor == 2,
		func(bodyWidth int) string {
			return renderVerticalHistogramBody(reuseBuckets, bodyWidth, histHeight, colorChartToken)
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

func (m statsModel) cacheDailySeries() (string, []statspkg.DailyRate, color.Color, linechart.LabelFormatter) {
	switch m.cacheMetric { //nolint:exhaustive // only cache metrics handled here
	case cacheMetricReuseRatio:
		rates, cap, hasInfinite := normalizeReuseDailyRates(m.snapshot.Cache.DailyReuseRatio)
		return "Daily Reuse Ratio", rates, colorChartBar, reuseYLabel(cap, hasInfinite)
	default:
		return "Daily Hit Rate", m.snapshot.Cache.DailyHitRate, colorChartToken, percentYLabel()
	}
}

func percentYLabel() linechart.LabelFormatter {
	return func(_ int, v float64) string {
		return fmt.Sprintf("%.0f%%", v*100)
	}
}

func reuseYLabel(maxValue float64, hasInfinite bool) linechart.LabelFormatter {
	return func(_ int, v float64) string {
		if hasInfinite && v >= maxValue {
			return "inf"
		}
		return fmt.Sprintf("%.1fx", v)
	}
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

func cacheReuseBuckets(durations []statspkg.CacheDurationBucket) []histBucket {
	buckets := make([]histBucket, 0, len(durations))
	maxFiniteCount := 0
	infiniteIndexes := make([]int, 0)
	for _, d := range durations {
		if math.IsInf(d.ReuseRatio, 1) {
			infiniteIndexes = append(infiniteIndexes, len(buckets))
			buckets = append(buckets, histBucket{
				Label:   d.Label,
				Display: "inf",
			})
			continue
		}
		count := int(math.Round(d.ReuseRatio * 100))
		if count > maxFiniteCount {
			maxFiniteCount = count
		}
		buckets = append(buckets, histBucket{
			Label: d.Label,
			Count: count,
		})
	}
	if len(infiniteIndexes) == 0 {
		return buckets
	}
	if maxFiniteCount == 0 {
		maxFiniteCount = 100
	}
	for _, index := range infiniteIndexes {
		buckets[index].Count = maxFiniteCount
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
	if m.cacheGrouped {
		return m.renderGroupedCacheMetricDetail(width, lane)
	}

	cache := m.snapshot.Cache
	switch lane.id { //nolint:exhaustive // only cache lanes handled here
	case statsLaneCacheDaily:
		return m.renderCacheDailyMetricDetail(cache, lane.title, width)
	case statsLaneCacheSegment:
		return renderCacheSegmentMetricDetail(cache, width)
	case statsLaneCacheReuse:
		return renderCacheReuseMetricDetail(cache, width)
	case statsLaneCacheHitDur:
		return renderCacheHitDurMetricDetail(cache, width)
	default:
		return renderStatsMetricDetail("Cache", width, nil, noDataLabel)
	}
}

func (m statsModel) renderCacheDailyMetricDetail(cache statspkg.Cache, title string, width int) string {
	switch m.cacheMetric { //nolint:exhaustive // only cache metrics handled here
	case cacheMetricReuseRatio:
		return m.renderCacheDailyReuseDetail(cache, title, width)
	default:
		return m.renderCacheDailyHitRateDetail(cache, title, width)
	}
}

func (m statsModel) renderCacheDailyHitRateDetail(cache statspkg.Cache, title string, width int) string {
	peakDay, peakRate := peakDailyRate(cache.DailyHitRate)
	return renderStatsMetricDetail(title, width, []chip{
		{Label: "peak day", Value: peakDay},
		{Label: "peak hit rate", Value: formatRate(peakRate)},
		{Label: "overall hit rate", Value: formatRate(cache.HitRate)},
	},
		metricDetailLine("Question", "How does cache hit rate change over time?"),
		metricDetailLine(
			"Reading",
			"Columns are daily buckets. Bars mean active days, dots mean no sessions, "+
				"and baseline marks mean worked but hit 0%. Press m to toggle to reuse ratio.",
		),
	)
}

func (m statsModel) renderCacheDailyReuseDetail(cache statspkg.Cache, title string, width int) string {
	peakDay, peakRate := peakDailyRate(cache.DailyReuseRatio)
	return renderStatsMetricDetail(title, width, []chip{
		{Label: "peak day", Value: peakDay},
		{Label: "peak reuse", Value: formatReuse(peakRate)},
		{Label: "overall reuse", Value: formatReuse(cache.ReuseRatio)},
	},
		metricDetailLine("Question", "How does cache write leverage change over time?"),
		metricDetailLine(
			"Reading",
			"Columns are daily buckets. Bars mean active days, dots mean no sessions, "+
				"and baseline marks mean worked but got no cache reuse. Press m to toggle.",
		),
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

func renderCacheReuseMetricDetail(cache statspkg.Cache, width int) string {
	best := bestCacheReuseBucket(cache.DurationBuckets)
	return renderStatsMetricDetail("Cache Reuse by Duration", width, []chip{
		{Label: "best reuse bucket", Value: best},
		{Label: "overall reuse", Value: formatReuse(cache.ReuseRatio)},
	},
		metricDetailLine("Question", "Which session durations leverage cache writes best?"),
		metricDetailLine(
			"Reading",
			"Taller bars mean more cache reads per write. "+
				"Low reuse signals wasted writes from TTL expiry or context shifts.",
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

func bestCacheReuseBucket(buckets []statspkg.CacheDurationBucket) string {
	if len(buckets) == 0 {
		return noDataLabel
	}
	best := statspkg.CacheDurationBucket{}
	for _, b := range buckets {
		if b.Sessions > 0 && b.ReuseRatio > best.ReuseRatio {
			best = b
		}
	}
	if best.Sessions == 0 {
		return noDataLabel
	}
	return best.Label
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

func peakDailyRate(rates []statspkg.DailyRate) (string, float64) {
	if len(rates) == 0 {
		return noDataLabel, 0
	}
	peak := rates[0]
	for _, r := range rates[1:] {
		if r.Rate > peak.Rate {
			peak = r
		}
	}
	if peak.Rate == 0 {
		return noDataLabel, 0
	}
	return peak.Date.Format("Jan 02"), peak.Rate
}

func formatRate(rate float64) string {
	return fmt.Sprintf("%.1f%%", rate*100)
}

func formatReuse(ratio float64) string {
	if math.IsInf(ratio, 1) {
		return "inf"
	}
	if ratio < 0.05 {
		return "0x"
	}
	return fmt.Sprintf("%.1fx", ratio)
}

func normalizeReuseDailyRates(rates []statspkg.DailyRate) ([]statspkg.DailyRate, float64, bool) {
	if len(rates) == 0 {
		return nil, 1, false
	}

	maxFinite := 0.0
	hasInfinite := false
	for _, rate := range rates {
		if math.IsInf(rate.Rate, 1) {
			hasInfinite = true
			continue
		}
		if rate.Rate > maxFinite {
			maxFinite = rate.Rate
		}
	}

	cap := max(maxFinite, 0.01)
	if hasInfinite && cap < 1 {
		cap = 1
	}

	normalized := make([]statspkg.DailyRate, len(rates))
	copy(normalized, rates)
	if !hasInfinite {
		return normalized, cap, false
	}
	for i := range normalized {
		if math.IsInf(normalized[i].Rate, 1) {
			normalized[i].Rate = cap
		}
	}
	return normalized, cap, true
}
