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
	if m.sessionsGrouped {
		turnChartHeight = 14
	}

	promptMetrics := sessionStats.ClaudeTurnMetrics
	turnCostMetrics := sessionStats.ClaudeTurnMetrics
	if !m.sessionsGrouped {
		promptMetrics = m.sessionTurnMetricsForMode(m.sessionsPromptMode)
		turnCostMetrics = m.sessionTurnMetricsForMode(m.sessionsTurnCostMode)
	}

	promptGrowth := renderStatsLaneBox(
		m.theme,
		m.sessionTurnLaneTitle(statsClaudePromptGrowthTitle, m.sessionsPromptMode),
		m.sessionsLaneCursor == 2,
		width,
		m.renderSessionTurnMetricBody(
			width,
			turnChartHeight,
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
	metrics []statspkg.PositionTokenMetrics,
	value func(statspkg.PositionTokenMetrics) float64,
) string {
	bodyWidth := statsLaneBodyWidth(width)
	if m.sessionsGrouped {
		return m.renderVersionedTurnMetricLaneBody(bodyWidth, height, value)
	}
	return m.renderClaudeTurnMetricLaneBody(bodyWidth, height, metrics, value)
}

func (m statsModel) sessionTurnLaneTitle(base string, mode statspkg.StatisticMode) string {
	providerLabel := ""
	if m.sessionsGrouped && m.groupScope.hasProvider() {
		providerLabel = m.groupScope.provider.Label()
	}
	return buildSessionTurnLaneTitle(base, m.sessionsGrouped, providerLabel, mode)
}

func (m statsModel) sessionTurnSummaryChips() []chip {
	if !m.sessionsGrouped {
		return claudeTurnMetricChips(m.snapshot.Sessions.ClaudeTurnMetrics)
	}
	if !m.groupScope.hasProvider() {
		return []chip{
			{Label: "mode", Value: "grouped"},
			{Label: "provider", Value: "Select with v"},
		}
	}
	return []chip{
		{Label: "mode", Value: "grouped"},
		{Label: "provider", Value: m.groupScope.provider.Label()},
		{Label: "versions", Value: statspkg.FormatNumber(len(m.groupedTurnSeries()))},
	}
}
