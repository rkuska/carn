package app

import tea "charm.land/bubbletea/v2"

func (m statsModel) maybeStartStatsBackgroundLoad() (statsModel, tea.Cmd) {
	if m.tab != statsTabSessions {
		return m, nil
	}
	return m.maybeStartClaudeTurnMetricsLoad()
}

func (m statsModel) statsBackgroundLoading() bool {
	return m.claudeTurnMetricsLoading()
}
