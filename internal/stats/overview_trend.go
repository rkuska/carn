package stats

import (
	"math"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

const overviewFlatTrendThreshold = 5.0

func ComputeTokenTrend(sessions []conv.SessionMeta, timeRange TimeRange) TokenTrend {
	if timeRange.Start.IsZero() && timeRange.End.IsZero() {
		return TokenTrend{}
	}

	previousRange := previousTimeRange(timeRange)
	previousSessions := FilterByTimeRange(sessions, previousRange)
	if len(previousSessions) == 0 {
		return TokenTrend{}
	}

	currentTokens := totalTokensForSessions(FilterByTimeRange(sessions, timeRange))
	previousTokens := totalTokensForSessions(previousSessions)
	if previousTokens <= 0 {
		return TokenTrend{}
	}

	change := float64(currentTokens-previousTokens) / float64(previousTokens) * 100
	if math.Abs(change) < overviewFlatTrendThreshold {
		return TokenTrend{Direction: TrendDirectionFlat}
	}
	if change > 0 {
		return TokenTrend{
			Direction:     TrendDirectionUp,
			PercentChange: int(math.Round(change)),
		}
	}
	return TokenTrend{
		Direction:     TrendDirectionDown,
		PercentChange: int(math.Round(change)),
	}
}

func ComputeTokenTrendFromBuckets(
	activityBuckets []conv.ActivityBucketRow,
	timeRange TimeRange,
) TokenTrend {
	if timeRange.Start.IsZero() && timeRange.End.IsZero() {
		return TokenTrend{}
	}

	previousRange := previousTimeRange(timeRange)
	previousTokens := totalTokensForActivityBuckets(activityBuckets, previousRange)
	if previousTokens <= 0 {
		return TokenTrend{}
	}

	currentTokens := totalTokensForActivityBuckets(activityBuckets, timeRange)
	change := float64(currentTokens-previousTokens) / float64(previousTokens) * 100
	if math.Abs(change) < overviewFlatTrendThreshold {
		return TokenTrend{Direction: TrendDirectionFlat}
	}
	if change > 0 {
		return TokenTrend{
			Direction:     TrendDirectionUp,
			PercentChange: int(math.Round(change)),
		}
	}
	return TokenTrend{
		Direction:     TrendDirectionDown,
		PercentChange: int(math.Round(change)),
	}
}

func previousTimeRange(current TimeRange) TimeRange {
	duration := current.End.Sub(current.Start)
	end := current.Start.Add(-time.Nanosecond)
	start := end.Add(-duration)
	return TimeRange{Start: start, End: end}
}

func totalTokensForSessions(sessions []conv.SessionMeta) int {
	total := 0
	for _, session := range sessions {
		total += session.TotalUsage.TotalTokens()
	}
	return total
}

func totalTokensForActivityBuckets(activityBuckets []conv.ActivityBucketRow, timeRange TimeRange) int {
	if len(activityBuckets) == 0 {
		return 0
	}

	location := time.UTC
	switch {
	case !timeRange.Start.IsZero():
		location = timeRange.Start.Location()
	case !timeRange.End.IsZero():
		location = timeRange.End.Location()
	case !activityBuckets[0].BucketStart.IsZero() && activityBuckets[0].BucketStart.Location() != nil:
		location = activityBuckets[0].BucketStart.Location()
	}

	start := startOfDayInLocation(timeRange.Start, location)
	end := startOfDayInLocation(timeRange.End, location)
	total := 0
	for _, row := range activityBuckets {
		day := activityBucketDay(row.BucketStart, location)
		if !day.Before(start) && !day.After(end) {
			total += activityBucketTotalTokens(row)
		}
	}
	return total
}
