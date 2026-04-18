package stats

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

const cacheFirstTurnMinSessions = 3

// ComputeCacheFirstTurnByVersion returns the share of Claude sessions whose
// first turn came in with zero cache reads, grouped by normalized version.
// Groups with fewer than cacheFirstTurnMinSessions sessions are dropped.
func ComputeCacheFirstTurnByVersion(series []conv.SessionTurnMetrics) []CacheFirstTurnVersionStat {
	if len(series) == 0 {
		return nil
	}

	byVersion := make(map[string][]int)
	for _, session := range series {
		if session.Provider != conv.ProviderClaude {
			continue
		}
		if len(session.Turns) == 0 {
			continue
		}
		key := NormalizeVersionLabel(session.Version)
		byVersion[key] = append(byVersion[key], session.Turns[0].CacheReadTokens)
	}
	if len(byVersion) == 0 {
		return nil
	}

	stats := make([]CacheFirstTurnVersionStat, 0, len(byVersion))
	for version, reads := range byVersion {
		if len(reads) < cacheFirstTurnMinSessions {
			continue
		}
		sort.Ints(reads)
		zeros := 0
		for _, read := range reads {
			if read != 0 {
				break
			}
			zeros++
		}
		stats = append(stats, CacheFirstTurnVersionStat{
			Version:         version,
			SessionCount:    len(reads),
			ZeroCount:       zeros,
			ZeroReadRate:    float64(zeros) / float64(len(reads)),
			MedianFirstRead: medianSorted(reads),
		})
	}
	slices.SortFunc(stats, compareCacheFirstTurnVersionStat)
	return stats
}

func medianSorted(sorted []int) int {
	if len(sorted) == 0 {
		return 0
	}
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func compareCacheFirstTurnVersionStat(left, right CacheFirstTurnVersionStat) int {
	return compareVersionLabel(left.Version, right.Version)
}

// compareVersionLabel orders semantic versions numerically when both sides
// parse cleanly. Unknown / unparsable labels fall back to string order and
// sort after parsed versions so ordinary releases appear first.
func compareVersionLabel(left, right string) int {
	leftParts, leftOK := parseVersionLabel(left)
	rightParts, rightOK := parseVersionLabel(right)
	switch {
	case leftOK && rightOK:
		return compareVersionParts(leftParts, rightParts)
	case leftOK:
		return -1
	case rightOK:
		return 1
	default:
		return strings.Compare(left, right)
	}
}

func parseVersionLabel(label string) ([]int, bool) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(label), "v")
	if trimmed == "" {
		return nil, false
	}
	segments := strings.Split(trimmed, ".")
	parts := make([]int, 0, len(segments))
	for _, segment := range segments {
		n, err := strconv.Atoi(segment)
		if err != nil {
			return nil, false
		}
		parts = append(parts, n)
	}
	return parts, len(parts) > 0
}

func compareVersionParts(left, right []int) int {
	size := max(len(left), len(right))
	for i := range size {
		leftPart := 0
		if i < len(left) {
			leftPart = left[i]
		}
		rightPart := 0
		if i < len(right) {
			rightPart = right[i]
		}
		if leftPart != rightPart {
			if leftPart < rightPart {
				return -1
			}
			return 1
		}
	}
	return 0
}
