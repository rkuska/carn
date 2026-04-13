package stats

import (
	"math"
	"slices"
	"time"
)

const (
	performanceMinSessionSamples  = 10
	performanceMinSequenceSamples = 5
)

type performanceWindow struct {
	current  TimeRange
	baseline TimeRange
}

type performanceBucket struct {
	start time.Time
	end   time.Time
}

type performanceMetricScore struct {
	score       int
	hasScore    bool
	trend       TrendDirection
	hasBaseline bool
}

func performanceTimeWindow[T any](
	timeRange TimeRange,
	sessions []T,
	timestamp func(T) time.Time,
) performanceWindow {
	current := normalizePerformanceTimeRange(timeRange, sessions, timestamp)
	if current.Start.IsZero() || current.End.IsZero() {
		return performanceWindow{current: current}
	}

	duration := current.End.Sub(current.Start)
	if duration < 0 {
		return performanceWindow{current: current}
	}
	baselineEnd := current.Start.Add(-time.Nanosecond)
	baselineStart := baselineEnd.Add(-duration)
	return performanceWindow{
		current: current,
		baseline: TimeRange{
			Start: baselineStart,
			End:   baselineEnd,
		},
	}
}

func normalizePerformanceTimeRange[T any](
	timeRange TimeRange,
	sessions []T,
	timestamp func(T) time.Time,
) TimeRange {
	if !timeRange.Start.IsZero() || !timeRange.End.IsZero() {
		return timeRange
	}
	if len(sessions) == 0 {
		return TimeRange{}
	}

	start := timestamp(sessions[0])
	end := start
	for _, session := range sessions[1:] {
		sessionTimestamp := timestamp(session)
		if sessionTimestamp.Before(start) {
			start = sessionTimestamp
		}
		if sessionTimestamp.After(end) {
			end = sessionTimestamp
		}
	}
	if start.IsZero() || end.IsZero() {
		return TimeRange{}
	}
	return TimeRange{Start: start, End: end}
}

func performanceBuckets(timeRange TimeRange) []performanceBucket {
	if timeRange.Start.IsZero() || timeRange.End.IsZero() || timeRange.End.Before(timeRange.Start) {
		return nil
	}

	step := 24 * time.Hour
	if timeRange.End.Sub(timeRange.Start) > 60*24*time.Hour {
		step = 7 * 24 * time.Hour
	}

	buckets := make([]performanceBucket, 0, 16)
	for start := timeRange.Start; !start.After(timeRange.End); start = start.Add(step) {
		end := start.Add(step - time.Nanosecond)
		if end.After(timeRange.End) {
			end = timeRange.End
		}
		buckets = append(buckets, performanceBucket{start: start, end: end})
	}
	return buckets
}

func scorePerformanceMetric(
	current, baseline float64,
	higherIsBetter bool,
	sampleCount, baselineSampleCount int,
	minSamples int,
	floor float64,
) performanceMetricScore {
	if minSamples <= 0 {
		minSamples = performanceMinSessionSamples
	}
	if sampleCount < minSamples || baselineSampleCount < minSamples {
		return performanceMetricScore{}
	}
	if floor <= 0 {
		floor = 1
	}

	scale := math.Max(math.Abs(baseline), floor)
	delta := (current - baseline) / scale
	if !higherIsBetter {
		delta = -delta
	}
	score := 50 + int(math.Round(delta*50))
	score = min(max(score, 0), 100)
	return performanceMetricScore{
		score:       score,
		hasScore:    true,
		trend:       performanceTrendDirection(delta),
		hasBaseline: true,
	}
}

func performanceTrendDirection(delta float64) TrendDirection {
	switch {
	case math.Abs(delta) < 0.05:
		return TrendDirectionFlat
	case delta > 0:
		return TrendDirectionUp
	default:
		return TrendDirectionDown
	}
}

func combinePerformanceScores(metrics []PerformanceMetric) PerformanceScore {
	totalScore := 0.0
	totalWeight := 0.0
	trendSum := 0.0
	for _, metric := range metrics {
		if !metric.HasScore {
			continue
		}
		if metric.ScoreWeight <= 0 {
			continue
		}
		totalScore += float64(metric.Score) * metric.ScoreWeight
		totalWeight += metric.ScoreWeight
		trendSum += float64(trendScore(metric.Trend)) * metric.ScoreWeight
	}
	if totalWeight == 0 {
		return PerformanceScore{}
	}
	return PerformanceScore{
		Score:    int(math.Round(totalScore / totalWeight)),
		HasScore: true,
		Trend:    aggregateWeightedTrend(trendSum),
	}
}

func aggregateWeightedTrend(sum float64) TrendDirection {
	switch {
	case sum > 0:
		return TrendDirectionUp
	case sum < 0:
		return TrendDirectionDown
	default:
		return TrendDirectionFlat
	}
}

func trendScore(direction TrendDirection) int {
	switch direction {
	case TrendDirectionNone:
		return 0
	case TrendDirectionUp:
		return 1
	case TrendDirectionDown:
		return -1
	case TrendDirectionFlat:
		return 0
	}
	return 0
}

func sortedStringKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}
