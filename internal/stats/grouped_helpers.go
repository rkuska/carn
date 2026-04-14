package stats

import (
	"slices"

	conv "github.com/rkuska/carn/internal/conversation"
)

func matchSessionVersionScope(
	session conv.SessionMeta,
	timeRange TimeRange,
	provider conv.Provider,
	versions map[string]bool,
) (string, bool) {
	if session.Provider != provider || !timeRangeContains(timeRange, session.Timestamp) {
		return "", false
	}
	versionLabel := NormalizeVersionLabel(session.Version)
	if len(versions) > 0 && !versions[versionLabel] {
		return "", false
	}
	return versionLabel, true
}

func sortVersionValues(values map[string]int) []VersionValue {
	items := make([]VersionValue, 0, len(values))
	for version, value := range values {
		if value <= 0 {
			continue
		}
		items = append(items, VersionValue{Version: version, Value: value})
	}
	slices.SortFunc(items, func(left, right VersionValue) int {
		switch {
		case left.Version < right.Version:
			return -1
		case left.Version > right.Version:
			return 1
		default:
			return right.Value - left.Value
		}
	})
	return items
}

func groupedNamedStatFromCounts(name string, total int, versions map[string]int) GroupedNamedStat {
	return GroupedNamedStat{
		Name:     name,
		Total:    total,
		Versions: sortVersionValues(versions),
	}
}
