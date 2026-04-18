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

// buildSessionTurnLaneTitle composes the lane title for the session turn
// charts. The current statistic-mode badge is always prefixed; an optional
// split label is appended in parentheses so split and non-split lanes share
// the same "{badge} {base}" prefix.
func buildSessionTurnLaneTitle(
	base string,
	mode statspkg.StatisticMode,
	splitLabel string,
) string {
	title := sessionTurnMetricBadge(mode) + " " + base
	if splitLabel == "" {
		return title
	}
	return title + " (by " + splitLabel + ")"
}

func sessionTurnMetricBadge(mode statspkg.StatisticMode) string {
	if mode == statspkg.StatisticModeAverage {
		return "Avg"
	}
	return mode.ShortLabel()
}
