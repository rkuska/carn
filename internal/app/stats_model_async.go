package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/internal/stats"
)

const statsPerformanceSequenceLoadingLabel = "Loading transcript sequence metrics..."

type claudeTurnMetricSessionTarget struct {
	conversation conv.Conversation
	session      conv.SessionMeta
}

func (m statsModel) claudeTurnMetricSessionTargets() []claudeTurnMetricSessionTarget {
	conversations := m.filteredConversations()
	targets := make([]claudeTurnMetricSessionTarget, 0, len(conversations))
	for _, conversation := range conversations {
		for _, session := range conversation.Sessions {
			targets = append(targets, claudeTurnMetricSessionTarget{
				conversation: conversation,
				session:      session,
			})
		}
	}
	return targets
}

func (m statsModel) claudeTurnMetricsSourceCacheKey() string {
	parts := []string{"turn-metrics"}
	parts = append(parts, filterBadges(m.filter.dimensions)...)
	return strings.Join(parts, "|")
}

func (m statsModel) performanceSequenceSourceCacheKey() string {
	parts := []string{"performance-sequence"}
	parts = append(parts, filterBadges(m.filter.dimensions)...)
	return strings.Join(parts, "|")
}

func (m statsModel) renderViewportContentAndMaybeLoad(resetScroll bool) (statsModel, tea.Cmd) {
	m = m.renderViewportContent(resetScroll)
	return m.maybeStartStatsBackgroundLoad()
}

func (m statsModel) applyFilterChangeAndMaybeLoad() (statsModel, tea.Cmd) {
	m = m.applyFilterChange()
	return m.maybeStartStatsBackgroundLoad()
}

func (m statsModel) maybeStartClaudeTurnMetricsLoad() (statsModel, tea.Cmd) {
	if m.tab != statsTabSessions || m.snapshot.Overview.SessionCount == 0 {
		return m, nil
	}

	key := m.claudeTurnMetricsSourceCacheKey()
	switch key {
	case m.claudeTurnMetricsSourceKey:
		return m, nil
	case m.claudeTurnMetricsLoadingKey:
		return m, m.spinner.Tick
	default:
		m.claudeTurnMetricsLoadingKey = key
		m = m.setViewportContent(m.renderSessionsTab(m.contentWidth()))
		return m, tea.Batch(
			loadClaudeTurnMetricsCmd(m.ctx, m.store, m.claudeTurnMetricSessionTargets(), key),
			m.spinner.Tick,
		)
	}
}

func (m statsModel) maybeStartPerformanceSequenceLoad() (statsModel, tea.Cmd) {
	if m.tab != statsTabPerformance || m.snapshot.Overview.SessionCount == 0 || !m.performanceScopeAllowsScorecard() {
		return m, nil
	}

	key := m.performanceSequenceSourceCacheKey()
	switch key {
	case m.performanceSequenceSourceKey:
		return m, nil
	case m.performanceSequenceLoadingKey:
		return m, m.spinner.Tick
	default:
		m.performanceSequenceLoadingKey = key
		m = m.setViewportContent(m.renderPerformanceTab(m.contentWidth()))
		return m, tea.Batch(
			loadPerformanceSequenceCmd(m.ctx, m.store, m.claudeTurnMetricSessionTargets(), key),
			m.spinner.Tick,
		)
	}
}

func (m statsModel) claudeTurnMetricsLoading() bool {
	if m.tab != statsTabSessions || m.snapshot.Overview.SessionCount == 0 {
		return false
	}
	key := m.claudeTurnMetricsSourceCacheKey()
	return key != "" && key == m.claudeTurnMetricsLoadingKey && key != m.claudeTurnMetricsSourceKey
}

func (m statsModel) performanceSequenceLoading() bool {
	if m.tab != statsTabPerformance || m.snapshot.Overview.SessionCount == 0 || !m.performanceScopeAllowsScorecard() {
		return false
	}
	key := m.performanceSequenceSourceCacheKey()
	return key != "" && key == m.performanceSequenceLoadingKey && key != m.performanceSequenceSourceKey
}

func (m statsModel) applyClaudeTurnMetricsLoaded(msg claudeTurnMetricsLoadedMsg) statsModel {
	if msg.key != m.claudeTurnMetricsLoadingKey || msg.key != m.claudeTurnMetricsSourceCacheKey() {
		return m
	}

	m.claudeTurnMetricSessions = msg.sessions
	m.claudeTurnMetricsSourceKey = msg.key
	m.claudeTurnMetrics = stats.ComputeTurnTokenMetricsForRange(msg.sessions, m.timeRange)
	m.claudeTurnMetricsLoadingKey = ""
	m.snapshot.Sessions.ClaudeTurnMetrics = m.claudeTurnMetrics
	if m.tab == statsTabSessions {
		m = m.setViewportContent(m.renderSessionsTab(m.contentWidth()))
	}
	return m
}

func (m statsModel) applyPerformanceSequenceLoaded(msg performanceSequenceLoadedMsg) statsModel {
	if msg.key != m.performanceSequenceLoadingKey || msg.key != m.performanceSequenceSourceCacheKey() {
		return m
	}

	m.performanceSequenceSessions = msg.sessions
	m.performanceSequenceSourceKey = msg.key
	m.performanceSequenceLoadingKey = ""
	m.snapshot.Performance = stats.ComputePerformance(
		m.filteredConversations(),
		m.timeRange,
		m.performanceSequenceSessions,
	)
	m = m.normalizePerformanceSelection()
	if m.tab == statsTabPerformance {
		m = m.setViewportContent(m.renderPerformanceTab(m.contentWidth()))
	}
	return m
}

func (m statsModel) handleSpinnerTick(msg spinner.TickMsg) (statsModel, tea.Cmd) {
	if !m.statsBackgroundLoading() {
		return m, nil
	}

	previousContent := m.renderedTabContent
	previousClaudeLine := m.claudeTurnMetricsLoadingLine()
	previousPerformanceLine := m.performanceSequenceLoadingLine()

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)

	switch {
	case m.tab == statsTabSessions && m.claudeTurnMetricsLoading():
		m = m.setViewportContent(replaceStatsLoadingLine(
			previousContent,
			previousClaudeLine,
			m.claudeTurnMetricsLoadingLine(),
		))
	case m.tab == statsTabPerformance && m.performanceSequenceLoading():
		m = m.setViewportContent(replaceStatsLoadingLine(
			previousContent,
			previousPerformanceLine,
			m.performanceSequenceLoadingLine(),
		))
	}
	return m, cmd
}

func (m statsModel) claudeTurnMetricsLoadingLine() string {
	return fmt.Sprintf("%s %s", m.spinner.View(), statsClaudeTurnChartsLoadingLabel)
}

func (m statsModel) performanceSequenceLoadingLine() string {
	return fmt.Sprintf("%s %s", m.spinner.View(), statsPerformanceSequenceLoadingLabel)
}

func replaceStatsLoadingLine(content, previousLine, nextLine string) string {
	if content == "" || previousLine == nextLine {
		return content
	}
	return strings.ReplaceAll(content, previousLine, nextLine)
}
