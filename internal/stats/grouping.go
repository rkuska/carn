package stats

import (
	"slices"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

const UnknownVersionLabel = "unknown"

func NormalizeVersionLabel(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return UnknownVersionLabel
	}
	return version
}

func ComputeTurnTokenMetricsByVersion(
	sessions []SessionTurnMetrics,
	timeRange TimeRange,
	provider conv.Provider,
	versions map[string]bool,
) []VersionTurnSeries {
	if len(sessions) == 0 {
		return nil
	}

	grouped := groupTurnMetricsByVersion(sessions, provider, versions)
	if len(grouped) == 0 {
		return nil
	}

	return buildVersionTurnSeries(grouped, timeRange)
}

func groupTurnMetricsByVersion(
	sessions []SessionTurnMetrics,
	provider conv.Provider,
	versions map[string]bool,
) map[string][]SessionTurnMetrics {
	grouped := make(map[string][]SessionTurnMetrics)
	for _, session := range sessions {
		versionLabel, ok := matchVersionScope(session, provider, versions)
		if !ok {
			continue
		}
		grouped[versionLabel] = append(grouped[versionLabel], session)
	}
	return grouped
}

func matchVersionScope(
	session SessionTurnMetrics,
	provider conv.Provider,
	versions map[string]bool,
) (string, bool) {
	if session.Provider != provider {
		return "", false
	}
	versionLabel := NormalizeVersionLabel(session.Version)
	if len(versions) > 0 && !versions[versionLabel] {
		return "", false
	}
	return versionLabel, true
}

func buildVersionTurnSeries(
	grouped map[string][]SessionTurnMetrics,
	timeRange TimeRange,
) []VersionTurnSeries {
	items := make([]VersionTurnSeries, 0, len(grouped))
	for versionLabel, versionSessions := range grouped {
		metrics := ComputeTurnTokenMetricsForRange(versionSessions, timeRange)
		if len(metrics) == 0 {
			continue
		}
		items = append(items, VersionTurnSeries{
			Version: versionLabel,
			Metrics: metrics,
		})
	}
	slices.SortFunc(items, compareVersionTurnSeries)
	return items
}

func compareVersionTurnSeries(left, right VersionTurnSeries) int {
	return strings.Compare(left.Version, right.Version)
}
