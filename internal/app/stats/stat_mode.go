package stats

import (
	statspkg "github.com/rkuska/carn/internal/stats"
)

var statisticModesWithoutTotal = []statspkg.StatisticMode{
	statspkg.StatisticModeAverage,
	statspkg.StatisticModeP50,
	statspkg.StatisticModeP95,
	statspkg.StatisticModeP99,
	statspkg.StatisticModeMax,
}

func (m statsModel) sessionTurnMetricsForMode(
	mode statspkg.StatisticMode,
) []statspkg.PositionTokenMetrics {
	if mode == statspkg.StatisticModeAverage && len(m.snapshot.Sessions.ClaudeTurnMetrics) > 0 {
		return m.snapshot.Sessions.ClaudeTurnMetrics
	}
	return statspkg.ComputeTurnTokenMetricsForRangeWithMode(m.statsTurnMetrics, m.timeRange, mode)
}

func buildSessionTurnLaneTitle(
	base string,
	grouped bool,
	providerLabel string,
	mode statspkg.StatisticMode,
) string {
	title := base
	if !grouped {
		title = sessionTurnMetricBadge(mode) + " " + base
	}
	if providerLabel == "" {
		return title
	}
	return title + " (" + providerLabel + ")"
}

func sessionTurnMetricBadge(mode statspkg.StatisticMode) string {
	if mode == statspkg.StatisticModeAverage {
		return "Avg"
	}
	return mode.ShortLabel()
}
