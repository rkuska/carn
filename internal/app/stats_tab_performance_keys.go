package app

import tea "charm.land/bubbletea/v2"

func (m statsModel) handlePerformanceSelectionKey(msg tea.KeyPressMsg) (statsModel, bool) {
	if m.tab != statsTabPerformance || !m.performanceScopeAllowsScorecard() {
		return m, false
	}

	switch {
	case msg.Text == "h" || msg.Code == tea.KeyLeft:
		return m.movePerformanceLane(-1), true
	case msg.Text == "l" || msg.Code == tea.KeyRight:
		return m.movePerformanceLane(1), true
	default:
		return m, false
	}
}

func (m statsModel) handleStatsMetricAction() (statsModel, tea.Cmd, bool) {
	if m.tab == statsTabActivity {
		m.activityMetric = nextActivityMetric(m.activityMetric)
		return m.renderViewportContent(true), nil, true
	}
	if m.tab == statsTabPerformance {
		if !m.performanceScopeAllowsScorecard() {
			return m, nil, true
		}
		return m.nextPerformanceMetric().renderViewportContent(true), nil, true
	}
	return m, nil, true
}
