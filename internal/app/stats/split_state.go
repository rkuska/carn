package stats

import (
	"slices"

	statspkg "github.com/rkuska/carn/internal/stats"
)

var splitDimensionOptions = []statspkg.SplitDimension{
	statspkg.SplitDimensionProvider,
	statspkg.SplitDimensionVersion,
	statspkg.SplitDimensionModel,
	statspkg.SplitDimensionProject,
}

func splitToFilterDim(dim statspkg.SplitDimension) (filterDimension, bool) {
	switch dim {
	case statspkg.SplitDimensionProvider:
		return filterDimProvider, true
	case statspkg.SplitDimensionVersion:
		return filterDimVersion, true
	case statspkg.SplitDimensionModel:
		return filterDimModel, true
	case statspkg.SplitDimensionProject:
		return filterDimProject, true
	case statspkg.SplitDimensionNone:
		return 0, false
	default:
		return 0, false
	}
}

func splitDimensionSupportsTab(tab statsTab) bool {
	switch tab {
	case statsTabSessions, statsTabTools, statsTabCache:
		return true
	case statsTabOverview, statsTabActivity, statsTabPerformance:
		return false
	default:
		return false
	}
}

func tabLabel(tab statsTab) string {
	switch tab {
	case statsTabOverview:
		return "Overview"
	case statsTabActivity:
		return "Activity"
	case statsTabSessions:
		return "Sessions"
	case statsTabTools:
		return "Tools"
	case statsTabCache:
		return "Cache"
	case statsTabPerformance:
		return "Performance"
	default:
		return ""
	}
}

func (m statsModel) splitActive() bool {
	return m.splitBy.IsActive() && splitDimensionSupportsTab(m.tab)
}

func (m statsModel) splitAllowed() map[string]bool {
	dim, ok := splitToFilterDim(m.splitBy)
	if !ok {
		return nil
	}
	selected := m.filter.Dimensions[dim].Selected
	if len(selected) == 0 {
		return nil
	}
	allowed := make(map[string]bool, len(selected))
	for value := range selected {
		allowed[value] = true
	}
	return allowed
}

func (m statsModel) splitKeys() []string {
	if !m.splitBy.IsActive() {
		return nil
	}
	allowed := m.splitAllowed()
	if len(allowed) > 0 {
		keys := make([]string, 0, len(allowed))
		for key := range allowed {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		return keys
	}
	return slices.Clone(m.splitValues)
}

func (m statsModel) extractSplitValues() []string {
	if !m.splitBy.IsActive() {
		return nil
	}
	values := make(map[string]bool)
	for _, session := range m.statsSessions {
		key := m.splitBy.SessionKey(session)
		if key == "" {
			continue
		}
		values[key] = true
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func (m statsModel) splitTitle(base string) string {
	if !m.splitActive() {
		return base
	}
	return base + " (by " + m.splitBy.Label() + ")"
}

func (m statsModel) splitTurnSeries(mode statspkg.StatisticMode) []statspkg.SplitTurnSeries {
	if !m.splitBy.IsActive() || !m.splitBy.SupportsTurnMetrics() {
		return nil
	}
	return statspkg.ComputeTurnTokenMetricsBySplit(
		m.statsTurnMetrics,
		m.timeRange,
		m.splitBy,
		m.splitAllowed(),
		mode,
	)
}

func (m statsModel) computeSplitTools() statspkg.ToolsBySplit {
	return statspkg.ComputeToolsBySplit(
		m.statsSessions,
		m.timeRange,
		m.splitBy,
		m.splitAllowed(),
	)
}

func (m statsModel) computeSplitCache() statspkg.CacheBySplit {
	return statspkg.ComputeCacheBySplit(
		m.statsSessions,
		m.timeRange,
		m.splitBy,
		m.splitAllowed(),
	)
}
