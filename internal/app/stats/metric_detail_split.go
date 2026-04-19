package stats

import (
	"slices"
	"strings"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func splitDetailChips(splitChips []chip, extra ...chip) []chip {
	out := slices.Clone(splitChips)
	return append(out, extra...)
}

func (m statsModel) renderSplitToolsMetricDetail(width int, lane statsLane) string {
	grouped := m.splitToolsResult
	splitChips := splitTurnMetricDetailChips(m)
	switch lane.id { //nolint:exhaustive // split tools detail only handles tool lanes
	case statsLaneToolsCalls:
		bucket, count := dominantSplitHistogramBucket(grouped.CallsPerSession)
		chips := splitDetailChips(splitChips,
			chip{Label: "dominant bucket", Value: bucket},
			chip{Label: "sessions", Value: statspkg.FormatNumber(count)},
		)
		return m.renderStatsMetricDetailBody(lane.title, width, chips,
			m.metricDetailLine("Question", "Is tool use light and frequent or concentrated in a few heavy sessions?"),
			m.metricDetailLine(
				"Reading",
				"The X-axis is call-count bucket and the Y-axis is session count. "+m.colorsStackSuffix(),
			),
		)
	case statsLaneToolsTop:
		leader, total := leadingSplitTool(grouped.TopTools)
		chips := splitDetailChips(splitChips,
			chip{Label: "top tool", Value: leader},
			chip{Label: "calls", Value: statspkg.FormatNumber(total)},
		)
		return m.renderStatsMetricDetailBody(lane.title, width, chips,
			m.metricDetailLine("Question", "Which tools dominate the workflow?"),
			m.metricDetailLine(
				"Reading",
				"Longer bars mean more total calls in the active slice. "+m.colorsSplitSuffix("each bar"),
			),
		)
	case statsLaneToolsErrors:
		name, rate := topSplitToolRate(grouped.ToolErrorRates)
		chips := splitDetailChips(splitChips,
			chip{Label: "top rate", Value: name},
			chip{Label: "error rate", Value: rate},
		)
		return m.renderStatsMetricDetailBody(lane.title, width, chips,
			m.metricDetailLine("Question", "Which tools are failing often enough to inspect?"),
			m.metricDetailLine(
				"Reading",
				"Rates exclude user-declined suggestions and show absolute failures alongside percentage. "+
					m.colorsSplitSuffix("failing calls"),
			),
		)
	default:
		name, rate := topSplitToolRate(grouped.ToolRejectRates)
		chips := splitDetailChips(splitChips,
			chip{Label: "top rate", Value: name},
			chip{Label: "rejected", Value: rate},
		)
		return m.renderStatsMetricDetailBody(lane.title, width, chips,
			m.metricDetailLine("Question", "Which proposed tools are users pushing back on before execution?"),
			m.metricDetailLine(
				"Reading",
				"Higher rates mean stronger user resistance to the suggested tool choice. "+
					m.colorsSplitSuffix("rejected calls"),
			),
		)
	}
}

func (m statsModel) renderSplitCacheMetricDetail(width int, lane statsLane) string {
	grouped := m.splitCacheResult
	splitChips := splitTurnMetricDetailChips(m)
	switch lane.id { //nolint:exhaustive // split cache detail only handles cache lanes
	case statsLaneCacheDaily:
		title, shares := m.splitCacheDailyData(grouped)
		peakSeries, peakDay, peakRate := peakSplitDailyRate(shares)
		chips := splitDetailChips(splitChips,
			chip{Label: "peak series", Value: peakSeries},
			chip{Label: "peak day", Value: peakDay},
			chip{Label: "peak rate", Value: formatRate(peakRate)},
		)
		return m.renderStatsMetricDetailBody(title, width, chips,
			m.metricDetailLine("Question", "How do split series compare on daily cache efficiency?"),
			m.metricDetailLine(
				"Reading",
				"Each day bucket uses a shared Y-axis and splits bars by the active series, "+
					"so read and write rates stay directly comparable.",
			),
		)
	case statsLaneCacheSegment:
		name, total := leadingSplitTool(grouped.SegmentRows)
		chips := splitDetailChips(splitChips,
			chip{Label: "largest row", Value: name},
			chip{Label: "tokens", Value: statspkg.FormatNumber(total)},
		)
		return m.renderStatsMetricDetailBody(lane.title, width, chips,
			m.metricDetailLine("Question", "Does the main thread cache better than subagents?"),
			m.metricDetailLine(
				"Reading",
				"Rows compare main vs subagent cache components and the X-axis is total tokens. "+
					m.colorsSplitSuffix("each row"),
			),
		)
	case statsLaneCacheReuse:
		name, total := leadingSplitHistogram(grouped.WriteDuration)
		chips := splitDetailChips(splitChips,
			chip{Label: "largest bucket", Value: name},
			chip{Label: "tokens", Value: statspkg.FormatNumber(total)},
		)
		return m.renderStatsMetricDetailBody("Cache Write by Duration", width, chips,
			m.metricDetailLine("Question", "Which session durations write the most cache tokens?"),
			m.metricDetailLine(
				"Reading",
				"The X-axis is session duration bucket and the Y-axis is cache-write token count. "+
					m.colorsStackSuffix(),
			),
		)
	case statsLaneCacheFirstTurn:
		return m.renderCacheFirstTurnMetricDetail(m.snapshot.Cache, width)
	default:
		name, total := leadingSplitHistogram(grouped.ReadDuration)
		chips := splitDetailChips(splitChips,
			chip{Label: "largest bucket", Value: name},
			chip{Label: "tokens", Value: statspkg.FormatNumber(total)},
		)
		return m.renderStatsMetricDetailBody("Cache Read by Duration", width, chips,
			m.metricDetailLine("Question", "Which session durations read the most cache tokens?"),
			m.metricDetailLine(
				"Reading",
				"The X-axis is session duration bucket and the Y-axis is cache-read token count. "+
					m.colorsStackSuffix(),
			),
		)
	}
}

func (m statsModel) colorsStackSuffix() string {
	return "Colors stack each bar by " + m.splitBy.Label() + "."
}

func (m statsModel) colorsSplitSuffix(target string) string {
	return "Colors split " + target + " by " + m.splitBy.Label() + "."
}

func dominantSplitHistogramBucket(buckets []statspkg.SplitHistogramBucket) (string, int) {
	best := statspkg.SplitHistogramBucket{}
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

func leadingSplitTool(items []statspkg.SplitNamedStat) (string, int) {
	if len(items) == 0 {
		return noDataLabel, 0
	}
	return items[0].Name, items[0].Total
}

func topSplitToolRate(items []statspkg.SplitRateStat) (string, string) {
	if len(items) == 0 {
		return noDataLabel, "0.0%"
	}
	return items[0].Name, formatToolRatePercent(items[0].Rate)
}

func peakSplitDailyRate(shares []statspkg.SplitDailyShare) (string, string, float64) {
	bestSeries := noDataLabel
	bestRate := 0.0
	bestDay := noDataLabel
	for _, item := range splitCacheDailyRateSeries(shares) {
		for _, rate := range item.Rates {
			if !rate.HasActivity || rate.Rate <= bestRate {
				continue
			}
			bestSeries = item.Key
			bestDay = rate.Date.Format("2006-01-02")
			bestRate = rate.Rate
		}
	}
	return bestSeries, bestDay, bestRate
}

func leadingSplitHistogram(items []statspkg.SplitHistogramBucket) (string, int) {
	best := statspkg.SplitHistogramBucket{}
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
