package app

import (
	"strings"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderGroupedToolsMetricDetail(width int, lane statsLane) string {
	if !m.groupScope.hasProvider() {
		return renderStatsMetricDetail(lane.title, width, []chip{
			{Label: "mode", Value: "grouped"},
			{Label: "provider", Value: "Select with v"},
		}, metricDetailLine("Scope", "Choose a provider with v to split tool metrics by version."))
	}

	grouped := m.groupedTools()
	baseChips := []chip{
		{Label: "provider", Value: m.groupScope.provider.Label()},
		{Label: "versions", Value: statspkg.FormatNumber(len(m.groupedVersionLabels()))},
	}
	switch lane.id { //nolint:exhaustive // grouped tools detail only handles tool lanes
	case statsLaneToolsCalls:
		bucket, count := dominantGroupedHistogramBucket(grouped.CallsPerSession)
		return renderStatsMetricDetail(m.groupedProviderTitle(lane.title), width, append(baseChips,
			chip{Label: "dominant bucket", Value: bucket},
			chip{Label: "sessions", Value: statspkg.FormatNumber(count)},
		),
			metricDetailLine("Question", "How is tool usage distributed across versions for the selected provider?"),
			metricDetailLine("Reading", "Each bucket stacks session counts by version within the active time range."),
			metricDetailLine("Scope", groupedTurnMetricScope(m)),
		)
	case statsLaneToolsTop:
		leader, total := leadingGroupedTool(grouped.TopTools)
		return renderStatsMetricDetail(m.groupedProviderTitle(lane.title), width, append(baseChips,
			chip{Label: "top tool", Value: leader},
			chip{Label: "calls", Value: statspkg.FormatNumber(total)},
		),
			metricDetailLine("Question", "Which tools dominate, and which versions contribute those calls?"),
			metricDetailLine("Reading", "Each bar keeps the total call volume and colors split that total by version."),
		)
	case statsLaneToolsErrors:
		name, rate := topGroupedToolRate(grouped.ToolErrorRates)
		return renderStatsMetricDetail(m.groupedProviderTitle(lane.title), width, append(baseChips,
			chip{Label: "top rate", Value: name},
			chip{Label: "error rate", Value: rate},
		),
			metricDetailLine("Question", "Which tools fail, and which versions contribute those failures?"),
			metricDetailLine(
				"Reading",
				"Bar length stays tied to the overall error rate, while colors split the failing calls by version.",
			),
		)
	default:
		name, rate := topGroupedToolRate(grouped.ToolRejectRates)
		return renderStatsMetricDetail(m.groupedProviderTitle(lane.title), width, append(baseChips,
			chip{Label: "top rate", Value: name},
			chip{Label: "rejected", Value: rate},
		),
			metricDetailLine("Question", "Which tool suggestions are rejected, and which versions drive those rejections?"),
			metricDetailLine(
				"Reading",
				"Bar length stays tied to the total rejection rate, while colors split rejected calls by version.",
			),
		)
	}
}

func (m statsModel) renderGroupedCacheMetricDetail(width int, lane statsLane) string {
	if !m.groupScope.hasProvider() {
		return renderStatsMetricDetail(lane.title, width, []chip{
			{Label: "mode", Value: "grouped"},
			{Label: "provider", Value: "Select with v"},
		}, metricDetailLine("Scope", "Choose a provider with v to split cache metrics by version."))
	}

	grouped := m.groupedCache()
	baseChips := []chip{
		{Label: "provider", Value: m.groupScope.provider.Label()},
		{Label: "versions", Value: statspkg.FormatNumber(len(m.groupedVersionLabels()))},
	}
	switch lane.id { //nolint:exhaustive // grouped cache detail only handles cache lanes
	case statsLaneCacheDaily:
		title, shares := m.groupedCacheDailyData(grouped)
		peakDay, peakRate := peakGroupedDailyShare(shares)
		return renderStatsMetricDetail(m.groupedProviderTitle(title), width, append(baseChips,
			chip{Label: "peak day", Value: peakDay},
			chip{Label: "peak share", Value: formatRate(peakRate)},
		),
			metricDetailLine("Question", "How much of prompt traffic is cache read or write share each day?"),
			metricDetailLine("Reading", "Each day keeps the total share and colors split the tokens by version."),
		)
	case statsLaneCacheSegment:
		name, total := leadingGroupedTool(grouped.SegmentRows)
		return renderStatsMetricDetail(m.groupedProviderTitle(lane.title), width, append(baseChips,
			chip{Label: "largest row", Value: name},
			chip{Label: "tokens", Value: statspkg.FormatNumber(total)},
		),
			metricDetailLine("Question", "How do main-thread and subagent cache components split by version?"),
			metricDetailLine("Reading", "Each row keeps the total tokens for that cache component, split by version."),
		)
	case statsLaneCacheReuse:
		name, total := leadingGroupedHistogram(grouped.WriteDuration)
		return renderStatsMetricDetail(m.groupedProviderTitle("Cache Write by Duration"), width, append(baseChips,
			chip{Label: "largest bucket", Value: name},
			chip{Label: "tokens", Value: statspkg.FormatNumber(total)},
		),
			metricDetailLine("Question", "Which session durations write the most cache tokens, split by version?"),
			metricDetailLine("Reading", "Each duration bucket stacks cache-write tokens by version."),
		)
	default:
		name, total := leadingGroupedHistogram(grouped.ReadDuration)
		return renderStatsMetricDetail(m.groupedProviderTitle("Cache Read by Duration"), width, append(baseChips,
			chip{Label: "largest bucket", Value: name},
			chip{Label: "tokens", Value: statspkg.FormatNumber(total)},
		),
			metricDetailLine("Question", "Which session durations read the most cache tokens, split by version?"),
			metricDetailLine("Reading", "Each duration bucket stacks cache-read tokens by version."),
		)
	}
}

func dominantGroupedHistogramBucket(buckets []statspkg.GroupedHistogramBucket) (string, int) {
	best := statspkg.GroupedHistogramBucket{}
	for _, bucket := range buckets {
		if bucket.Total > best.Total {
			best = bucket
		}
	}
	if best.Label == "" {
		return noDataLabel, 0
	}
	return best.Label, best.Total
}

func leadingGroupedTool(items []statspkg.GroupedNamedStat) (string, int) {
	if len(items) == 0 {
		return noDataLabel, 0
	}
	return items[0].Name, items[0].Total
}

func topGroupedToolRate(items []statspkg.GroupedRateStat) (string, string) {
	if len(items) == 0 {
		return noDataLabel, "0.0%"
	}
	return items[0].Name, formatToolRatePercent(items[0].Rate)
}

func peakGroupedDailyShare(shares []statspkg.GroupedDailyShare) (string, float64) {
	best := statspkg.GroupedDailyShare{}
	bestRate := 0.0
	for _, share := range shares {
		rate := 0.0
		if share.Prompt > 0 {
			rate = float64(share.Total) / float64(share.Prompt)
		}
		if rate > bestRate {
			best = share
			bestRate = rate
		}
	}
	if best.Date.IsZero() {
		return noDataLabel, 0
	}
	return best.Date.Format("2006-01-02"), bestRate
}

func leadingGroupedHistogram(items []statspkg.GroupedHistogramBucket) (string, int) {
	best := statspkg.GroupedHistogramBucket{}
	for _, item := range items {
		if item.Total > best.Total {
			best = item
		}
	}
	if strings.TrimSpace(best.Label) == "" {
		return noDataLabel, 0
	}
	return best.Label, best.Total
}
