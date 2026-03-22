package app

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/internal/stats"
)

type statsTab int

const (
	statsTabOverview statsTab = iota
	statsTabActivity
	statsTabSessions
	statsTabTools
)

type activityMetric int

const (
	metricSessions activityMetric = iota
	metricMessages
	metricTokens
)

type statsModel struct {
	conversations               []conv.Conversation
	store                       browserStore
	ctx                         context.Context
	archiveDir                  string
	tab                         statsTab
	timeRange                   stats.TimeRange
	snapshot                    stats.Snapshot
	claudeTurnMetricSessions    []stats.SessionTurnMetrics
	claudeTurnMetricsSourceKey  string
	claudeTurnMetrics           []stats.PositionTokenMetrics
	claudeTurnMetricsLoadingKey string
	toolMetricSessions          []stats.SessionToolMetrics
	toolMetricsSourceKey        string
	toolMetricsLoadingKey       string
	filter                      browserFilterState
	helpOpen                    bool
	viewport                    viewport.Model
	spinner                     spinner.Model
	width, height               int
	activityMetric              activityMetric
}

const (
	statsRangeLabel7d  = "7d"
	statsRangeLabel30d = "30d"
	statsRangeLabel90d = "90d"
	statsRangeLabelAll = "All"
)

var statsNow = time.Now

func newStatsModel(
	conversations []conv.Conversation,
	store browserStore,
	width, height int,
	filter browserFilterState,
) statsModel {
	vp := viewport.New()
	vp.KeyMap.PageDown.SetEnabled(false)
	vp.KeyMap.PageUp.SetEnabled(false)
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	model := statsModel{
		conversations: conversations,
		store:         store,
		ctx:           context.Background(),
		filter:        copyBrowserFilterState(filter),
		viewport:      vp,
		spinner:       s,
		width:         width,
		height:        height,
		tab:           statsTabOverview,
		timeRange:     statsRange30d(),
	}
	model.filter.values = extractFilterValues(conversations)
	model = model.setSize(width, height)
	model = model.applyFilterChange()
	return model
}

func (m statsModel) Update(msg tea.Msg) (statsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.setSize(msg.Width, msg.Height), nil
	case closeStatsMsg:
		return m, nil
	case claudeTurnMetricsLoadedMsg:
		return m.applyClaudeTurnMetricsLoaded(msg), nil
	case toolMetricsLoadedMsg:
		return m.applyToolMetricsLoaded(msg), nil
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if m.helpOpen {
			return m.handleHelpKey(keyMsg)
		}
		if m.filter.active {
			return m.handleFilterKey(keyMsg)
		}
		return m.handleStatsKey(keyMsg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m statsModel) handleHelpKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if key.Matches(msg, statsKeys.Help) || key.Matches(msg, statsKeys.Close) {
		m.helpOpen = false
		return m, nil
	}
	return m, nil
}

func (m statsModel) setSize(width, height int) statsModel {
	m.width = width
	m.height = height
	m.viewport.SetWidth(m.contentWidth())
	m.viewport.SetHeight(m.contentHeight())
	return m.renderViewportContent(false)
}

func (m statsModel) applyFilterChange() statsModel {
	m.snapshot = stats.ComputeSnapshot(m.filteredSessions(), m.timeRange)
	m.claudeTurnMetricSessions = nil
	m.claudeTurnMetricsSourceKey = ""
	m.claudeTurnMetrics = nil
	m.claudeTurnMetricsLoadingKey = ""
	m.toolMetricSessions = nil
	m.toolMetricsSourceKey = ""
	m.toolMetricsLoadingKey = ""
	return m.renderViewportContent(true)
}

func (m statsModel) renderViewportContent(resetScroll bool) statsModel {
	content := m.renderActiveTab()
	m.viewport.SetWidth(m.contentWidth())
	m.viewport.SetHeight(m.contentHeight())
	m.viewport.SetContent(content)
	if resetScroll {
		m.viewport.GotoTop()
	}
	return m
}

