package stats

import (
	"fmt"
	"time"

	statspkg "github.com/rkuska/carn/internal/stats"
)

const performanceNotAvailable = "n/a"

func formatPerformanceScore(score statspkg.PerformanceScore) string {
	if !score.HasScore {
		return "low sample"
	}
	return fmt.Sprintf("%d %s", score.Score, trendGlyph(score.Trend))
}

func formatPerformanceLaneScore(lane statspkg.PerformanceLane) string {
	return formatPerformanceScore(statspkg.PerformanceScore{
		Score:    lane.Score,
		HasScore: lane.HasScore,
		Trend:    lane.Trend,
	})
}

func trendGlyph(direction statspkg.TrendDirection) string {
	switch direction {
	case statspkg.TrendDirectionNone:
		return "·"
	case statspkg.TrendDirectionUp:
		return "↑"
	case statspkg.TrendDirectionDown:
		return "↓"
	case statspkg.TrendDirectionFlat:
		return "→"
	}
	return "·"
}

func performanceVerdictText(score statspkg.PerformanceScore) string {
	if !score.HasScore {
		return "Low sample"
	}
	switch score.Trend {
	case statspkg.TrendDirectionNone:
		return "Unclear"
	case statspkg.TrendDirectionUp:
		return "Improving"
	case statspkg.TrendDirectionDown:
		return "Declining"
	case statspkg.TrendDirectionFlat:
		return "Stable"
	default:
		return "Unclear"
	}
}

func performanceMetricStatusText(status statspkg.PerformanceMetricStatus) string {
	switch status {
	case statspkg.PerformanceMetricStatusNone:
		return performanceNotAvailable
	case statspkg.PerformanceMetricStatusBetter:
		return "better"
	case statspkg.PerformanceMetricStatusWorse:
		return "worse"
	case statspkg.PerformanceMetricStatusFlat:
		return "flat"
	case statspkg.PerformanceMetricStatusLowSample:
		return "low sample"
	default:
		return performanceNotAvailable
	}
}

func performanceBaselineValue(metric statspkg.PerformanceMetric) string {
	if !metric.HasBaseline {
		return performanceNotAvailable
	}
	if metric.ID == "" {
		return fmt.Sprintf("%.1f", metric.Baseline)
	}
	return statspkg.FormatValue(metric.ID, metric.Baseline)
}

func performanceDelta(metric statspkg.PerformanceMetric) string {
	if metric.DeltaText == "" {
		return performanceNotAvailable
	}
	return metric.DeltaText
}

func performanceDirection(higherIsBetter bool) string {
	if higherIsBetter {
		return "higher"
	}
	return "lower"
}

func performanceScopeSummary(scope statspkg.PerformanceScope) string {
	if scope.SingleFamily {
		return scope.PrimaryProvider + " / " + scope.PrimaryModel
	}
	return fmt.Sprintf("%d providers / %d models", len(scope.Providers), len(scope.Models))
}

func formatPerformanceTimeRange(timeRange statspkg.TimeRange) string {
	if timeRange.Start.IsZero() || timeRange.End.IsZero() {
		return performanceNotAvailable
	}
	return formatPerformanceDate(timeRange.Start) + " - " + formatPerformanceDate(timeRange.End)
}

func formatPerformanceDate(ts time.Time) string {
	return ts.Format("2006-01-02")
}
