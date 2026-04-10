package app

import statspkg "github.com/rkuska/carn/internal/stats"

func (m statsModel) normalizePerformanceSelection() statsModel {
	laneCount := len(m.performanceLanes())
	if laneCount == 0 {
		m.performanceLaneCursor = 0
		m.performanceMetricCursor = 0
		return m
	}

	if m.performanceLaneCursor < 0 {
		m.performanceLaneCursor = 0
	}
	if m.performanceLaneCursor >= laneCount {
		m.performanceLaneCursor = laneCount - 1
	}

	lane := m.performanceLanes()[m.performanceLaneCursor]
	if len(lane.Metrics) == 0 {
		m.performanceMetricCursor = 0
		return m
	}
	if m.performanceMetricCursor < 0 {
		m.performanceMetricCursor = 0
	}
	if m.performanceMetricCursor >= len(lane.Metrics) {
		m.performanceMetricCursor = len(lane.Metrics) - 1
	}
	return m
}

func (m statsModel) performanceLanes() []statspkg.PerformanceLane {
	return []statspkg.PerformanceLane{
		m.snapshot.Performance.Outcome,
		m.snapshot.Performance.Discipline,
		m.snapshot.Performance.Efficiency,
		m.snapshot.Performance.Robustness,
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
	laneCount := len(m.performanceLanes())
	if laneCount == 0 {
		return statspkg.PerformanceLane{}, 0, false
	}
	m = m.normalizePerformanceSelection()
	return m.performanceLanes()[m.performanceLaneCursor], m.performanceLaneCursor, true
}

func (m statsModel) selectedPerformanceMetric() (statspkg.PerformanceMetric, statspkg.PerformanceLane, int, bool) {
	lane, _, ok := m.selectedPerformanceLane()
	if !ok || len(lane.Metrics) == 0 {
		return statspkg.PerformanceMetric{}, statspkg.PerformanceLane{}, 0, false
	}
	m = m.normalizePerformanceSelection()
	return lane.Metrics[m.performanceMetricCursor], lane, m.performanceMetricCursor, true
}

func (m statsModel) movePerformanceLane(delta int) statsModel {
	laneCount := len(m.performanceLanes())
	if laneCount == 0 {
		return m
	}
	m.performanceLaneCursor = (m.performanceLaneCursor + delta + laneCount) % laneCount
	m.performanceMetricCursor = 0
	return m.normalizePerformanceSelection()
}

func (m statsModel) nextPerformanceMetric() statsModel {
	lane, _, ok := m.selectedPerformanceLane()
	if !ok || len(lane.Metrics) == 0 {
		return m
	}
	m.performanceMetricCursor = (m.performanceMetricCursor + 1) % len(lane.Metrics)
	return m.normalizePerformanceSelection()
}

func performanceVisibleMetrics(lane statspkg.PerformanceLane, selectedMetricIndex int) []statspkg.PerformanceMetric {
	if len(lane.Metrics) == 0 {
		return nil
	}

	selectedMetricIndex = max(selectedMetricIndex, 0)
	selectedMetricIndex = min(selectedMetricIndex, len(lane.Metrics)-1)

	visible := make([]statspkg.PerformanceMetric, 0, 3)
	selectedIncluded := false
	for i, metric := range lane.Metrics {
		if !metric.VisibleByDefault {
			continue
		}
		visible = append(visible, metric)
		if i == selectedMetricIndex {
			selectedIncluded = true
		}
		if len(visible) == 3 {
			break
		}
	}

	if selectedIncluded {
		return visible
	}

	selectedMetric := lane.Metrics[selectedMetricIndex]
	if len(visible) < 3 {
		return append(visible, selectedMetric)
	}
	visible[len(visible)-1] = selectedMetric
	return visible
}
