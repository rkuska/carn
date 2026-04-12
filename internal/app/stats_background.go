package app

import tea "charm.land/bubbletea/v2"

func (m statsModel) maybeStartStatsBackgroundLoad() (statsModel, tea.Cmd) {
	switch m.tab {
	case statsTabOverview, statsTabActivity, statsTabTools, statsTabCache:
		return m, nil
	case statsTabSessions:
		return m.maybeStartClaudeTurnMetricsLoad()
	case statsTabPerformance:
		return m.maybeStartPerformanceSequenceLoad()
	}
	return m, nil
}

func (m statsModel) statsBackgroundLoading() bool {
	return m.claudeTurnMetricsLoading() || m.performanceSequenceLoading()
}
