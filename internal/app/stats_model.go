package app

import (
	"context"
	"fmt"
	"math"
	"sync"
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
	statsTabCache
	statsTabPerformance
)

type activityMetric int

const (
	metricSessions activityMetric = iota
	metricMessages
	metricTokens
)

type cacheMetric int

const (
	cacheMetricRead cacheMetric = iota
	cacheMetricWrite
)

type statsModel struct {
	conversations                 []conv.Conversation
	store                         browserStore
	ctx                           context.Context
	archiveDir                    string
	tab                           statsTab
	timeRange                     stats.TimeRange
	snapshot                      stats.Snapshot
	claudeTurnMetricSessions      []stats.SessionTurnMetrics
	claudeTurnMetricsSourceKey    string
	claudeTurnMetrics             []stats.PositionTokenMetrics
	claudeTurnMetricsLoadingKey   string
	performanceSequenceSessions   []stats.PerformanceSequenceSession
	performanceSequenceSourceKey  string
	performanceSequenceLoadingKey string
	filter                        browserFilterState
	viewer                        viewerModel
	viewerOpen                    bool
	notification                  notification
	helpOpen                      bool
	renderedTabContent            string
	viewport                      viewport.Model
	spinner                       spinner.Model
	width, height                 int
	overviewLaneCursor            int
	overviewSessionCursor         int
	activityMetric                activityMetric
	activityLaneCursor            int
	sessionsLaneCursor            int
	toolsLaneCursor               int
	cacheLaneCursor               int
	cacheMetric                   cacheMetric
	performanceLaneCursor         int
	performanceMetricCursor       int
	glamourStyle                  string
	timestampFormat               string
	launcher                      sessionLauncher
}

const (
	statsRangeLabel7d  = "7d"
	statsRangeLabel30d = "30d"
	statsRangeLabel90d = "90d"
	statsRangeLabelAll = "All"
)

var (
	statsNowMu   sync.RWMutex
	statsNowFunc = time.Now
)

func statsNow() time.Time {
	statsNowMu.RLock()
	now := statsNowFunc
	statsNowMu.RUnlock()
	return now()
}

func setStatsNowForTest(now func() time.Time) func() {
	statsNowMu.Lock()
	previous := statsNowFunc
	statsNowFunc = now
	statsNowMu.Unlock()
	return func() {
		statsNowMu.Lock()
		statsNowFunc = previous
		statsNowMu.Unlock()
	}
}

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
	if m.viewerOpen {
		return m.updateViewer(msg)
	}

	if next, cmd, handled := m.handleStatsMessage(msg); handled {
		return next, cmd
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

func (m statsModel) handleStatsMessage(msg tea.Msg) (statsModel, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.setSize(msg.Width, msg.Height), nil, true
	case closeStatsMsg:
		return m, nil, true
	case claudeTurnMetricsLoadedMsg:
		return m.applyClaudeTurnMetricsLoaded(msg), nil, true
	case performanceSequenceLoadedMsg:
		return m.applyPerformanceSequenceLoaded(msg), nil, true
	case statsSessionLoadedMsg:
		return m.openLoadedViewer(msg), nil, true
	case spinner.TickMsg:
		next, cmd := m.handleSpinnerTick(msg)
		return next, cmd, true
	case notificationMsg:
		m.notification = msg.notification
		return m, clearNotificationAfter(msg.notification.kind), true
	case clearNotificationMsg:
		m.notification = notification{}
		return m, nil, true
	default:
		return m, nil, false
	}
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
	return m.normalizeStatsSelection().renderViewportContent(false)
}

func (m statsModel) applyFilterChange() statsModel {
	m = m.recomputeSnapshot()
	m.claudeTurnMetricSessions = nil
	m.claudeTurnMetricsSourceKey = ""
	m.claudeTurnMetrics = nil
	m.claudeTurnMetricsLoadingKey = ""
	m.performanceSequenceSessions = nil
	m.performanceSequenceSourceKey = ""
	m.performanceSequenceLoadingKey = ""
	return m.renderViewportContent(true)
}

func (m statsModel) renderViewportContent(resetScroll bool) statsModel {
	content := m.renderActiveTab()
	m.viewport.SetWidth(m.contentWidth())
	m.viewport.SetHeight(m.contentHeight())
	m = m.setViewportContent(content)
	if resetScroll {
		m.viewport.GotoTop()
	}
	return m
}

func (m statsModel) setViewportContent(content string) statsModel {
	m.renderedTabContent = content
	m.viewport.SetContent(content)
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
	case statsTabCache:
		return m.renderCacheTab(m.contentWidth(), m.contentHeight())
	case statsTabPerformance:
		return m.renderPerformanceTab(m.contentWidth())
	default:
		return m.renderOverviewTab(m.contentWidth())
	}
}

func (m statsModel) recomputeSnapshot() statsModel {
	conversations := m.filteredConversations()
	var sequence []stats.PerformanceSequenceSession
	if m.performanceSequenceSourceKey == m.performanceSequenceSourceCacheKey() {
		sequence = m.performanceSequenceSessions
	}
	m.snapshot = stats.ComputeSnapshot(conversations, m.timeRange, sequence)
	m = m.normalizeStatsSelection()
	return m
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
