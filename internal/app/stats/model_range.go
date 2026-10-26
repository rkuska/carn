package stats

import (
	"time"

	"github.com/rkuska/carn/internal/stats"
)

func statsRange30d(now time.Time) stats.TimeRange {
	return statsRangeDays(now, 30)
}

func statsRange90d(now time.Time) stats.TimeRange {
	return statsRangeDays(now, 90)
}

func statsRange7d(now time.Time) stats.TimeRange {
	return statsRangeDays(now, 7)
}

// statsRangeDays returns the calendar days ending on now's day, in now's
// location: midnight days-1 days earlier through the last instant of today.
func statsRangeDays(now time.Time, days int) stats.TimeRange {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).
		AddDate(0, 0, -(days - 1))
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), now.Location())
	return stats.TimeRange{Start: start, End: end}
}

func nextStatsTab(tab statsTab) statsTab {
	return statsTab((int(tab) + 1) % 6)
}

func prevStatsTab(tab statsTab) statsTab {
	return statsTab((int(tab) + 5) % 6)
}

func nextCacheMetric(metric cacheMetric) cacheMetric {
	return cacheMetric((int(metric) + 1) % 2)
}

func nextActivityMetric(metric activityMetric) activityMetric {
	return activityMetric((int(metric) + 1) % 3)
}

func nextStatsTimeRange(current stats.TimeRange, now time.Time) stats.TimeRange {
	switch statsTimeRangeLabel(current) {
	case statsRangeLabel7d:
		return statsRange30d(now)
	case statsRangeLabel30d:
		return statsRange90d(now)
	case statsRangeLabel90d:
		return stats.TimeRange{}
	default:
		return statsRange7d(now)
	}
}

func statsTimeRangeLabel(current stats.TimeRange) string {
	switch {
	case current.Start.IsZero() && current.End.IsZero():
		return statsRangeLabelAll
	default:
		switch statsRangeCalendarDays(current) {
		case 7:
			return statsRangeLabel7d
		case 30:
			return statsRangeLabel30d
		case 90:
			return statsRangeLabel90d
		default:
			return statsRangeLabelAll
		}
	}
}

// statsRangeCalendarDays counts the calendar days a range spans, inclusive.
// It compares calendar dates rather than elapsed hours so that a range
// crossing a daylight-saving transition (a 23h or 25h day) keeps its length.
func statsRangeCalendarDays(current stats.TimeRange) int {
	start := time.Date(current.Start.Year(), current.Start.Month(), current.Start.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(current.End.Year(), current.End.Month(), current.End.Day(), 0, 0, 0, 0, time.UTC)
	return int(end.Sub(start).Hours()/24) + 1
}