func (m statsModel) renderActiveTab() string {
	if m.snapshot.Overview.SessionCount == 0 {
		return lipgloss.Place(
			m.contentWidth(),
			m.contentHeight(),
			lipgloss.Center,
			lipgloss.Center,
			"No sessions match",
		)
	}

	switch m.tab {
	case statsTabOverview:
		return m.renderOverviewTab(m.contentWidth())
	case statsTabActivity:
		return m.renderActivityTab(m.contentWidth(), m.contentHeight())
	case statsTabSessions:
		return m.renderSessionsTab(m.contentWidth())
	case statsTabTools:
		return m.renderToolsTab(m.contentWidth())
	default:
		return m.renderOverviewTab(m.contentWidth())
	}
}

func (m statsModel) contentWidth() int {
	return max(m.width-2, 1)
}

func (m statsModel) contentHeight() int {
	return max(m.height-7, 1)
}

func (m statsModel) filteredConversations() []conv.Conversation {
	return applyStructuredFilters(m.conversations, m.filter.dimensions)
}

func (m statsModel) filteredSessions() []conv.SessionMeta {
	conversations := m.filteredConversations()
	sessions := make([]conv.SessionMeta, 0, len(conversations))
	for _, conversation := range conversations {
		sessions = append(sessions, conversation.Sessions...)
	}
	return sessions
}

func (m statsModel) footerStatusParts() []string {
	parts := make([]string, 0, 1+len(filterBadges(m.filter.dimensions)))
	for _, badge := range filterBadges(m.filter.dimensions) {
		parts = append(parts, styleToolCall.Render("["+badge+"]"))
	}
	parts = append(parts, fmt.Sprintf("[stats] %d sessions", m.snapshot.Overview.SessionCount))
	return parts
}

func (m statsModel) scrollStatus() string {
	if m.viewport.TotalLineCount() <= m.viewport.VisibleLineCount() {
		return ""
	}

	return fmt.Sprintf("%d%%", int(math.Round(m.viewport.ScrollPercent()*100)))
}

type claudeTurnMetricSessionTarget struct {
	conversation conv.Conversation
	session      conv.SessionMeta
}

func (m statsModel) claudeTurnMetricSessionTargets() []claudeTurnMetricSessionTarget {
	conversations := m.filteredConversations()
	targets := make([]claudeTurnMetricSessionTarget, 0, len(conversations))
	for _, conversation := range conversations {
		if conversation.Ref.Provider != conv.ProviderClaude {
			continue
		}
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
	parts := []string{"claude-turn-metrics"}
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
		return m, tea.Batch(
			loadClaudeTurnMetricsCmd(m.ctx, m.store, m.claudeTurnMetricSessionTargets(), key),
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
		m.viewport.SetContent(m.renderSessionsTab(m.contentWidth()))
	}
	return m
}

func (m statsModel) handleSpinnerTick(msg spinner.TickMsg) (statsModel, tea.Cmd) {
	if !m.statsBackgroundLoading() {
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	m.viewport.SetContent(m.renderActiveTab())
	return m, cmd
}

func statsRange30d() stats.TimeRange {
	return statsRangeDays(30)
}

func statsRange90d() stats.TimeRange {
	return statsRangeDays(90)
}

func statsRange7d() stats.TimeRange {
	return statsRangeDays(7)
}

func statsRangeDays(days int) stats.TimeRange {
	now := statsNow()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).
		AddDate(0, 0, -(days - 1))
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), now.Location())
	return stats.TimeRange{Start: start, End: end}
}

func nextStatsTab(tab statsTab) statsTab {
	return statsTab((int(tab) + 1) % 4)
}

func prevStatsTab(tab statsTab) statsTab {
	return statsTab((int(tab) + 3) % 4)
}

func nextActivityMetric(metric activityMetric) activityMetric {
	return activityMetric((int(metric) + 1) % 3)
}

func nextStatsTimeRange(current stats.TimeRange) stats.TimeRange {
	switch statsTimeRangeLabel(current) {
	case statsRangeLabel7d:
		return statsRange30d()
	case statsRangeLabel30d:
		return statsRange90d()
	case statsRangeLabel90d:
		return stats.TimeRange{}
	default:
		return statsRange7d()
	}
}

func statsTimeRangeLabel(current stats.TimeRange) string {
	switch {
	case current.Start.IsZero() && current.End.IsZero():
		return statsRangeLabelAll
	default:
		days := int(current.End.Sub(current.Start).Hours()/24) + 1
		switch days {
		case 7:
			return statsRangeLabel7d
		case 30:
			return statsRangeLabel30d
		case 90:
			return statsRangeLabel90d
		default:
			return statsRangeLabelAll
		}
	}
}
