package stats

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

type statsKeyMap struct {
	NextTab key.Binding
	PrevTab key.Binding
	Range   key.Binding
	Filter  key.Binding
	Group   key.Binding
	Metric  key.Binding
	Help    key.Binding
	Close   key.Binding
}

var statsKeys = statsKeyMap{
	NextTab: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "next tab"),
	),
	PrevTab: key.NewBinding(
		key.WithKeys("ctrl+b"),
		key.WithHelp("ctrl+b", "prev tab"),
	),
	Range: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "range"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter"),
	),
	Group: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "group"),
	),
	Metric: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "metric"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Close: key.NewBinding(
		key.WithKeys("q", "esc"),
		key.WithHelp("q/esc", "close"),
	),
}

func (m statsModel) handleStatsKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if next, cmd, handled := m.handleStatsActionKey(msg); handled {
		return next, cmd
	}
	if next, handled := m.handleStatsLaneKey(msg); handled {
		return next.renderViewportContent(true), nil
	}
	if next, cmd, handled := m.handleStatsOpenSessionKey(msg); handled {
		return next, cmd
	}
	if next, handled := m.handleStatsJumpKey(msg); handled {
		return next, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m statsModel) handleStatsActionKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd, bool) {
	if next, cmd, handled := m.handleStatsMetricKey(msg); handled {
		return next, cmd, true
	}
	return m.handleStatsGlobalActionKey(msg)
}

func (m statsModel) handleStatsMetricKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd, bool) {
	if !key.Matches(msg, statsKeys.Metric) {
		return m, nil, false
	}
	return m.handleStatsMetricAction()
}

func (m statsModel) handleStatsGlobalActionKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, statsKeys.NextTab):
		m.tab = nextStatsTab(m.tab)
		return m.renderViewportContent(true), nil, true
	case key.Matches(msg, statsKeys.PrevTab):
		m.tab = prevStatsTab(m.tab)
		return m.renderViewportContent(true), nil, true
	case key.Matches(msg, statsKeys.Range):
		next, cmd := m.handleStatsRangeAction()
		return next, cmd, true
	case key.Matches(msg, statsKeys.Filter):
		return m.openFilterOverlay(), nil, true
	case key.Matches(msg, statsKeys.Group):
		return m.handleStatsGroupAction()
	case key.Matches(msg, statsKeys.Help):
		m.helpOpen = true
		return m, nil, true
	case key.Matches(msg, statsKeys.Close):
		return m, closeStatsCmd(), true
	default:
		return m, nil, false
	}
}

func (m statsModel) handleStatsRangeAction() (statsModel, tea.Cmd) {
	m.timeRange = nextStatsTimeRange(m.timeRange)
	m = m.recomputeSnapshot()
	return m.renderViewportContent(true), nil
}

func (m statsModel) handleStatsOpenSessionKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd, bool) {
	if msg.Code != tea.KeyEnter || !m.activeLaneSupportsOpen() {
		return m, nil, false
	}
	next, cmd := m.openHeavySession(m.overviewSessionCursor)
	return next, cmd, true
}

func (m statsModel) handleStatsJumpKey(msg tea.KeyPressMsg) (statsModel, bool) {
	switch {
	case msg.Text == "g" || msg.Code == tea.KeyHome:
		m.viewport.GotoTop()
		return m, true
	case msg.Text == "G" || msg.Code == tea.KeyEnd:
		m.viewport.GotoBottom()
		return m, true
	default:
		return m, false
	}
}
