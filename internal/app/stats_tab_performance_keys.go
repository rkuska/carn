package app

import tea "charm.land/bubbletea/v2"

func (m statsModel) handleStatsLaneKey(msg tea.KeyPressMsg) (statsModel, bool) {
	switch {
	case msg.Text == "h" || msg.Code == tea.KeyLeft:
		return m.moveStatsLane(-1), true
	case msg.Text == "l" || msg.Code == tea.KeyRight:
		return m.moveStatsLane(1), true
	default:
		return m, false
	}
}

func (m statsModel) handleStatsMetricAction() (statsModel, tea.Cmd, bool) {
	lane, _, ok := m.selectedStatsLane()
	if !ok || !lane.supportsMetric {
		return m, nil, false
	}

	switch lane.id { //nolint:exhaustive // only metric-enabled lanes handled
	case statsLaneOverviewTop:
		return m.nextOverviewSessionSelection().renderViewportContent(true), nil, true
	case statsLaneActivityDaily:
		m.activityMetric = nextActivityMetric(m.activityMetric)
		return m.renderViewportContent(true), nil, true
	case statsLaneCacheDaily:
		m.cacheMetric = nextCacheMetric(m.cacheMetric)
		return m.renderViewportContent(true), nil, true
	case statsLanePerformanceOutcome,
		statsLanePerformanceDiscipline,
		statsLanePerformanceEfficiency,
		statsLanePerformanceRobustness:
		if !m.performanceScopeAllowsScorecard() {
			return m, nil, false
		}
		return m.nextPerformanceMetric().renderViewportContent(true), nil, true
	default:
		return m, nil, false
	}
}
