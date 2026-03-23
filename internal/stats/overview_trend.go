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
