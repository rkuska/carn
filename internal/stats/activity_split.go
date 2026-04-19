package stats

import (
	"slices"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type splitActivityAccumulator struct {
	sessions []DailyValue
	messages []DailyValue
	tokens   []DailyValue
}

func ComputeActivityBySplit(
	sessions []conv.SessionMeta,
	activityBuckets []conv.ActivityBucketRow,
	timeRange TimeRange,
	dim SplitDimension,
	allowed map[string]bool,
) ActivityBySplit {
	if !dim.IsActive() {
		return ActivityBySplit{}
	}

	location := activityLocation(sessions, timeRange)
	start, end, ok := resolveActivityBoundsFromBuckets(
		sessions,
		activityBuckets,
		timeRange,
		location,
	)
	if !ok {
		return ActivityBySplit{}
	}

	start = startOfDayInLocation(start, location)
	end = startOfDayInLocation(end, location)
	startDayKey := activityDayKey(start)
	dayCount := activityDayKey(end) - startDayKey + 1
	if dayCount <= 0 {
		return ActivityBySplit{}
	}

	if len(activityBuckets) > 0 {
		return computeActivityBySplitFromBuckets(
			activityBuckets,
			dim,
			allowed,
			location,
			start,
			end,
			startDayKey,
			dayCount,
		)
	}

	return computeActivityBySplitFromSessions(
		sessions,
		timeRange,
		dim,
		allowed,
		location,
		start,
		startDayKey,
		dayCount,
	)
}

func computeActivityBySplitFromBuckets(
	rows []conv.ActivityBucketRow,
	dim SplitDimension,
	allowed map[string]bool,
	location *time.Location,
	start, end time.Time,
	startDayKey int,
	dayCount int,
) ActivityBySplit {
	byKey := make(map[string]*splitActivityAccumulator)
	for _, row := range rows {
		day := activityBucketDay(row.BucketStart, location)
		if day.IsZero() || day.Before(start) || day.After(end) {
			continue
		}
		key, ok := matchActivityBucketSplitScope(row, dim, allowed)
		if !ok {
			continue
		}

		acc := splitActivityAccumulatorForKey(byKey, key, start, dayCount)
		accumulateSplitActivityBucketDay(
			acc,
			activityDayKey(day)-startDayKey,
			row,
		)
	}
	return buildActivityBySplit(byKey)
}

func computeActivityBySplitFromSessions(
	sessions []conv.SessionMeta,
	timeRange TimeRange,
	dim SplitDimension,
	allowed map[string]bool,
	location *time.Location,
	start time.Time,
	startDayKey int,
	dayCount int,
) ActivityBySplit {
	byKey := make(map[string]*splitActivityAccumulator)
	for _, session := range sessions {
		key, ok := matchSessionSplitScope(session, timeRange, dim, allowed)
		if !ok {
			continue
		}

		day := startOfDayInLocation(
			normalizeActivityTime(session.Timestamp, location),
			location,
		)
		dayIndex := activityDayKey(day) - startDayKey
		if dayIndex < 0 || dayIndex >= dayCount {
			continue
		}

		acc := splitActivityAccumulatorForKey(byKey, key, start, dayCount)
		acc.sessions[dayIndex].Value++
		acc.sessions[dayIndex].HasValue = true

		messageCount := sessionMessageCount(session)
		if messageCount > 0 {
			acc.messages[dayIndex].Value += float64(messageCount)
			acc.messages[dayIndex].HasValue = true
		}

		totalTokens := session.TotalUsage.TotalTokens()
		if totalTokens > 0 {
			acc.tokens[dayIndex].Value += float64(totalTokens)
			acc.tokens[dayIndex].HasValue = true
		}
	}
	return buildActivityBySplit(byKey)
}

func splitActivityAccumulatorForKey(
	byKey map[string]*splitActivityAccumulator,
	key string,
	start time.Time,
	dayCount int,
) *splitActivityAccumulator {
	if acc := byKey[key]; acc != nil {
		return acc
	}

	acc := &splitActivityAccumulator{
		sessions: makeSplitDailyValues(start, dayCount),
		messages: makeSplitDailyValues(start, dayCount),
		tokens:   makeSplitDailyValues(start, dayCount),
	}
	byKey[key] = acc
	return acc
}

func makeSplitDailyValues(start time.Time, dayCount int) []DailyValue {
	values := make([]DailyValue, dayCount)
	for i := range dayCount {
		values[i].Date = start.AddDate(0, 0, i)
	}
	return values
}

func accumulateSplitActivityBucketDay(
	acc *splitActivityAccumulator,
	dayIndex int,
	row conv.ActivityBucketRow,
) {
	if dayIndex < 0 || dayIndex >= len(acc.sessions) {
		return
	}
	if row.SessionCount > 0 {
		acc.sessions[dayIndex].Value += float64(row.SessionCount)
		acc.sessions[dayIndex].HasValue = true
	}
	if row.MessageCount > 0 {
		acc.messages[dayIndex].Value += float64(row.MessageCount)
		acc.messages[dayIndex].HasValue = true
	}
	if totalTokens := activityBucketTotalTokens(row); totalTokens > 0 {
		acc.tokens[dayIndex].Value += float64(totalTokens)
		acc.tokens[dayIndex].HasValue = true
	}
}

func matchActivityBucketSplitScope(
	row conv.ActivityBucketRow,
	dim SplitDimension,
	allowed map[string]bool,
) (string, bool) {
	key := activityBucketSplitKey(row, dim)
	if key == "" {
		return "", false
	}
	if len(allowed) > 0 && !allowed[key] {
		return "", false
	}
	return key, true
}

func activityBucketSplitKey(row conv.ActivityBucketRow, dim SplitDimension) string {
	switch dim {
	case SplitDimensionProvider:
		return providerLabelOrUnknown(conv.Provider(row.Provider))
	case SplitDimensionVersion:
		return NormalizeVersionLabel(row.Version)
	case SplitDimensionModel:
		return labelOrUnknown(row.Model)
	case SplitDimensionProject:
		return labelOrUnknown(row.Project)
	case SplitDimensionNone:
		return ""
	default:
		return ""
	}
}

func buildActivityBySplit(byKey map[string]*splitActivityAccumulator) ActivityBySplit {
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	result := ActivityBySplit{
		DailySessions: make([]SplitDailyValueSeries, 0, len(keys)),
		DailyMessages: make([]SplitDailyValueSeries, 0, len(keys)),
		DailyTokens:   make([]SplitDailyValueSeries, 0, len(keys)),
	}
	for _, key := range keys {
		acc := byKey[key]
		result.DailySessions = append(result.DailySessions, SplitDailyValueSeries{
			Key:    key,
			Values: acc.sessions,
		})
		result.DailyMessages = append(result.DailyMessages, SplitDailyValueSeries{
			Key:    key,
			Values: acc.messages,
		})
		result.DailyTokens = append(result.DailyTokens, SplitDailyValueSeries{
			Key:    key,
			Values: acc.tokens,
		})
	}
	return result
}
