package stats

import (
	"image/color"

	el "github.com/rkuska/carn/internal/app/elements"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderSplitToolsTab(width int, tools statspkg.Tools) string {
	chips := renderSummaryChips(m.theme, m.splitToolsSummaryChips(tools), width)
	body := m.renderSplitToolsBody(width)
	return joinSections(chips, body)
}

func (m statsModel) splitToolsSummaryChips(tools statspkg.Tools) []chip {
	chips := []chip{
		{Label: "mode", Value: "split"},
		{Label: "by", Value: m.splitBy.Label()},
		{Label: "series", Value: statspkg.FormatNumber(len(m.splitKeys()))},
		{Label: "total calls", Value: statspkg.FormatNumber(tools.TotalCalls)},
		{Label: "error rate", Value: toolRateChipValue(tools.ErrorRate)},
		{Label: "rejected", Value: toolRateChipValue(tools.RejectionRate)},
	}
	return chips
}

func (m statsModel) renderSplitToolsBody(width int) string {
	grouped := m.splitToolsResult
	colorByKey := m.splitColors
	callsKeys := presentSplitKeys(grouped.CallsPerSession, histBucketSplits)
	topKeys := presentSplitKeys(grouped.TopTools, namedStatSplits)
	errorKeys := presentSplitKeys(grouped.ToolErrorRates, rateStatSplits)
	rejectKeys := presentSplitKeys(grouped.ToolRejectRates, rateStatSplits)

	usageCharts := renderStatsLanePair(
		m.theme,
		width,
		30,
		m.splitTitle("Tool Calls/Session"),
		m.toolsLaneCursor == 0,
		func(bodyWidth int) string {
			return renderChartWithSplitLegend(
				m.theme,
				bodyWidth,
				callsKeys,
				colorByKey,
				splitChartMinWidth,
				func(chartWidth int) string {
					return renderVerticalStackedHistogramBody(
						m.theme,
						splitToolCallBuckets(grouped.CallsPerSession, colorByKey),
						chartWidth,
						5,
						statspkg.FormatNumber,
					)
				},
			)
		},
		m.splitTitle("Top Tools"),
		m.toolsLaneCursor == 1,
		func(bodyWidth int) string {
			return renderChartWithSplitLegend(
				m.theme,
				bodyWidth,
				topKeys,
				colorByKey,
				splitChartMinWidth,
				func(chartWidth int) string {
					return renderHorizontalStackedBarsBody(
						splitToolTopRows(grouped.TopTools, colorByKey),
						chartWidth,
					)
				},
			)
		},
	)

	qualityCharts := renderStatsLanePair(
		m.theme,
		width,
		30,
		m.splitTitle("Tool Error Rate"),
		m.toolsLaneCursor == 2,
		func(bodyWidth int) string {
			return renderChartWithSplitLegend(
				m.theme,
				bodyWidth,
				errorKeys,
				colorByKey,
				splitChartMinWidth,
				func(chartWidth int) string {
					return renderHorizontalStackedBarsBody(
						splitToolRateRows(m.theme, grouped.ToolErrorRates, colorByKey, true),
						chartWidth,
					)
				},
			)
		},
		m.splitTitle(statsRejectedSuggestionsTitle),
		m.toolsLaneCursor == 3,
		func(bodyWidth int) string {
			return renderChartWithSplitLegend(
				m.theme,
				bodyWidth,
				rejectKeys,
				colorByKey,
				splitChartMinWidth,
				func(chartWidth int) string {
					return renderHorizontalStackedBarsBody(
						splitToolRateRows(m.theme, grouped.ToolRejectRates, colorByKey, false),
						chartWidth,
					)
				},
			)
		},
	)
	return usageCharts + "\n\n" + qualityCharts
}

func splitToolCallBuckets(
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

func splitToolTopRows(
	items []statspkg.SplitNamedStat,
	colorByKey map[string]color.Color,
) []stackedRowItem {
	rows := make([]stackedRowItem, 0, len(items))
	for _, item := range items {
		rows = append(rows, stackedRowItem{
			Label:    item.Name,
			Scale:    float64(item.Total),
			Value:    statspkg.FormatNumber(item.Total),
			Segments: splitRowSegmentsForValues(item.Splits, colorByKey),
		})
	}
	return rows
}

func splitToolRateRows(
	theme *el.Theme,
	items []statspkg.SplitRateStat,
	colorByKey map[string]color.Color,
	showCount bool,
) []stackedRowItem {
	rows := make([]stackedRowItem, 0, len(items))
	for _, item := range items {
		rows = append(rows, stackedRowItem{
			Label: item.Name,
			Scale: item.Rate,
			Value: renderToolRateValue(theme, statspkg.ToolRateStat{
				Name: item.Name, Count: item.Count, Total: item.Total, Rate: item.Rate,
			}, showCount),
			Segments: splitRowSegmentsForValues(item.Splits, colorByKey),
		})
	}
	return rows
}
