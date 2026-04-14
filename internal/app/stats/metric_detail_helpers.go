package stats

import (
	"fmt"
	"slices"
	"strings"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func leadingModelDetail(overview statspkg.Overview) (string, string) {
	if len(overview.ByModel) == 0 {
		return noDataLabel, "0%"
	}
	item := overview.ByModel[0]
	return item.Model, formatPercent(item.Tokens, overview.Tokens.Total)
}

func leadingProjectDetail(overview statspkg.Overview) (string, string) {
	if len(overview.ByProject) == 0 {
		return noDataLabel, "0%"
	}
	item := overview.ByProject[0]
	return item.Project, formatPercent(item.Tokens, overview.Tokens.Total)
}

func formatPercent(part, total int) string {
	if total <= 0 || part <= 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", float64(part)/float64(total)*100)
}

func activityMetricName(metric activityMetric) string {
	switch metric {
	case metricSessions:
		return "Sessions"
	case metricMessages:
		return "Messages"
	case metricTokens:
		return "Tokens"
	}
	return "Sessions"
}

func peakDailyCount(counts []statspkg.DailyCount) (string, int) {
	if len(counts) == 0 {
		return noDataLabel, 0
	}

	peak := counts[0]
	for _, count := range counts[1:] {
		if count.Count > peak.Count {
			peak = count
		}
	}
	return peak.Date.Format("2006-01-02"), peak.Count
}

func totalDailyCount(counts []statspkg.DailyCount) int {
	total := 0
	for _, count := range counts {
		total += count.Count
	}
	return total
}

func busiestHeatmapSlot(cells [7][24]int) (string, int) {
	dayNames := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	bestDay, bestHour, bestCount := 0, 0, 0
	for day := range 7 {
		for hour := range 24 {
			if cells[day][hour] > bestCount {
				bestDay = day
				bestHour = hour
				bestCount = cells[day][hour]
			}
		}
	}
	if bestCount == 0 {
		return noDataLabel, 0
	}
	return fmt.Sprintf("%s %02d:00", dayNames[bestDay], bestHour), bestCount
}

func dominantHistogramBucket(buckets []statspkg.HistogramBucket) (string, int) {
	if len(buckets) == 0 {
		return noDataLabel, 0
	}
	best := buckets[0]
	for _, bucket := range buckets[1:] {
		if bucket.Count > best.Count {
			best = bucket
		}
	}
	return best.Label, best.Count
}

func leadingTool(tools []statspkg.ToolStat) (string, int) {
	if len(tools) == 0 {
		return noDataLabel, 0
	}
	return tools[0].Name, tools[0].Count
}

func topToolRate(rates []statspkg.ToolRateStat) (string, string) {
	if len(rates) == 0 {
		return noDataLabel, "0.0%"
	}
	return rates[0].Name, formatToolRatePercent(rates[0].Rate)
}

func formatRatio(value float64) string {
	return fmt.Sprintf("%.1f:1", value)
}

func promptMetricDetailChips(
	metrics []statspkg.PositionTokenMetrics,
	mode statspkg.StatisticMode,
) []chip {
	early := averageTurnMetricRange(metrics, 1, 5, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AveragePromptTokens
	})
	late := averageTurnMetricRange(metrics, 20, 999, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AveragePromptTokens
	})
	return []chip{
		{Label: turnMetricRangeLabel("prompt", "1-5", mode), Value: formatFloat(early)},
		{Label: turnMetricRangeLabel("prompt", "20+", mode), Value: formatFloat(late)},
		{Label: turnMetricFactorLabel("prompt", mode), Value: formatTurnMetricMultiplier(early, late)},
	}
}

func turnCostMetricDetailChips(
	metrics []statspkg.PositionTokenMetrics,
	mode statspkg.StatisticMode,
) []chip {
	early := averageTurnMetricRange(metrics, 1, 5, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageTurnTokens
	})
	late := averageTurnMetricRange(metrics, 20, 999, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageTurnTokens
	})
	return []chip{
		{Label: turnMetricRangeLabel("turn cost", "1-5", mode), Value: formatFloat(early)},
		{Label: turnMetricRangeLabel("turn cost", "20+", mode), Value: formatFloat(late)},
		{Label: turnMetricFactorLabel("turn cost", mode), Value: formatTurnMetricMultiplier(early, late)},
	}
}

func turnMetricRangeLabel(prefix, window string, mode statspkg.StatisticMode) string {
	return prefix + " " + window + " " + mode.ShortLabel()
}

func turnMetricFactorLabel(prefix string, mode statspkg.StatisticMode) string {
	if mode == statspkg.StatisticModeAverage {
		return prefix + " multiplier"
	}
	return prefix + " " + mode.ShortLabel() + " multiplier"
}

func groupedTurnMetricDetailChips(m statsModel) []chip {
	if !m.groupScope.hasProvider() {
		return []chip{{Label: "provider", Value: "Select with v"}}
	}

	series := m.groupedTurnSeries()
	return []chip{
		{Label: "provider", Value: m.groupScope.provider.Label()},
		{Label: "versions", Value: statspkg.FormatNumber(len(series))},
		{Label: "mode", Value: "grouped"},
	}
}

func groupedTurnMetricScope(m statsModel) string {
	if !m.groupScope.hasProvider() {
		return "Choose a provider with v to compare versions."
	}
	if len(m.groupScope.versions) == 0 {
		return "All versions for the selected provider in the active range."
	}
	versions := make([]string, 0, len(m.groupScope.versions))
	for version := range m.groupScope.versions {
		versions = append(versions, version)
	}
	slices.Sort(versions)
	return "Selected versions: " + strings.Join(versions, ", ")
}

func providerVersionProviderCount(items []statspkg.ProviderVersionTokens) int {
	providers := make(map[string]bool, len(items))
	for _, item := range items {
		providers[item.Provider.Label()] = true
	}
	return len(providers)
}
