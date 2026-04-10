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

	if lane.id == statsLaneOverviewTop {
		return m.nextOverviewSessionSelection().renderViewportContent(true), nil, true
	}
	if lane.id == statsLaneActivityDaily {
		m.activityMetric = nextActivityMetric(m.activityMetric)
		return m.renderViewportContent(true), nil, true
	}
	if lane.id == statsLanePerformanceOutcome ||
		lane.id == statsLanePerformanceDiscipline ||
		lane.id == statsLanePerformanceEfficiency ||
		lane.id == statsLanePerformanceRobustness {
		if !m.performanceScopeAllowsScorecard() {
			return m, nil, false
		}
		return m.nextPerformanceMetric().renderViewportContent(true), nil, true
	}
	return m, nil, false
}
