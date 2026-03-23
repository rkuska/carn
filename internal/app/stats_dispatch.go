package app

import tea "charm.land/bubbletea/v2"

type openStatsMsg struct{}

type closeStatsMsg struct{}

func updateOpenStatsCmd() tea.Cmd {
	return func() tea.Msg {
		return openStatsMsg{}
	}
}

func updateCloseStatsCmd() tea.Cmd {
	return func() tea.Msg {
		return closeStatsMsg{}
	}
}

func (m appModel) updateStats(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case openStatsMsg:
		if m.state != viewBrowser {
			return m, nil
		}
		stats := newStatsModel(
			m.browser.mainConversations,
			m.store,
			m.width,
			m.height,
			m.browser.filter,
		)
		stats.ctx = m.ctx
		stats.archiveDir = m.cfg.ArchiveDir
		stats.glamourStyle = m.glamourStyle
		stats.timestampFormat = m.browser.timestampFormat
		stats.launcher = m.launcher
		m.stats = stats
		m.state = viewStats
		return m, nil
	case closeStatsMsg:
		m.state = viewBrowser
		return m, nil
	}

	if m.state != viewStats {
		return m, nil
	}

	var cmd tea.Cmd
	m.stats, cmd = m.stats.Update(msg)
	return m, cmd
}
