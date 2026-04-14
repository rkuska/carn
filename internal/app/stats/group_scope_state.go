package stats

import (
	"maps"
	"slices"

	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

type statsGroupScopeState struct {
	active         bool
	cursor         int
	expanded       int
	expandedCursor int
	provider       conv.Provider
	versions       map[string]bool
}

const (
	statsGroupScopeProvider = iota
	statsGroupScopeVersion
	statsGroupScopeCount
)

func (s statsGroupScopeState) hasProvider() bool {
	return s.provider != ""
}

func (s statsGroupScopeState) clear() statsGroupScopeState {
	s.active = false
	s.expanded = -1
	s.expandedCursor = 0
	s.provider = ""
	s.versions = nil
	return s
}

func (m statsModel) sessionsInRange() []conv.SessionMeta {
	return statspkg.FilterByTimeRange(m.statsSessions, m.timeRange)
}

func (m statsModel) groupScopeProviderValues() []conv.Provider {
	values := make(map[conv.Provider]bool)
	for _, session := range m.sessionsInRange() {
		if session.Provider == "" {
			continue
		}
		values[session.Provider] = true
	}
	providers := make([]conv.Provider, 0, len(values))
	for provider := range values {
		providers = append(providers, provider)
	}
	slices.SortFunc(providers, func(left, right conv.Provider) int {
		if left.Label() < right.Label() {
			return -1
		}
		if left.Label() > right.Label() {
			return 1
		}
		return 0
	})
	return providers
}

func (m statsModel) groupScopeVersionValues(provider conv.Provider) []string {
	if provider == "" {
		return nil
	}
	values := make(map[string]bool)
	for _, session := range m.sessionsInRange() {
		if session.Provider != provider {
			continue
		}
		values[statspkg.NormalizeVersionLabel(session.Version)] = true
	}
	versions := make([]string, 0, len(values))
	for version := range values {
		versions = append(versions, version)
	}
	slices.Sort(versions)
	return versions
}

func (m statsModel) groupedTurnSeries() []statspkg.VersionTurnSeries {
	if !m.groupScope.hasProvider() {
		return nil
	}
	return statspkg.ComputeTurnTokenMetricsByVersion(
		m.statsTurnMetrics,
		m.timeRange,
		m.groupScope.provider,
		m.groupScope.versions,
	)
}

func cloneGroupScopeVersions(values map[string]bool) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	return maps.Clone(values)
}
