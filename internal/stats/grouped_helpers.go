package stats

import (
	"slices"

	conv "github.com/rkuska/carn/internal/conversation"
)

func matchSessionSplitScope(
	session conv.SessionMeta,
	timeRange TimeRange,
	dim SplitDimension,
	allowed map[string]bool,
) (string, bool) {
	if !timeRangeContains(timeRange, session.Timestamp) {
		return "", false
	}
	key := dim.SessionKey(session)
	if key == "" {
		return "", false
	}
	if len(allowed) > 0 && !allowed[key] {
		return "", false
	}
	return key, true
}

func sortSplitValues(values map[string]int) []SplitValue {
	items := make([]SplitValue, 0, len(values))
	for key, value := range values {
		if value <= 0 {
			continue
		}
		items = append(items, SplitValue{Key: key, Value: value})
	}
	slices.SortFunc(items, func(left, right SplitValue) int {
		switch {
		case left.Key < right.Key:
			return -1
		case left.Key > right.Key:
			return 1
		default:
			return right.Value - left.Value
		}
	})
	return items
}

func splitNamedStatFromCounts(name string, total int, values map[string]int) SplitNamedStat {
	return SplitNamedStat{
		Name:   name,
		Total:  total,
		Splits: sortSplitValues(values),
	}
}
