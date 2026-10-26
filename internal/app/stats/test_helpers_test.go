package stats

import (
	"time"

	"github.com/rkuska/carn/internal/app/testutil"
	conv "github.com/rkuska/carn/internal/conversation"
)

// newStatsModel builds a stats model whose relative time ranges are anchored
// to now, so fixtures dated around now stay in range on any wall-clock date.
func newStatsModel(
	conversations []conv.Conversation,
	store browserStore,
	width, height int,
	filter browserFilterState,
	now time.Time,
) statsModel {
	return newStatsModelWithTheme(conversations, store, width, height, filter, nil, fixedClock(now))
}

func fixedClock(now time.Time) func() time.Time {
	return func() time.Time { return now }
}

func testClaudeVersionTurnMetricRows(
	now time.Time,
	unknownPromptStart int,
	unknownTurnStart int,
) []conv.SessionTurnMetrics {
	return []conv.SessionTurnMetrics{
		testClaudeTurnMetricRow(now.Add(-4*time.Hour), "1.0.0", 100, 150),
		testClaudeTurnMetricRow(now.Add(-3*time.Hour), "1.0.0", 120, 180),
		testClaudeTurnMetricRow(now.Add(-2*time.Hour), "1.0.0", 140, 210),
		testClaudeTurnMetricRow(
			now.Add(-4*time.Hour),
			"",
			unknownPromptStart,
			unknownTurnStart,
		),
		testClaudeTurnMetricRow(
			now.Add(-3*time.Hour),
			"",
			unknownPromptStart+20,
			unknownTurnStart+20,
		),
		testClaudeTurnMetricRow(
			now.Add(-2*time.Hour),
			"",
			unknownPromptStart+40,
			unknownTurnStart+40,
		),
	}
}

func testClaudeTurnMetricRow(
	timestamp time.Time,
	version string,
	promptTokens int,
	turnTokens int,
) conv.SessionTurnMetrics {
	return conv.SessionTurnMetrics{
		Provider:  conv.ProviderClaude,
		Version:   version,
		Timestamp: timestamp,
		Turns: []conv.TurnTokens{{
			PromptTokens: promptTokens,
			TurnTokens:   turnTokens,
		}},
	}
}

func testTurnMetricRowWithLength(
	timestamp time.Time,
	provider conv.Provider,
	version string,
	turns int,
) conv.SessionTurnMetrics {
	row := conv.SessionTurnMetrics{
		Provider:  provider,
		Version:   version,
		Timestamp: timestamp,
		Turns:     make([]conv.TurnTokens, 0, turns),
	}
	for i := range turns {
		row.Turns = append(row.Turns, conv.TurnTokens{
			PromptTokens: 100 + i*10,
			TurnTokens:   150 + i*10,
		})
	}
	return row
}

func helpItemKeys(items []helpItem) []string {
	return testutil.HelpItemKeys(items)
}
