package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/internal/stats"
)

type toolMetricSessionTarget struct {
	conversation conv.Conversation
	session      conv.SessionMeta
}

func (m statsModel) toolMetricSessionTargets() []toolMetricSessionTarget {
	conversations := m.filteredConversations()
	targets := make([]toolMetricSessionTarget, 0, len(conversations))
	for _, conversation := range conversations {
		for _, session := range conversation.Sessions {
			targets = append(targets, toolMetricSessionTarget{
				conversation: conversation,
				session:      session,
			})
		}
	}
	return targets
}

func (m statsModel) toolMetricsSourceCacheKey() string {
	parts := []string{"tool-metrics"}
	parts = append(parts, filterBadges(m.filter.dimensions)...)
	return strings.Join(parts, "|")
}

func (m statsModel) maybeStartStatsBackgroundLoad() (statsModel, tea.Cmd) {
	switch m.tab {
	case statsTabOverview, statsTabActivity:
		return m, nil
	case statsTabSessions:
		return m.maybeStartClaudeTurnMetricsLoad()
	case statsTabTools:
		return m.maybeStartToolMetricsLoad()
	}
	return m, nil
}

func (m statsModel) maybeStartToolMetricsLoad() (statsModel, tea.Cmd) {
	if m.tab != statsTabTools || m.snapshot.Overview.SessionCount == 0 {
		return m, nil
	}

	key := m.toolMetricsSourceCacheKey()
	switch key {
	case m.toolMetricsSourceKey:
		m.snapshot.Tools = stats.ComputeToolsFromSessionMetrics(m.toolMetricSessions, m.timeRange)
		return m, nil
	case m.toolMetricsLoadingKey:
		return m, m.spinner.Tick
	default:
		m.toolMetricsLoadingKey = key
		return m, tea.Batch(
			loadToolMetricsCmd(m.ctx, m.store, m.toolMetricSessionTargets(), key),
			m.spinner.Tick,
		)
	}
}

func (m statsModel) toolMetricsLoading() bool {
	if m.tab != statsTabTools || m.snapshot.Overview.SessionCount == 0 {
		return false
	}
	key := m.toolMetricsSourceCacheKey()
	return key != "" && key == m.toolMetricsLoadingKey && key != m.toolMetricsSourceKey
}

func (m statsModel) toolMetricsLoadedForCurrentFilter() bool {
	key := m.toolMetricsSourceCacheKey()
	return key != "" && key == m.toolMetricsSourceKey
}

func (m statsModel) applyToolMetricsLoaded(msg toolMetricsLoadedMsg) statsModel {
	if msg.key != m.toolMetricsLoadingKey || msg.key != m.toolMetricsSourceCacheKey() {
		return m
	}

	m.toolMetricSessions = msg.sessions
	m.toolMetricsSourceKey = msg.key
	m.toolMetricsLoadingKey = ""
	m.snapshot.Tools = stats.ComputeToolsFromSessionMetrics(msg.sessions, m.timeRange)
	if m.tab == statsTabTools {
		m.viewport.SetContent(m.renderToolsTab(m.contentWidth()))
	}
	return m
}

func (m statsModel) statsBackgroundLoading() bool {
	return m.claudeTurnMetricsLoading() || m.toolMetricsLoading()
}
