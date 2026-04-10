package app

import statspkg "github.com/rkuska/carn/internal/stats"

const performanceLaneCount = 4

func (m statsModel) normalizePerformanceSelection() statsModel {
	m.performanceLaneCursor = clampCursor(m.performanceLaneCursor, performanceLaneCount)
	lane, ok := m.performanceLaneAt(m.performanceLaneCursor)
	if !ok {
		m.performanceLaneCursor = 0
		m.performanceMetricCursor = 0
		return m
	}
	if len(lane.Metrics) == 0 {
		m.performanceMetricCursor = 0
		return m
	}
	m.performanceMetricCursor = clampCursor(m.performanceMetricCursor, len(lane.Metrics))
	return m
}

func (m statsModel) performanceLaneAt(cursor int) (statspkg.PerformanceLane, bool) {
	switch cursor {
	case 0:
		return m.snapshot.Performance.Outcome, true
	case 1:
		return m.snapshot.Performance.Discipline, true
	case 2:
		return m.snapshot.Performance.Efficiency, true
	case 3:
		return m.snapshot.Performance.Robustness, true
	default:
		return statspkg.PerformanceLane{}, false
	}
}

func (m statsModel) performanceScopeAllowsScorecard() bool {
	return m.snapshot.Performance.Scope.SingleFamily
}

func (m statsModel) performanceScopeGateActive() bool {
	return m.tab == statsTabPerformance && m.snapshot.Overview.SessionCount > 0 && !m.performanceScopeAllowsScorecard()
}

func (m statsModel) performanceScopeFilterDimension() filterDimension {
	scope := m.snapshot.Performance.Scope
	if !scope.SingleProvider {
		return filterDimProvider
	}
	if !scope.SingleModel {
		return filterDimModel
	}
	return filterDimProvider
}

func (m statsModel) selectedPerformanceLane() (statspkg.PerformanceLane, int, bool) {
	m = m.normalizePerformanceSelection()
	lane, ok := m.performanceLaneAt(m.performanceLaneCursor)
	if !ok {
		return statspkg.PerformanceLane{}, 0, false
	}
	return lane, m.performanceLaneCursor, true
}

func (m statsModel) selectedPerformanceMetric() (statspkg.PerformanceMetric, statspkg.PerformanceLane, int, bool) {
	m = m.normalizePerformanceSelection()
	lane, ok := m.performanceLaneAt(m.performanceLaneCursor)
	if !ok || len(lane.Metrics) == 0 {
		return statspkg.PerformanceMetric{}, statspkg.PerformanceLane{}, 0, false
	}
	return lane.Metrics[m.performanceMetricCursor], lane, m.performanceMetricCursor, true
}

func (m statsModel) nextPerformanceMetric() statsModel {
	lane, _, ok := m.selectedPerformanceLane()
	if !ok || len(lane.Metrics) == 0 {
		return m
	}
	m.performanceMetricCursor = (m.performanceMetricCursor + 1) % len(lane.Metrics)
	return m.normalizePerformanceSelection()
}
