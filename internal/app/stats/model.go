package stats

import (
	"context"
	"fmt"
	"image/color"
	"math"
	"sync"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	el "github.com/rkuska/carn/internal/app/elements"
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
	cacheMetricHitRate cacheMetric = iota
	cacheMetricReuseRatio
)

type statsModel struct {
	conversations           []conv.Conversation
	store                   browserStore
	theme                   *el.Theme
	ctx                     context.Context
	archiveDir              string
	tab                     statsTab
	timeRange               stats.TimeRange
	snapshot                stats.Snapshot
	filter                  browserFilterState
	statsSessions           []conv.SessionMeta
	statsTurnMetrics        []conv.SessionTurnMetrics
	statsActivityBuckets    []conv.ActivityBucketRow
	splitBy                 stats.SplitDimension
	splitValues             []string
	splitToolsResult        stats.ToolsBySplit
	splitCacheResult        stats.CacheBySplit
	splitColors             map[string]color.Color
	splitExpanded           bool
	splitExpandedCursor     int
	notification            notification
	helpOpen                bool
	renderedTabContent      string
	viewport                viewport.Model
	width, height           int
	overviewLaneCursor      int
	overviewSessionCursor   int
	activityMetric          activityMetric
	activityLaneCursor      int
	sessionsLaneCursor      int
	sessionsPromptMode      stats.StatisticMode
	sessionsTurnCostMode    stats.StatisticMode
	toolsLaneCursor         int
	cacheLaneCursor         int
	cacheMetric             cacheMetric
	statsQueryFailures      statsQueryFailures
	performanceLaneCursor   int
	performanceMetricCursor int
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

func newStatsModelWithTheme(
	conversations []conv.Conversation,
	store browserStore,
	width, height int,
	filter browserFilterState,
	theme *el.Theme,
) statsModel {
	if theme == nil {
		theme = el.NewTheme(true)
	}
	vp := viewport.New()
	vp.KeyMap.PageDown.SetEnabled(false)
	vp.KeyMap.PageUp.SetEnabled(false)

	model := statsModel{
		conversations: conversations,
		store:         store,
		theme:         theme,
		ctx:           context.Background(),
		filter:        copyBrowserFilterState(filter),
		viewport:      vp,
		width:         width,
		height:        height,
		tab:           statsTabOverview,
		timeRange:     statsRange30d(),
	}
	model.filter.Values = extractFilterValues(conversations)
	model = model.setSize(width, height)
	model = model.applyFilterChange()
	return model
}

func newStatsModel(
	conversations []conv.Conversation,
	store browserStore,
	width, height int,
	filter browserFilterState,
) statsModel {
	return newStatsModelWithTheme(conversations, store, width, height, filter, nil)
}

func (m statsModel) Update(msg tea.Msg) (statsModel, tea.Cmd) {
	if next, cmd, handled := m.handleStatsMessage(msg); handled {
		return next, cmd
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if m.helpOpen {
			return m.handleHelpKey(keyMsg)
		}
		if m.filter.Active {
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
	case notificationMsg:
		m.notification = msg.Notification
		return m, clearNotificationAfter(msg.Notification.Kind), true
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
	return m.renderViewportContent(true)
}

func (m statsModel) applyFilterChangeAndMaybeLoad() (statsModel, tea.Cmd) {
	return m.applyFilterChange(), nil
}

func (m statsModel) renderViewportContent(resetScroll bool) statsModel {
	m = m.normalizeStatsSelection()
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

	if m.splitBy.IsActive() && !splitDimensionSupportsTab(m.tab) {
		return renderSplitUnsupportedPlaceholder(m.theme, m.contentWidth(), m.contentHeight(), m.splitBy, m.tab)
	}

	return m.renderTabBody()
}

func (m statsModel) renderTabBody() string {
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

func renderSplitUnsupportedPlaceholder(
	theme *el.Theme,
	width, height int,
	dim stats.SplitDimension,
	tab statsTab,
) string {
	msg := fmt.Sprintf("Split by %s is not supported on the %s tab.", dim.Label(), tabLabel(tab))
	body := lipgloss.NewStyle().Foreground(theme.ColorNormalDesc).Render(msg)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, body)
}

func (m statsModel) recomputeSnapshot() statsModel {
	conversations := m.filteredConversations()
	cacheKeys := filteredConversationCacheKeys(conversations)
	rows := m.loadPrecomputedStatsRows(cacheKeys)
	m = m.applyStatsQueryFailures(rows.queryFailure)
	m.statsSessions = flattenStatsSessions(conversations)
	m.statsTurnMetrics = append([]conv.SessionTurnMetrics(nil), rows.turnMetrics...)
	m.statsActivityBuckets = append([]conv.ActivityBucketRow(nil), rows.activityBuckets...)
	m.splitValues = m.extractSplitValues()
	sequence := rows.sequence
	if m.filter.Dimensions[filterDimVersion].IsActive() {
		sequence = nil
	}
	m.snapshot = stats.ComputeSnapshotWithPrecomputed(
		conversations,
		m.timeRange,
		sequence,
		m.statsTurnMetrics,
		m.statsActivityBuckets,
	)
	m = m.refreshSplitCaches()
	m = m.normalizeStatsSelection()
	return m
}

func (m statsModel) refreshSplitCaches() statsModel {
	if !m.splitBy.IsActive() {
		m.splitToolsResult = stats.ToolsBySplit{}
		m.splitCacheResult = stats.CacheBySplit{}
		m.splitColors = nil
		return m
	}
	m.splitToolsResult = m.computeSplitTools()
	m.splitCacheResult = m.computeSplitCache()
	m.splitColors = m.buildSplitColors()
	return m
}

func flattenStatsSessions(conversations []conv.Conversation) []conv.SessionMeta {
	if len(conversations) == 0 {
		return nil
	}

	count := 0
	for _, conversation := range conversations {
		count += len(conversation.Sessions)
	}
	sessions := make([]conv.SessionMeta, 0, count)
	for _, conversation := range conversations {
		for _, session := range conversation.Sessions {
			sessions = append(sessions, statsSessionWithConversation(session, conversation))
		}
	}
	return sessions
}

func statsSessionWithConversation(session conv.SessionMeta, conversation conv.Conversation) conv.SessionMeta {
	if session.Provider == "" {
		session.Provider = conversation.Ref.Provider
	}
	if session.Project.DisplayName == "" {
		session.Project = conversation.Project
	}
	return session
}

func (m statsModel) loadPrecomputedStatsRows(
	cacheKeys []string,
) statsPrecomputedRows {
	if m.store == nil || m.archiveDir == "" || len(cacheKeys) == 0 {
		return statsPrecomputedRows{}
	}

	return loadStatsRows(
		m.ctx,
		m.store,
		m.archiveDir,
		cacheKeys,
	)
}

func (m statsModel) contentWidth() int {
	return max(m.width-2, 1)
}

func (m statsModel) contentHeight() int {
	return max(m.height-7, 1)
}

func (m statsModel) filteredConversations() []conv.Conversation {
	return applyStructuredFilters(m.conversations, m.filter.Dimensions)
}

func filteredConversationCacheKeys(conversations []conv.Conversation) []string {
	cacheKeys := make([]string, 0, len(conversations))
	for _, conversation := range conversations {
		cacheKeys = append(cacheKeys, conversation.CacheKey())
	}
	return cacheKeys
}

func (m statsModel) footerStatusParts() []string {
	parts := make([]string, 0, 2+len(filterBadges(m.filter.Dimensions)))
	for _, badge := range filterBadges(m.filter.Dimensions) {
		parts = append(parts, m.theme.StyleToolCall.Render("["+badge+"]"))
	}
	if m.splitBy.IsActive() {
		parts = append(parts, m.theme.StyleToolCall.Render("[split:"+m.splitBy.Label()+"]"))
	}
	if m.statsQueryFailures.degraded() {
		parts = append(parts, renderStatsDegradedBadge(m.theme))
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
