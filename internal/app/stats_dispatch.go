package app

import (
	tea "charm.land/bubbletea/v2"

	appbrowser "github.com/rkuska/carn/internal/app/browser"
	appstats "github.com/rkuska/carn/internal/app/stats"
)

func (m appModel) updateStats(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case appbrowser.OpenStatsRequestedMsg:
		if m.state != viewBrowser {
			return m, nil
		}
		m.stats = appstats.NewModel(
			m.ctx,
			m.cfg.ArchiveDir,
			m.browser.AllConversations(),
			m.store,
			m.width,
			m.height,
			m.browser.FilterState(),
		)
		m.state = viewStats
		return m, nil
	case appstats.CloseRequestedMsg:
		m.state = viewBrowser
		return m, nil
	case appstats.OpenSessionRequestedMsg:
		return m, loadStatsSessionCmd(m.ctx, m.store, msg.Conversation, msg.SessionMeta)
	case statsSessionLoadedMsg:
		m.browser = m.browser.OpenLoadedSession(msg.conversation, msg.session)
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
