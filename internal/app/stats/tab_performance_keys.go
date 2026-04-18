package stats

import (
	tea "charm.land/bubbletea/v2"

	statspkg "github.com/rkuska/carn/internal/stats"
)

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
	if !ok || !m.activeLaneSupportsMetric() {
		return m, nil, false
	}
	return m.handleStatsLaneMetricAction(lane)
}

func (m statsModel) handleStatsLaneMetricAction(lane statsLane) (statsModel, tea.Cmd, bool) {
	if next, cmd, handled := m.handleSharedLaneMetricAction(lane.id); handled {
		return next, cmd, true
	}
	if next, cmd, handled := m.handleSessionLaneMetricAction(lane.id); handled {
		return next, cmd, true
	}
	return m.handlePerformanceLaneMetricAction(lane.id)
}

func (m statsModel) handleSharedLaneMetricAction(id statsLaneID) (statsModel, tea.Cmd, bool) {
	switch id { //nolint:exhaustive // only shared metric lanes handled
	case statsLaneOverviewTop:
		return m.nextOverviewSessionSelection().renderViewportContent(true), nil, true
	case statsLaneActivityDaily:
		m.activityMetric = nextActivityMetric(m.activityMetric)
		return m.renderViewportContent(true), nil, true
	case statsLaneCacheDaily:
		m.cacheMetric = nextCacheMetric(m.cacheMetric)
		return m.renderViewportContent(true), nil, true
	default:
		return m, nil, false
	}
}

func (m statsModel) handleSessionLaneMetricAction(id statsLaneID) (statsModel, tea.Cmd, bool) {
	switch id { //nolint:exhaustive // only session metric lanes handled
	case statsLaneSessionsContext:
		m.sessionsPromptMode = statspkg.NextStatisticMode(m.sessionsPromptMode, statisticModesWithoutTotal)
		return m.renderViewportContent(true), nil, true
	case statsLaneSessionsTurnCost:
		m.sessionsTurnCostMode = statspkg.NextStatisticMode(m.sessionsTurnCostMode, statisticModesWithoutTotal)
		return m.renderViewportContent(true), nil, true
	default:
		return m, nil, false
	}
}

func (m statsModel) handlePerformanceLaneMetricAction(id statsLaneID) (statsModel, tea.Cmd, bool) {
	switch id { //nolint:exhaustive // only performance metric lanes handled
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
