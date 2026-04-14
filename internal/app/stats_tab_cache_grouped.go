package app

import (
	"image/color"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) groupedCache() statspkg.CacheByVersion {
	return statspkg.ComputeCacheByVersion(
		m.statsSessions,
		m.timeRange,
		m.groupScope.provider,
		m.groupScope.versions,
	)
}

func (m statsModel) renderGroupedCacheTab(width int, cache statspkg.Cache) string {
	chips := renderSummaryChips(m.groupedCacheSummaryChips(cache), width)
	body := selectProviderWithVersionsPrompt
	if m.groupScope.hasProvider() {
		body = m.renderGroupedCacheBody(width)
	}
	return joinSections(chips, body, m.renderActiveMetricDetail(width))
}

func (m statsModel) groupedCacheSummaryChips(cache statspkg.Cache) []chip {
	chips := []chip{
		{Label: "mode", Value: "grouped"},
		{Label: "provider", Value: groupedProviderChipValue(m)},
	}
	if m.groupScope.hasProvider() {
		chips = append(chips, chip{Label: "versions", Value: statspkg.FormatNumber(len(m.groupedVersionLabels()))})
	}
	chips = append(chips,
		chip{Label: "hit rate", Value: formatRate(cache.HitRate)},
		chip{Label: "reuse", Value: formatReuse(cache.ReuseRatio)},
		chip{Label: "cache-rd", Value: statspkg.FormatNumber(cache.TotalCacheRead)},
		chip{Label: "cache-wr", Value: statspkg.FormatNumber(cache.TotalCacheWrite)},
	)
	return chips
}

func (m statsModel) renderGroupedCacheBody(width int) string {
	grouped := m.groupedCache()
	versions := m.groupedVersionLabels()
	colorByVersion := m.groupScopeColorMap()
	dailyTitle, dailyShares := m.groupedCacheDailyData(grouped)

	topPair := renderStatsLanePair(
		width,
		30,
		m.groupedProviderTitle(dailyTitle),
		m.cacheLaneCursor == 0,
		func(bodyWidth int) string {
			return renderChartWithVersionLegend(
				bodyWidth,
				versions,
				colorByVersion,
				24,
				func(chartWidth int) string {
					return renderGroupedDailyShareChartBody(
						dailyShares,
						chartWidth,
						11,
						colorByVersion,
					)
				},
			)
		},
		m.groupedProviderTitle("Main vs Subagent"),
		m.cacheLaneCursor == 1,
		func(bodyWidth int) string {
			return renderChartWithVersionLegend(
				bodyWidth,
				versions,
				colorByVersion,
				24,
				func(chartWidth int) string {
					return renderHorizontalStackedBarsBody(
						groupedCacheRows(grouped.SegmentRows, colorByVersion),
						chartWidth,
					)
				},
			)
		},
	)

	bottomPair := renderStatsLanePair(
		width,
		30,
		m.groupedProviderTitle("Cache Write by Duration"),
		m.cacheLaneCursor == 2,
		func(bodyWidth int) string {
			return renderChartWithVersionLegend(
				bodyWidth,
				versions,
				colorByVersion,
				24,
				func(chartWidth int) string {
					return renderVerticalStackedHistogramBody(
						groupedCacheDurationBuckets(grouped.WriteDuration, colorByVersion),
						chartWidth,
						7,
						statspkg.FormatNumber,
					)
				},
			)
		},
		m.groupedProviderTitle("Cache Read by Duration"),
		m.cacheLaneCursor == 3,
		func(bodyWidth int) string {
			return renderChartWithVersionLegend(
				bodyWidth,
				versions,
				colorByVersion,
				24,
				func(chartWidth int) string {
					return renderVerticalStackedHistogramBody(
						groupedCacheDurationBuckets(grouped.ReadDuration, colorByVersion),
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

func (m statsModel) groupedCacheDailyData(grouped statspkg.CacheByVersion) (string, []statspkg.GroupedDailyShare) {
	switch m.cacheMetric { //nolint:exhaustive
	case cacheMetricReuseRatio:
		return "Daily Cache Write Share", grouped.DailyWriteShare
	default:
		return "Daily Cache Read Share", grouped.DailyReadShare
	}
}

func groupedCacheRows(
	rows []statspkg.GroupedNamedStat,
	colorByVersion map[string]color.Color,
) []stackedRowItem {
	items := make([]stackedRowItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, stackedRowItem{
			Label:    row.Name,
			Scale:    float64(row.Total),
			Value:    statspkg.FormatNumber(row.Total),
			Segments: groupedRowSegments(row.Versions, colorByVersion),
		})
	}
	return items
}

func groupedCacheDurationBuckets(
	buckets []statspkg.GroupedHistogramBucket,
	colorByVersion map[string]color.Color,
) []stackedHistBucket {
	items := make([]stackedHistBucket, 0, len(buckets))
	for _, bucket := range buckets {
		items = append(items, stackedHistBucket{
			Label:    bucket.Label,
			Total:    bucket.Total,
			Segments: groupedSegments(bucket.Versions, colorByVersion),
		})
	}
	return items
}
