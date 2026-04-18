package stats

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestStatsSessionsPromptGrowthMetricKeyCyclesSelectedStatistic(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	restoreNow := setStatsNowForTest(func() time.Time {
		return now
	})
	defer restoreNow()

	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversation("stats-1", "alpha", now),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.tab = statsTabSessions
	m.sessionsLaneCursor = 2
	m.statsTurnMetrics = []conv.SessionTurnMetrics{
		{
			Timestamp: now,
			Turns: []conv.TurnTokens{
				{PromptTokens: 10, TurnTokens: 15},
				{PromptTokens: 50, TurnTokens: 60},
			},
		},
		{
			Timestamp: now.Add(-time.Hour),
			Turns: []conv.TurnTokens{
				{PromptTokens: 20, TurnTokens: 25},
				{PromptTokens: 60, TurnTokens: 70},
			},
		},
		{
			Timestamp: now.Add(-2 * time.Hour),
			Turns: []conv.TurnTokens{
				{PromptTokens: 30, TurnTokens: 35},
				{PromptTokens: 70, TurnTokens: 80},
			},
		},
		{
			Timestamp: now.Add(-3 * time.Hour),
			Turns: []conv.TurnTokens{
				{PromptTokens: 40, TurnTokens: 45},
				{PromptTokens: 80, TurnTokens: 90},
			},
		},
	}
	m.snapshot.Sessions.ClaudeTurnMetrics = statspkg.ComputeTurnTokenMetricsForRange(m.statsTurnMetrics, m.timeRange)

	body := ansi.Strip(m.renderSessionsTab(120))
	detail := ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, body, "Avg Prompt Growth")
	assert.Contains(t, detail, "stat avg")

	m, _ = m.Update(tea.KeyPressMsg{Text: "m"})

	assert.Equal(t, statspkg.StatisticModeP50, m.sessionsPromptMode)

	body = ansi.Strip(m.renderSessionsTab(120))
	detail = ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, body, "p50 Prompt Growth")
	assert.Contains(t, detail, "stat p50")
	assert.Contains(t, detail, "Y-axis is p50 prompt-side tokens")
}
