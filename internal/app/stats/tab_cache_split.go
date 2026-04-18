package stats

import (
	"image/color"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderSplitCacheTab(width int, cache statspkg.Cache) string {
	chips := renderSummaryChips(m.theme, m.splitCacheSummaryChips(cache), width)
	body := m.renderSplitCacheBody(width)
	return joinSections(chips, body, m.renderActiveMetricDetail(width))
}

func (m statsModel) splitCacheSummaryChips(cache statspkg.Cache) []chip {
	chips := []chip{
		{Label: "mode", Value: "split"},
		{Label: "by", Value: m.splitBy.Label()},
		{Label: "series", Value: statspkg.FormatNumber(len(m.splitKeys()))},
		{Label: "hit rate", Value: formatRate(cache.HitRate)},
		{Label: "reuse", Value: formatReuse(cache.ReuseRatio)},
		{Label: "cache-rd", Value: statspkg.FormatNumber(cache.TotalCacheRead)},
		{Label: "cache-wr", Value: statspkg.FormatNumber(cache.TotalCacheWrite)},
	}
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

	topPair := renderStatsLanePair(
		m.theme,
		width,
		30,
		m.splitTitle(dailyTitle),
		m.cacheLaneCursor == 0,
		func(bodyWidth int) string {
			return renderChartWithSplitLegend(
				m.theme,
				bodyWidth,
				dailyKeys,
				colorByKey,
				splitChartMinWidth,
				func(chartWidth int) string {
					return renderSplitDailyShareChartBody(
						m.theme,
						dailyShares,
						chartWidth,
						11,
						colorByKey,
					)
				},
			)
		},
		m.splitTitle("Main vs Subagent"),
		m.cacheLaneCursor == 1,
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
		m.cacheLaneCursor == 2,
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
		m.cacheLaneCursor == 3,
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
	return topPair + "\n\n" + bottomPair
}

func (m statsModel) splitCacheDailyData(grouped statspkg.CacheBySplit) (string, []statspkg.SplitDailyShare) {
	switch m.cacheMetric { //nolint:exhaustive
	case cacheMetricReuseRatio:
		return "Daily Cache Write Share", grouped.DailyWriteShare
	default:
		return "Daily Cache Read Share", grouped.DailyReadShare
	}
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
