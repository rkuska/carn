package stats

import (
	"fmt"
	"math"
)

type performanceSeriesAggregate interface {
	performanceSampleCount() int
}

func performanceMetricFromCounts[T performanceSeriesAggregate](
	id, label string,
	currentNumerator, currentDenominator int,
	baselineNumerator, baselineDenominator int,
	higherIsBetter bool,
	minSamples int,
	floor float64,
	detail string,
	context performanceMetricContext[T],
	extract func(T) (int, int),
) PerformanceMetric {
	return performanceMetricFromRatio(
		id,
		label,
		float64(currentNumerator),
		float64(currentDenominator),
		float64(baselineNumerator),
		float64(baselineDenominator),
		higherIsBetter,
		minSamples,
		floor,
		detail,
		context,
		func(agg T) (float64, float64) {
			numerator, denominator := extract(agg)
			return float64(numerator), float64(denominator)
		},
	)
}

func performanceMetricFromRatio[T performanceSeriesAggregate](
	id, label string,
	currentNumerator, currentDenominator float64,
	baselineNumerator, baselineDenominator float64,
	higherIsBetter bool,
	minSamples int,
	floor float64,
	detail string,
	context performanceMetricContext[T],
	extract func(T) (float64, float64),
) PerformanceMetric {
	currentValue := safeRatio(currentNumerator, currentDenominator)
	baselineValue := safeRatio(baselineNumerator, baselineDenominator)
	currentSamples := performanceRelevantSampleCount(currentDenominator, context.currentSampleCount)
	baselineSamples := performanceRelevantSampleCount(baselineDenominator, context.baselineSampleCount)
	score := scorePerformanceMetric(
		currentValue,
		baselineValue,
		higherIsBetter,
		currentSamples,
		baselineSamples,
		minSamples,
		floor,
	)
	return PerformanceMetric{
		ID:          id,
		Label:       label,
		Value:       FormatValue(id, currentValue),
		Detail:      detail,
		Current:     currentValue,
		Baseline:    baselineValue,
		HasBaseline: score.hasBaseline,
		Score:       score.score,
		ScoreWeight: 1,
		HasScore:    score.hasScore,
		Trend:       score.trend,
		SampleCount: currentSamples,
		Series:      performanceSeries(context, extract),
	}
}

func performanceDiagnosticFromRatio[T performanceSeriesAggregate](
	label string,
	currentNumerator, currentDenominator float64,
	baselineNumerator, baselineDenominator float64,
	higherIsBetter bool,
	minSamples int,
	detail string,
	context performanceMetricContext[T],
	extract func(T) (float64, float64),
) PerformanceDiagnostic {
	currentValue := safeRatio(currentNumerator, currentDenominator)
	baselineValue := safeRatio(baselineNumerator, baselineDenominator)
	currentSamples := performanceRelevantSampleCount(currentDenominator, context.currentSampleCount)
	baselineSamples := performanceRelevantSampleCount(baselineDenominator, context.baselineSampleCount)
	score := scorePerformanceMetric(
		currentValue,
		baselineValue,
		higherIsBetter,
		currentSamples,
		baselineSamples,
		minSamples,
		0.05,
	)
	return PerformanceDiagnostic{
		Group:       "provider_signals",
		Label:       label,
		Value:       formatPerformanceRatio(currentValue),
		Detail:      detail,
		Current:     currentValue,
		Baseline:    baselineValue,
		HasBaseline: score.hasBaseline,
		Trend:       score.trend,
		Series:      performanceSeries(context, extract),
	}
}

func performanceAverageDiagnostic(
	label string,
	currentTotal, currentSamples int,
	baselineTotal, baselineSamples int,
	detail string,
) PerformanceDiagnostic {
	currentValue := safeRatio(float64(currentTotal), float64(currentSamples))
	baselineValue := safeRatio(float64(baselineTotal), float64(baselineSamples))
	score := scorePerformanceMetric(
		currentValue,
		baselineValue,
		true,
		currentSamples,
		baselineSamples,
		performanceMinSessionSamples,
		1,
	)
	return PerformanceDiagnostic{
		Group:       "provider_signals",
		Label:       label,
		Value:       FormatNumber(int(currentValue)),
		Detail:      detail,
		Current:     currentValue,
		Baseline:    baselineValue,
		HasBaseline: score.hasBaseline,
		Trend:       score.trend,
	}
}

func performanceTopCountDiagnostic(
	label string,
	currentCounts, baselineCounts map[string]int,
	detail string,
) PerformanceDiagnostic {
	currentName, currentValue := topCountEntry(currentCounts)
	baselineName, baselineValue := topCountEntry(baselineCounts)
	trend := TrendDirectionNone
	hasBaseline := baselineName != ""
	if hasBaseline && currentName == baselineName {
		trend = performanceTrendDirection(float64(currentValue-baselineValue) / max(float64(baselineValue), 1.0))
	}
	value := "n/a"
	if currentName != "" {
		value = fmt.Sprintf("%s (%s)", currentName, FormatNumber(currentValue))
	}
	return PerformanceDiagnostic{
		Group:       "provider_signals",
		Label:       label,
		Value:       value,
		Detail:      detail,
		Current:     float64(currentValue),
		Baseline:    float64(baselineValue),
		HasBaseline: hasBaseline,
		Trend:       trend,
	}
}

func performanceSeries[T performanceSeriesAggregate](
	context performanceMetricContext[T],
	extract func(T) (float64, float64),
) []PerformancePoint {
	if len(context.bucketAggregates) == 0 {
		return nil
	}

	points := make([]PerformancePoint, 0, len(context.bucketAggregates))
	for _, bucket := range context.bucketAggregates {
		numerator, denominator := extract(bucket.aggregate)
		points = append(points, PerformancePoint{
			Timestamp:   bucket.bucket.start,
			Value:       safeRatio(numerator, denominator),
			SampleCount: performanceRelevantSampleCount(denominator, bucket.aggregate.performanceSampleCount()),
		})
	}
	return points
}

func performanceRelevantSampleCount(denominator float64, fallback int) int {
	if denominator <= 0 {
		return 0
	}
	return max(int(math.Round(denominator)), fallbackCountFloor(fallback))
}

func fallbackCountFloor(fallback int) int {
	if fallback <= 0 {
		return 0
	}
	return min(fallback, 1)
}

// FormatValue renders a performance metric value using the metric's display convention.
func FormatValue(id string, value float64) string {
	if performanceMetricIsRatio(id) {
		return formatPerformanceRatio(value)
	}
	switch id {
	case perfMetricTokensPerTurn, perfMetricActionsPerTurn:
		return fmt.Sprintf("%.1f", value)
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

func formatPerformanceRatio(value float64) string {
	return fmt.Sprintf("%.1f%%", value*100)
}

func safeRatio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	return numerator / denominator
}
