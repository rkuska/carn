package app

import "slices"

const selectProviderWithVersionsPrompt = "Select a provider with v."

func (m statsModel) versionGroupingSupportedTab() bool {
	switch m.tab {
	case statsTabSessions, statsTabTools, statsTabCache:
		return true
	case statsTabOverview, statsTabActivity, statsTabPerformance:
		return false
	default:
		return false
	}
}

func (m statsModel) versionGroupingActive() bool {
	if !m.versionGroupingSupportedTab() {
		return false
	}
	return m.anyVersionGroupingActive()
}

func (m statsModel) setVersionGrouping(active bool) statsModel {
	if !m.versionGroupingSupportedTab() {
		return m
	}
	m.sessionsGrouped = active
	m.toolsGrouped = active
	m.cacheGrouped = active
	return m
}

func (m statsModel) anyVersionGroupingActive() bool {
	return m.sessionsGrouped || m.toolsGrouped || m.cacheGrouped
}

func (m statsModel) seedSingleProviderGroupScope() statsModel {
	if m.groupScope.hasProvider() {
		return m
	}
	providers := m.groupScopeProviderValues()
	if len(providers) != 1 {
		return m
	}
	m.groupScope.provider = providers[0]
	m.groupScope.versions = nil
	return m
}

func (m statsModel) groupedProviderTitle(title string) string {
	if !m.groupScope.hasProvider() {
		return title
	}
	return title + " (" + m.groupScope.provider.Label() + ")"
}

func (m statsModel) groupedVersionLabels() []string {
	if !m.groupScope.hasProvider() {
		return nil
	}
	if len(m.groupScope.versions) == 0 {
		return m.groupScopeVersionValues(m.groupScope.provider)
	}
	versions := make([]string, 0, len(m.groupScope.versions))
	for version := range m.groupScope.versions {
		versions = append(versions, version)
	}
	slices.Sort(versions)
	return versions
}
