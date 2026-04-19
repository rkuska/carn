package stats

import (
	"image/color"
	"slices"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderSplitCacheTab(width int, cache statspkg.Cache) string {
	chips := renderSummaryChips(m.theme, m.splitCacheSummaryChips(cache), width)
	body := m.renderSplitCacheBody(width)
	return joinSections(chips, body)
}

func (m statsModel) splitCacheSummaryChips(cache statspkg.Cache) []chip {
	chips := []chip{
		{Label: "mode", Value: "split"},
		{Label: "by", Value: m.splitBy.Label()},
		{Label: "series", Value: statspkg.FormatNumber(len(m.splitKeys()))},
		{Label: "overall hit", Value: formatRate(cache.HitRate)},
		{Label: "overall write", Value: formatRate(cache.WriteRate)},
	}
	chips = append(chips, splitCacheRateChips(m.splitCacheResult)...)
	chips = append(chips,
		chip{Label: "cache-rd", Value: statspkg.FormatNumber(cache.TotalCacheRead)},
		chip{Label: "cache-wr", Value: statspkg.FormatNumber(cache.TotalCacheWrite)},
	)
	return chips
}

func (m statsModel) renderSplitCacheBody(width int) string {
	grouped := m.splitCacheResult
	colorByKey := m.splitColors
	dailyTitle, dailyShares := m.splitCacheDailyData(grouped)
	dailyKeys := presentSplitKeys(dailyShares, dailyShareSplits)
	segmentKeys := presentSplitKeys(grouped.SegmentRows, namedStatSplits)
	writeKeys := presentSplitKeys(grouped.WriteDuration, histBucketSplits)
	readKeys := presentSplitKeys(grouped.ReadDuration, histBucketSplits)
	selected, _, _ := m.selectedStatsLane()

	topPair := renderStatsLanePair(
		m.theme,
		width,
		30,
		m.splitTitle(dailyTitle),
		selected.id == statsLaneCacheDaily,
		func(bodyWidth int) string {
			return m.renderSplitCacheDailyChart(bodyWidth, dailyKeys, dailyShares, colorByKey)
		},
		m.splitTitle("Main vs Subagent"),
		selected.id == statsLaneCacheSegment,
		func(bodyWidth int) string {
			return renderChartWithSplitLegend(
				m.theme,
				bodyWidth,
				segmentKeys,
				colorByKey,
				splitChartMinWidth,
				func(chartWidth int) string {
					return renderHorizontalStackedBarsBody(
						splitCacheRows(grouped.SegmentRows, colorByKey),
						chartWidth,
					)
				},
			)
		},
	)

	bottomPair := renderStatsLanePair(
		m.theme,
		width,
		30,
		m.splitTitle("Cache Write by Duration"),
		selected.id == statsLaneCacheReuse,
		func(bodyWidth int) string {
			return renderChartWithSplitLegend(
				m.theme,
				bodyWidth,
				writeKeys,
				colorByKey,
				splitChartMinWidth,
				func(chartWidth int) string {
					return renderVerticalStackedHistogramBody(
						m.theme,
						splitCacheDurationBuckets(grouped.WriteDuration, colorByKey),
						chartWidth,
						7,
						statspkg.FormatNumber,
					)
				},
			)
		},
		m.splitTitle("Cache Read by Duration"),
		selected.id == statsLaneCacheHitDur,
		func(bodyWidth int) string {
			return renderChartWithSplitLegend(
				m.theme,
				bodyWidth,
				readKeys,
				colorByKey,
				splitChartMinWidth,
				func(chartWidth int) string {
					return renderVerticalStackedHistogramBody(
						m.theme,
						splitCacheDurationBuckets(grouped.ReadDuration, colorByKey),
						chartWidth,
						7,
						statspkg.FormatNumber,
					)
				},
			)
		},
	)
	firstTurnLane := m.renderCacheFirstTurnLane(m.snapshot.Cache, width, selected.id == statsLaneCacheFirstTurn)
	return topPair + "\n\n" + bottomPair + "\n\n" + firstTurnLane
}

func (m statsModel) splitCacheDailyData(grouped statspkg.CacheBySplit) (string, []statspkg.SplitDailyShare) {
	switch m.cacheMetric { //nolint:exhaustive
	case cacheMetricReuseRatio:
		return "Daily Cache Write Rate", grouped.DailyWriteShare
	default:
		return "Daily Cache Read Rate", grouped.DailyReadShare
	}
}

func (m statsModel) renderSplitCacheDailyChart(
	width int,
	dailyKeys []string,
	dailyShares []statspkg.SplitDailyShare,
	colorByKey map[string]color.Color,
) string {
	return renderChartWithSplitLegend(
		m.theme,
		width,
		dailyKeys,
		colorByKey,
		splitChartMinWidth,
		func(chartWidth int) string {
			return renderSplitDailyRateChartBody(
				m.theme,
				splitCacheDailyRateSeries(dailyShares),
				chartWidth,
				11,
				colorByKey,
			)
		},
	)
}

func splitCacheRateChips(
	grouped statspkg.CacheBySplit,
) []chip {
	type rates struct {
		read  float64
		write float64
	}

	readByKey := make(map[string]int)
	writeByKey := make(map[string]int)
	promptByKey := make(map[string]int)
	for i, row := range grouped.SegmentRows {
		for _, split := range row.Splits {
			promptByKey[split.Key] += split.Value
			if i < 2 {
				readByKey[split.Key] += split.Value
				continue
			}
			if i < 4 {
				writeByKey[split.Key] += split.Value
			}
		}
	}

	byKey := make(map[string]rates, len(promptByKey))
	keys := make([]string, 0, len(promptByKey))
	for key, prompt := range promptByKey {
		keys = append(keys, key)
		if prompt <= 0 {
			continue
		}
		byKey[key] = rates{
			read:  float64(readByKey[key]) / float64(prompt),
			write: float64(writeByKey[key]) / float64(prompt),
		}
	}
	slices.Sort(keys)

	chips := make([]chip, 0, len(keys)*2)
	for _, key := range keys {
		rate := byKey[key]
		chips = append(chips,
			chip{Label: key + " hit", Value: formatRate(rate.read)},
			chip{Label: key + " write", Value: formatRate(rate.write)},
		)
	}
	return chips
}

func splitCacheDailyRateSeries(
	shares []statspkg.SplitDailyShare,
) []statspkg.SplitDailyRateSeries {
	keys := make([]string, 0)
	seen := make(map[string]bool)
	for _, share := range shares {
		for _, split := range share.PromptSplits {
			if seen[split.Key] {
				continue
			}
			seen[split.Key] = true
			keys = append(keys, split.Key)
		}
	}
	slices.Sort(keys)
	if len(keys) == 0 {
		return nil
	}

	series := make([]statspkg.SplitDailyRateSeries, len(keys))
	for i, key := range keys {
		series[i] = statspkg.SplitDailyRateSeries{
			Key:   key,
			Rates: make([]statspkg.DailyRate, 0, len(shares)),
		}
	}

	for _, share := range shares {
		values := splitValueLookup(share.Splits)
		prompts := splitValueLookup(share.PromptSplits)
		for i, key := range keys {
			rate := 0.0
			hasActivity := false
			if prompt := prompts[key]; prompt > 0 {
				rate = float64(values[key]) / float64(prompt)
				hasActivity = true
			}
			series[i].Rates = append(series[i].Rates, statspkg.DailyRate{
				Date:        share.Date,
				Rate:        rate,
				HasActivity: hasActivity,
			})
		}
	}

	return series
}

func splitValueLookup(values []statspkg.SplitValue) map[string]int {
	lookup := make(map[string]int, len(values))
	for _, value := range values {
		lookup[value.Key] = value.Value
	}
	return lookup
}

func splitCacheRows(
	rows []statspkg.SplitNamedStat,
	colorByKey map[string]color.Color,
) []stackedRowItem {
	items := make([]stackedRowItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, stackedRowItem{
			Label:    row.Name,
			Scale:    float64(row.Total),
			Value:    statspkg.FormatNumber(row.Total),
			Segments: splitRowSegmentsForValues(row.Splits, colorByKey),
		})
	}
	return items
}

func splitCacheDurationBuckets(
	buckets []statspkg.SplitHistogramBucket,
	colorByKey map[string]color.Color,
) []stackedHistBucket {
	items := make([]stackedHistBucket, 0, len(buckets))
	for _, bucket := range buckets {
		items = append(items, stackedHistBucket{
			Label:    bucket.Label,
			Total:    bucket.Total,
			Segments: splitHistSegments(bucket.Splits, colorByKey),
		})
	}
	return items
}
