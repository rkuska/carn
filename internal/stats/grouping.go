package stats

import (
	"slices"
	"strings"
)

// UnknownVersionLabel is preserved as an alias for UnknownSplitKey because
// callers normalize blank session versions to the same fallback as other
// blank split keys.
const UnknownVersionLabel = UnknownSplitKey

func NormalizeVersionLabel(version string) string {
	return labelOrUnknown(version)
}

func ComputeTurnTokenMetricsBySplit(
	sessions []SessionTurnMetrics,
	timeRange TimeRange,
	dim SplitDimension,
	allowed map[string]bool,
	mode StatisticMode,
) []SplitTurnSeries {
	if len(sessions) == 0 || !dim.SupportsTurnMetrics() {
		return nil
	}

	grouped := groupTurnMetricsBySplit(sessions, dim, allowed)
	if len(grouped) == 0 {
		return nil
	}

	return buildSplitTurnSeries(grouped, timeRange, mode)
}

func groupTurnMetricsBySplit(
	sessions []SessionTurnMetrics,
	dim SplitDimension,
	allowed map[string]bool,
) map[string][]SessionTurnMetrics {
	grouped := make(map[string][]SessionTurnMetrics)
	for _, session := range sessions {
		key := dim.TurnMetricsKey(session)
		if key == "" {
			continue
		}
		if len(allowed) > 0 && !allowed[key] {
			continue
		}
		grouped[key] = append(grouped[key], session)
	}
	return grouped
}

func buildSplitTurnSeries(
	grouped map[string][]SessionTurnMetrics,
	timeRange TimeRange,
	mode StatisticMode,
) []SplitTurnSeries {
	items := make([]SplitTurnSeries, 0, len(grouped))
	for key, rows := range grouped {
		metrics := ComputeTurnTokenMetricsForRangeWithMode(rows, timeRange, mode)
		if len(metrics) == 0 {
			continue
		}
		items = append(items, SplitTurnSeries{
			Key:     key,
			Metrics: metrics,
		})
	}
	slices.SortFunc(items, compareSplitTurnSeries)
	return items
}

func compareSplitTurnSeries(left, right SplitTurnSeries) int {
	return strings.Compare(left.Key, right.Key)
}
