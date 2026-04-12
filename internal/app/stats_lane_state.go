package app

import statspkg "github.com/rkuska/carn/internal/stats"

type statsLaneID string

type statsLane struct {
	id             statsLaneID
	title          string
	metricHelpDesc string
	supportsMetric bool
	supportsOpen   bool
}

const (
	statsLaneOverviewModel statsLaneID = "overview_model"
	statsLaneOverviewProj  statsLaneID = "overview_project"
	statsLaneOverviewTop   statsLaneID = "overview_top_sessions"

	statsLaneActivityDaily   statsLaneID = "activity_daily"
	statsLaneActivityHeatmap statsLaneID = "activity_heatmap"

	statsLaneSessionsDuration statsLaneID = "sessions_duration"
	statsLaneSessionsMessages statsLaneID = "sessions_messages"
	statsLaneSessionsContext  statsLaneID = "sessions_context"
	statsLaneSessionsTurnCost statsLaneID = "sessions_turn_cost"

	statsLaneToolsCalls    statsLaneID = "tools_calls"
	statsLaneToolsTop      statsLaneID = "tools_top"
	statsLaneToolsErrors   statsLaneID = "tools_errors"
	statsLaneToolsRejected statsLaneID = "tools_rejected"

	statsLaneCacheDaily   statsLaneID = "cache_daily"
	statsLaneCacheSegment statsLaneID = "cache_segment"
	statsLaneCacheMiss    statsLaneID = "cache_miss"
	statsLaneCacheHitDur  statsLaneID = "cache_hit_dur"

	statsLanePerformanceOutcome    statsLaneID = "performance_outcome"
	statsLanePerformanceDiscipline statsLaneID = "performance_discipline"
	statsLanePerformanceEfficiency statsLaneID = "performance_efficiency"
	statsLanePerformanceRobustness statsLaneID = "performance_robustness"
)

func overviewStatsLanes() []statsLane {
	return []statsLane{
		{id: statsLaneOverviewModel, title: "Tokens by Model"},
		{id: statsLaneOverviewProj, title: "Tokens by Project"},
		{
			id:             statsLaneOverviewTop,
			title:          "Most Token-Heavy Sessions",
			metricHelpDesc: "session",
			supportsMetric: true,
			supportsOpen:   true,
		},
	}
}

func activityStatsLanes() []statsLane {
	return []statsLane{
		{id: statsLaneActivityDaily, title: "Daily Activity", metricHelpDesc: "metric", supportsMetric: true},
		{id: statsLaneActivityHeatmap, title: "Activity Heatmap"},
	}
}

func sessionsStatsLanes() []statsLane {
	return []statsLane{
		{id: statsLaneSessionsDuration, title: "Session Duration"},
		{id: statsLaneSessionsMessages, title: "Messages per Session"},
		{id: statsLaneSessionsContext, title: statsClaudeContextGrowthTitle},
		{id: statsLaneSessionsTurnCost, title: statsClaudeTurnCostTitle},
	}
}

func toolsStatsLanes() []statsLane {
	return []statsLane{
		{id: statsLaneToolsCalls, title: "Tool Calls/Session"},
		{id: statsLaneToolsTop, title: "Top Tools"},
		{id: statsLaneToolsErrors, title: "Tool Error Rate"},
		{id: statsLaneToolsRejected, title: statsRejectedSuggestionsTitle},
	}
}

func cacheStatsLanes() []statsLane {
	return []statsLane{
		{id: statsLaneCacheDaily, title: "Daily Cache Tokens", metricHelpDesc: "metric", supportsMetric: true},
		{id: statsLaneCacheSegment, title: "Main vs Subagent"},
		{id: statsLaneCacheMiss, title: "Cache Miss Cost by Duration"},
		{id: statsLaneCacheHitDur, title: "Cache Hit Rate by Duration"},
	}
}

func performanceStatsLanes(scorecard bool) []statsLane {
	return []statsLane{
		{id: statsLanePerformanceOutcome, title: "Outcome", metricHelpDesc: "metric", supportsMetric: scorecard},
		{id: statsLanePerformanceDiscipline, title: "Discipline", metricHelpDesc: "metric", supportsMetric: scorecard},
		{id: statsLanePerformanceEfficiency, title: "Efficiency", metricHelpDesc: "metric", supportsMetric: scorecard},
		{id: statsLanePerformanceRobustness, title: "Robustness", metricHelpDesc: "metric", supportsMetric: scorecard},
	}
}

func (m statsModel) normalizeStatsSelection() statsModel {
	m.overviewLaneCursor = clampCursor(m.overviewLaneCursor, len(overviewStatsLanes()))
	m.activityLaneCursor = clampCursor(m.activityLaneCursor, len(activityStatsLanes()))
	m.sessionsLaneCursor = clampCursor(m.sessionsLaneCursor, len(sessionsStatsLanes()))
	m.toolsLaneCursor = clampCursor(m.toolsLaneCursor, len(toolsStatsLanes()))
	m.cacheLaneCursor = clampCursor(m.cacheLaneCursor, len(cacheStatsLanes()))
	m = m.normalizePerformanceSelection()

	if m.tab != statsTabOverview || m.overviewLaneCursor != 2 {
		m.overviewSessionCursor = 0
		return m
	}

	m.overviewSessionCursor = clampCursor(m.overviewSessionCursor, len(m.snapshot.Overview.TopSessions))
	return m
}

