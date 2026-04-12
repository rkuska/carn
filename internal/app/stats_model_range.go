package app

import (
	"time"

	"github.com/rkuska/carn/internal/stats"
)

func statsRange30d() stats.TimeRange {
	return statsRangeDays(30)
}

func statsRange90d() stats.TimeRange {
	return statsRangeDays(90)
}

func statsRange7d() stats.TimeRange {
	return statsRangeDays(7)
}

func statsRangeDays(days int) stats.TimeRange {
	now := statsNow()
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

func nextStatsTimeRange(current stats.TimeRange) stats.TimeRange {
	switch statsTimeRangeLabel(current) {
	case statsRangeLabel7d:
		return statsRange30d()
	case statsRangeLabel30d:
		return statsRange90d()
	case statsRangeLabel90d:
		return stats.TimeRange{}
	default:
		return statsRange7d()
	}
}

func statsTimeRangeLabel(current stats.TimeRange) string {
	switch {
	case current.Start.IsZero() && current.End.IsZero():
		return statsRangeLabelAll
	default:
		days := int(current.End.Sub(current.Start).Hours()/24) + 1
		switch days {
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
