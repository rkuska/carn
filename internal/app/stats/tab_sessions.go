package stats

import (
	"fmt"

	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

const (
	statsClaudePromptGrowthTitle   = "Prompt Growth"
	statsClaudeTurnCostTitle       = "Turn Cost"
	statsNoClaudeTurnMetricsData   = "No main-thread turn metrics"
	statsClaudeMetricsNoDataLabel  = "turn metrics"
	statsClaudePromptEarlyLabel    = "prompt 1-5 avg"
	statsClaudePromptLateLabel     = "prompt 20+ avg"
	statsClaudePromptFactorLabel   = "prompt multiplier"
	statsClaudeTurnCostEarlyLabel  = "turn cost 1-5 avg"
	statsClaudeTurnCostLateLabel   = "turn cost 20+ avg"
	statsClaudeTurnCostFactorLabel = "turn cost multiplier"
)

func (m statsModel) renderSessionsTab(width int) string {
	sessionStats := m.snapshot.Sessions
	chips := renderSummaryChips(m.theme, []chip{
		{Label: "avg duration", Value: conv.FormatDuration(sessionStats.AverageDuration)},
		{Label: "avg messages", Value: formatFloat(sessionStats.AverageMessages)},
		{Label: "user:assistant", Value: formatRatio(sessionStats.UserAssistantRatio)},
		{Label: "abandoned", Value: fmt.Sprintf("%d (%.1f%%)", sessionStats.AbandonedCount, sessionStats.AbandonedRate)},
	}, width)

	durationBuckets := make([]histBucket, 0, len(sessionStats.DurationHistogram))
	for _, bucket := range sessionStats.DurationHistogram {
		durationBuckets = append(durationBuckets, histBucket{Label: bucket.Label, Count: bucket.Count})
	}
	messageBuckets := make([]histBucket, 0, len(sessionStats.MessageHistogram))
	for _, bucket := range sessionStats.MessageHistogram {
		messageBuckets = append(messageBuckets, histBucket{Label: bucket.Label, Count: bucket.Count})
	}

	histograms := renderStatsLanePair(
		m.theme,
		width,
		30,
		"Session Duration",
		m.sessionsLaneCursor == 0,
		func(bodyWidth int) string {
			return renderVerticalHistogramBody(m.theme, durationBuckets, bodyWidth, 8, m.theme.ColorChartTime)
		},
		"Messages per Session",
		m.sessionsLaneCursor == 1,
		func(bodyWidth int) string {
			return renderVerticalHistogramBody(m.theme, messageBuckets, bodyWidth, 8, m.theme.ColorChartBar)
		},
	)

	growthChips := renderSummaryChips(m.theme, m.sessionTurnSummaryChips(), width)
	turnChartHeight := 12
	if m.splitActive() {
		turnChartHeight = 14
	}

	promptMetrics := m.sessionTurnMetricsForMode(m.sessionsPromptMode)
	turnCostMetrics := m.sessionTurnMetricsForMode(m.sessionsTurnCostMode)

	promptGrowth := renderStatsLaneBox(
		m.theme,
		m.sessionTurnLaneTitle(statsClaudePromptGrowthTitle, m.sessionsPromptMode),
		m.sessionsLaneCursor == 2,
		width,
		m.renderSessionTurnMetricBody(
			width,
			turnChartHeight,
			m.sessionsPromptMode,
			promptMetrics,
			func(metric statspkg.PositionTokenMetrics) float64 {
				return metric.AveragePromptTokens
			},
		),
	)
	turnCost := renderStatsLaneBox(
		m.theme,
		m.sessionTurnLaneTitle(statsClaudeTurnCostTitle, m.sessionsTurnCostMode),
		m.sessionsLaneCursor == 3,
		width,
		m.renderSessionTurnMetricBody(
			width,
			turnChartHeight,
			m.sessionsTurnCostMode,
			turnCostMetrics,
			func(metric statspkg.PositionTokenMetrics) float64 {
				return metric.AverageTurnTokens
			},
		),
	)
	return joinSections(
		chips,
		histograms,
		growthChips,
		promptGrowth,
		turnCost,
		m.renderActiveMetricDetail(width),
	)
}

func (m statsModel) renderSessionTurnMetricBody(
	width, height int,
	mode statspkg.StatisticMode,
	metrics []statspkg.PositionTokenMetrics,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	bodyWidth := statsLaneBodyWidth(width)
	if m.splitActive() {
		return m.renderSplitTurnMetricLaneBody(bodyWidth, height, mode, value)
	}
	return m.renderClaudeTurnMetricLaneBody(bodyWidth, height, metrics, value)
}

func (m statsModel) sessionTurnLaneTitle(base string, mode statspkg.StatisticMode) string {
	splitLabel := ""
	if m.splitActive() && m.splitBy.SupportsTurnMetrics() {
		splitLabel = m.splitBy.Label()
	}
	return buildSessionTurnLaneTitle(base, mode, splitLabel)
}

func (m statsModel) sessionTurnSummaryChips() []chip {
	if !m.splitActive() {
		return claudeTurnMetricChips(m.snapshot.Sessions.ClaudeTurnMetrics)
	}
	return []chip{
		{Label: "mode", Value: "split"},
		{Label: "by", Value: m.splitBy.Label()},
		{Label: "series", Value: statspkg.FormatNumber(len(m.splitKeys()))},
	}
}