func (m statsModel) activeStatsLanes() []statsLane {
	switch m.tab {
	case statsTabOverview:
		return overviewStatsLanes()
	case statsTabActivity:
		return activityStatsLanes()
	case statsTabSessions:
		return sessionsStatsLanes()
	case statsTabTools:
		return toolsStatsLanes()
	case statsTabCache:
		return cacheStatsLanes()
	case statsTabPerformance:
		return performanceStatsLanes(m.performanceScopeAllowsScorecard())
	default:
		return overviewStatsLanes()
	}
}

func (m statsModel) selectedStatsLane() (statsLane, int, bool) {
	m = m.normalizeStatsSelection()
	lanes := m.activeStatsLanes()
	if len(lanes) == 0 {
		return statsLane{}, 0, false
	}
	cursor := m.activeStatsLaneCursor()
	return lanes[cursor], cursor, true
}

func (m statsModel) activeStatsLaneCursor() int {
	switch m.tab {
	case statsTabOverview:
		return m.overviewLaneCursor
	case statsTabActivity:
		return m.activityLaneCursor
	case statsTabSessions:
		return m.sessionsLaneCursor
	case statsTabTools:
		return m.toolsLaneCursor
	case statsTabCache:
		return m.cacheLaneCursor
	case statsTabPerformance:
		return m.performanceLaneCursor
	default:
		return 0
	}
}

func (m statsModel) setActiveStatsLaneCursor(cursor int) statsModel {
	switch m.tab {
	case statsTabOverview:
		m.overviewLaneCursor = cursor
	case statsTabActivity:
		m.activityLaneCursor = cursor
	case statsTabSessions:
		m.sessionsLaneCursor = cursor
	case statsTabTools:
		m.toolsLaneCursor = cursor
	case statsTabCache:
		m.cacheLaneCursor = cursor
	case statsTabPerformance:
		m.performanceLaneCursor = cursor
	}
	return m
}

func (m statsModel) moveStatsLane(delta int) statsModel {
	lanes := m.activeStatsLanes()
	if len(lanes) == 0 {
		return m
	}

	cursor := m.activeStatsLaneCursor()
	cursor = (cursor + delta + len(lanes)) % len(lanes)
	m = m.setActiveStatsLaneCursor(cursor)
	if m.tab == statsTabPerformance && m.performanceScopeAllowsScorecard() {
		m.performanceMetricCursor = 0
	}
	if lane, _, ok := m.selectedStatsLane(); ok && lane.id != statsLaneOverviewTop {
		m.overviewSessionCursor = 0
	}
	return m.normalizeStatsSelection()
}

func (m statsModel) activeLaneSupportsMetric() bool {
	lane, _, ok := m.selectedStatsLane()
	return ok && lane.supportsMetric
}

func (m statsModel) activeLaneMetricHelpItem() helpItem {
	lane, _, ok := m.selectedStatsLane()
	if !ok || !lane.supportsMetric {
		return helpItem{}
	}
	return helpItem{key: "m", desc: lane.metricHelpDesc}
}

func (m statsModel) activeLaneSupportsOpen() bool {
	lane, _, ok := m.selectedStatsLane()
	if !ok || !lane.supportsOpen {
		return false
	}
	return len(m.snapshot.Overview.TopSessions) > 0
}

func (m statsModel) statsContentScrollable() bool {
	return m.viewport.TotalLineCount() > m.viewport.VisibleLineCount()
}

func (m statsModel) selectedOverviewSession() (statspkg.SessionSummary, int, bool) {
	lane, _, ok := m.selectedStatsLane()
	if !ok || lane.id != statsLaneOverviewTop || len(m.snapshot.Overview.TopSessions) == 0 {
		return statspkg.SessionSummary{}, 0, false
	}
	m = m.normalizeStatsSelection()
	return m.snapshot.Overview.TopSessions[m.overviewSessionCursor], m.overviewSessionCursor, true
}

func (m statsModel) nextOverviewSessionSelection() statsModel {
	if !m.activeLaneSupportsMetric() || len(m.snapshot.Overview.TopSessions) == 0 {
		return m
	}
	m.overviewSessionCursor = (m.overviewSessionCursor + 1) % len(m.snapshot.Overview.TopSessions)
	return m.normalizeStatsSelection()
}

func clampCursor(cursor, count int) int {
	switch {
	case count <= 0:
		return 0
	case cursor < 0:
		return 0
	case cursor >= count:
		return count - 1
	default:
		return cursor
	}
}
