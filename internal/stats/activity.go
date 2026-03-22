package stats

import (
	"slices"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

func ComputeActivity(sessions []conv.SessionMeta, timeRange TimeRange) Activity {
	start, end, ok := resolveActivityBounds(sessions, timeRange)
	if !ok {
		return Activity{}
	}

	activity := Activity{
		DailySessions: make([]DailyCount, 0),
		DailyMessages: make([]DailyCount, 0),
		DailyTokens:   make([]DailyCount, 0),
	}
	sessionsByDay := make(map[time.Time]int)
	messagesByDay := make(map[time.Time]int)
	tokensByDay := make(map[time.Time]int)
	activeDates := make(map[time.Time]struct{})

	for _, session := range sessions {
		day := startOfDay(session.Timestamp)
		sessionsByDay[day]++
		messagesByDay[day] += session.MainMessageCount
		tokensByDay[day] += session.TotalUsage.TotalTokens()
		activeDates[day] = struct{}{}

		weekday := weekdayIndex(session.Timestamp)
		hour := session.Timestamp.Hour()
		activity.Heatmap[weekday][hour]++
	}

	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		activity.TotalDays++
		activity.DailySessions = append(activity.DailySessions, DailyCount{Date: day, Count: sessionsByDay[day]})
		activity.DailyMessages = append(activity.DailyMessages, DailyCount{Date: day, Count: messagesByDay[day]})
		activity.DailyTokens = append(activity.DailyTokens, DailyCount{Date: day, Count: tokensByDay[day]})
		if _, ok := activeDates[day]; ok {
			activity.ActiveDays++
		}
	}

	activity.CurrentStreak = countBackwardStreak(activeDates, end)
	activity.LongestStreak = countLongestStreak(activeDates)
	return activity
}

func resolveActivityBounds(
	sessions []conv.SessionMeta,
	timeRange TimeRange,
) (time.Time, time.Time, bool) {
	if len(sessions) == 0 {
		return time.Time{}, time.Time{}, false
	}

	minDay, maxDay := activitySessionBounds(sessions)
	start := timeRange.Start
	end := timeRange.End
	if start.IsZero() {
		start = minDay
	}
	if end.IsZero() {
		end = maxDay
	}

	start = startOfDay(start)
	end = startOfDay(end)
	if end.Before(start) {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func activitySessionBounds(sessions []conv.SessionMeta) (time.Time, time.Time) {
	minDay := startOfDay(sessions[0].Timestamp)
	maxDay := minDay
	for _, session := range sessions[1:] {
		day := startOfDay(session.Timestamp)
		if day.Before(minDay) {
			minDay = day
		}
		if day.After(maxDay) {
			maxDay = day
		}
	}
	return minDay, maxDay
}

func startOfDay(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Time{}
	}
	year, month, day := ts.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, ts.Location())
}

func weekdayIndex(ts time.Time) int {
	return (int(ts.Weekday()) + 6) % 7
}

func countBackwardStreak(activeDates map[time.Time]struct{}, end time.Time) int {
	streak := 0
	for day := end; ; day = day.AddDate(0, 0, -1) {
		if _, ok := activeDates[day]; !ok {
			return streak
		}
		streak++
	}
}

func countLongestStreak(activeDates map[time.Time]struct{}) int {
	if len(activeDates) == 0 {
		return 0
	}

	dates := make([]time.Time, 0, len(activeDates))
	for day := range activeDates {
		dates = append(dates, day)
	}
	slices.SortFunc(dates, func(left, right time.Time) int {
		switch {
		case left.Before(right):
			return -1
		case left.After(right):
			return 1
		default:
			return 0
		}
	})

	longest := 1
	current := 1
	for i := 1; i < len(dates); i++ {
		if sameDate(dates[i-1].AddDate(0, 0, 1), dates[i]) {
			current++
		} else {
			current = 1
		}
		if current > longest {
			longest = current
		}
	}
	return longest
}

func sameDate(left, right time.Time) bool {
	ly, lm, ld := left.Date()
	ry, rm, rd := right.Date()
	return ly == ry && lm == rm && ld == rd
}
