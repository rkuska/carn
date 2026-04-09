package stats

import "fmt"

func performanceMetricFromCounts(
	id, label string,
	currentNumerator, currentDenominator int,
	baselineNumerator, baselineDenominator int,
	higherIsBetter bool,
	floor float64,
	detail string,
	context performanceMetricContext,
	extract func(performanceAggregate) (int, int),
) PerformanceMetric {
	return performanceMetricFromRatio(
		id,
		label,
		float64(currentNumerator),
		float64(currentDenominator),
		float64(baselineNumerator),
		float64(baselineDenominator),
		higherIsBetter,
		floor,
		detail,
		context,
		func(agg performanceAggregate) (float64, float64) {
			numerator, denominator := extract(agg)
			return float64(numerator), float64(denominator)
		},
	)
}

func performanceMetricFromRatio(
	id, label string,
	currentNumerator, currentDenominator float64,
	baselineNumerator, baselineDenominator float64,
	higherIsBetter bool,
	floor float64,
	detail string,
	context performanceMetricContext,
	extract func(performanceAggregate) (float64, float64),
) PerformanceMetric {
	currentValue := safeRatio(currentNumerator, currentDenominator)
	baselineValue := safeRatio(baselineNumerator, baselineDenominator)
	score := scorePerformanceMetric(
		currentValue,
		baselineValue,
		higherIsBetter,
		context.currentSampleCount,
		context.baselineSampleCount,
		floor,
	)
	return PerformanceMetric{
		ID:          id,
		Label:       label,
		Value:       formatPerformanceValue(id, currentValue),
		Detail:      detail,
		Current:     currentValue,
		Baseline:    baselineValue,
		HasBaseline: score.hasBaseline,
		Score:       score.score,
		HasScore:    score.hasScore,
		Trend:       score.trend,
		SampleCount: context.currentSampleCount,
		Series:      performanceSeries(context, extract),
	}
}

func performanceDiagnosticFromRatio(
	label string,
	currentNumerator, currentDenominator float64,
	baselineNumerator, baselineDenominator float64,
	detail string,
	context performanceMetricContext,
	extract func(performanceAggregate) (float64, float64),
) PerformanceDiagnostic {
	currentValue := safeRatio(currentNumerator, currentDenominator)
	baselineValue := safeRatio(baselineNumerator, baselineDenominator)
	score := scorePerformanceMetric(
		currentValue,
		baselineValue,
		true,
		context.currentSampleCount,
		context.baselineSampleCount,
		0.05,
	)
	return PerformanceDiagnostic{
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
		1,
	)
	return PerformanceDiagnostic{
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
		trend = performanceTrendDirection(float64(currentValue-baselineValue) / maxFloat(float64(baselineValue), 1))
	}
	value := "n/a"
	if currentName != "" {
		value = fmt.Sprintf("%s (%s)", currentName, FormatNumber(currentValue))
	}
	return PerformanceDiagnostic{
		Label:       label,
		Value:       value,
		Detail:      detail,
		Current:     float64(currentValue),
		Baseline:    float64(baselineValue),
		HasBaseline: hasBaseline,
		Trend:       trend,
	}
}

func performanceSeries(
	context performanceMetricContext,
	extract func(performanceAggregate) (float64, float64),
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
			SampleCount: bucket.aggregate.sessionCount,
		})
	}
	return points
}

func formatPerformanceValue(id string, value float64) string {
	switch id {
	case perfMetricVerificationPass,
		perfMetricRewriteRate,
		perfMetricReasoningShare,
		perfMetricErrorRate,
		perfMetricRejectionRate,
		perfMetricAbortRate,
		perfMetricContextPressure,
		perfMetricRetryBurden:
		return formatPerformanceRatio(value)
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

func maxFloat(values ...float64) float64 {
	result := 0.0
	for _, value := range values {
		if value > result {
			result = value
		}
	}
	return result
}
