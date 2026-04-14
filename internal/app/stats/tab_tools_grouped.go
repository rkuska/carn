package stats

import (
	"image/color"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) groupedTools() statspkg.ToolsByVersion {
	return statspkg.ComputeToolsByVersion(
		m.statsSessions,
		m.timeRange,
		m.groupScope.provider,
		m.groupScope.versions,
	)
}

func (m statsModel) renderGroupedToolsTab(width int, tools statspkg.Tools) string {
	chips := renderSummaryChips(m.groupedToolsSummaryChips(tools), width)
	body := selectProviderWithVersionsPrompt
	if m.groupScope.hasProvider() {
		body = m.renderGroupedToolsBody(width)
	}
	return joinSections(chips, body, m.renderActiveMetricDetail(width))
}

func (m statsModel) groupedToolsSummaryChips(tools statspkg.Tools) []chip {
	chips := []chip{
		{Label: "mode", Value: "grouped"},
		{Label: "provider", Value: groupedProviderChipValue(m)},
	}
	if m.groupScope.hasProvider() {
		chips = append(chips, chip{Label: "versions", Value: statspkg.FormatNumber(len(m.groupedVersionLabels()))})
	}
	chips = append(chips,
		chip{Label: "total calls", Value: statspkg.FormatNumber(tools.TotalCalls)},
		chip{Label: "error rate", Value: toolRateChipValue(tools.ErrorRate)},
		chip{Label: "rejected", Value: toolRateChipValue(tools.RejectionRate)},
	)
	return chips
}

func groupedProviderChipValue(m statsModel) string {
	if !m.groupScope.hasProvider() {
		return "Select with v"
	}
	return m.groupScope.provider.Label()
}

func (m statsModel) renderGroupedToolsBody(width int) string {
	grouped := m.groupedTools()
	versions := m.groupedVersionLabels()
	colorByVersion := m.groupScopeColorMap()

	usageCharts := renderStatsLanePair(
		width,
		30,
		m.groupedProviderTitle("Tool Calls/Session"),
		m.toolsLaneCursor == 0,
		func(bodyWidth int) string {
			return renderChartWithVersionLegend(
				bodyWidth,
				versions,
				colorByVersion,
				24,
				func(chartWidth int) string {
					return renderVerticalStackedHistogramBody(
						groupedToolCallBuckets(grouped.CallsPerSession, colorByVersion),
						chartWidth,
						5,
						statspkg.FormatNumber,
					)
				},
			)
		},
		m.groupedProviderTitle("Top Tools"),
		m.toolsLaneCursor == 1,
		func(bodyWidth int) string {
			return renderChartWithVersionLegend(
				bodyWidth,
				versions,
				colorByVersion,
				24,
				func(chartWidth int) string {
					return renderHorizontalStackedBarsBody(
						groupedToolTopRows(grouped.TopTools, colorByVersion),
						chartWidth,
					)
				},
			)
		},
	)

	qualityCharts := renderStatsLanePair(
		width,
		30,
		m.groupedProviderTitle("Tool Error Rate"),
		m.toolsLaneCursor == 2,
		func(bodyWidth int) string {
			return renderChartWithVersionLegend(
				bodyWidth,
				versions,
				colorByVersion,
				24,
				func(chartWidth int) string {
					return renderHorizontalStackedBarsBody(
						groupedToolRateRows(grouped.ToolErrorRates, colorByVersion, true),
						chartWidth,
					)
				},
			)
		},
		m.groupedProviderTitle(statsRejectedSuggestionsTitle),
		m.toolsLaneCursor == 3,
		func(bodyWidth int) string {
			return renderChartWithVersionLegend(
				bodyWidth,
				versions,
				colorByVersion,
				24,
				func(chartWidth int) string {
					return renderHorizontalStackedBarsBody(
						groupedToolRateRows(grouped.ToolRejectRates, colorByVersion, false),
						chartWidth,
					)
				},
			)
		},
	)
	return usageCharts + "\n\n" + qualityCharts
}

func groupedToolCallBuckets(
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

func groupedToolTopRows(
	items []statspkg.GroupedNamedStat,
	colorByVersion map[string]color.Color,
) []stackedRowItem {
	rows := make([]stackedRowItem, 0, len(items))
	for _, item := range items {
		rows = append(rows, stackedRowItem{
			Label:    item.Name,
			Scale:    float64(item.Total),
			Value:    statspkg.FormatNumber(item.Total),
			Segments: groupedRowSegments(item.Versions, colorByVersion),
		})
	}
	return rows
}

func groupedToolRateRows(
	items []statspkg.GroupedRateStat,
	colorByVersion map[string]color.Color,
	showCount bool,
) []stackedRowItem {
	rows := make([]stackedRowItem, 0, len(items))
	for _, item := range items {
		rows = append(rows, stackedRowItem{
			Label: item.Name,
			Scale: item.Rate,
			Value: renderToolRateValue(statspkg.ToolRateStat{
				Name: item.Name, Count: item.Count, Total: item.Total, Rate: item.Rate,
			}, showCount),
			Segments: groupedRowSegments(item.Versions, colorByVersion),
		})
	}
	return rows
}

func groupedSegments(
	versions []statspkg.VersionValue,
	colorByVersion map[string]color.Color,
) []stackedHistSegment {
	segments := make([]stackedHistSegment, 0, len(versions))
	for _, version := range versions {
		segments = append(segments, stackedHistSegment{
			Value: version.Value,
			Color: colorByVersion[version.Version],
		})
	}
	return segments
}

func groupedRowSegments(
	versions []statspkg.VersionValue,
	colorByVersion map[string]color.Color,
) []stackedRowSegment {
	segments := make([]stackedRowSegment, 0, len(versions))
	for _, version := range versions {
		segments = append(segments, stackedRowSegment{
			Value: version.Value,
			Color: colorByVersion[version.Version],
		})
	}
	return segments
}
