package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestComputeSessionsAggregatesRatiosAndHistograms(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"s1",
			time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 5, 9, 2, 0, 0, time.UTC)),
			withMainMessages(2),
			withRoleCounts(1, 1),
		),
		testMeta(
			"s2",
			time.Date(2026, 1, 6, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 6, 9, 20, 0, 0, time.UTC)),
			withMainMessages(10),
			withRoleCounts(4, 6),
		),
		testMeta(
			"s3",
			time.Date(2026, 1, 7, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 7, 11, 10, 0, 0, time.UTC)),
			withMainMessages(70),
			withRoleCounts(20, 50),
		),
	}

	got := ComputeSessions(sessions)

	assert.Equal(t, 50*time.Minute+40*time.Second, got.AverageDuration)
	assert.InDelta(t, 27.3333333333, got.AverageMessages, 0.0001)
	assert.Equal(t, 25, got.UserMessageCount)
	assert.Equal(t, 57, got.AssistantMessageCount)
	assert.InDelta(t, 25.0/57.0, got.UserAssistantRatio, 0.0001)
	assert.Equal(t, 1, got.AbandonedCount)
	assert.InDelta(t, 33.3333333333, got.AbandonedRate, 0.0001)
	assert.Equal(t, HistogramBucket{Label: "<5m", Count: 1}, got.DurationHistogram[0])
	assert.Equal(t, HistogramBucket{Label: "15-30", Count: 1}, got.DurationHistogram[2])
	assert.Equal(t, HistogramBucket{Label: "2h+", Count: 1}, got.DurationHistogram[5])
	assert.Equal(t, HistogramBucket{Label: "1-5", Count: 1}, got.MessageHistogram[0])
	assert.Equal(t, HistogramBucket{Label: "5-15", Count: 1}, got.MessageHistogram[1])
	assert.Equal(t, HistogramBucket{Label: "60+", Count: 1}, got.MessageHistogram[4])
}

func TestComputeSessionsBucketBoundariesStayInclusiveAtUpperEdges(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"m5",
			time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 1, 9, 5, 0, 0, time.UTC)),
			withMainMessages(5),
		),
		testMeta(
			"m15",
			time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 1, 10, 15, 0, 0, time.UTC)),
			withMainMessages(15),
		),
		testMeta(
			"m30",
			time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 1, 11, 30, 0, 0, time.UTC)),
			withMainMessages(30),
		),
		testMeta(
			"m60",
			time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC)),
			withMainMessages(60),
		),
		testMeta(
			"m120",
			time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 1, 16, 0, 0, 0, time.UTC)),
			withMainMessages(61),
		),
	}

	got := ComputeSessions(sessions)

	assert.Equal(t, []HistogramBucket{
		{Label: "<5m", Count: 0},
		{Label: "5-15", Count: 1},
		{Label: "15-30", Count: 1},
		{Label: "30-60", Count: 1},
		{Label: "1-2h", Count: 1},
		{Label: "2h+", Count: 1},
	}, got.DurationHistogram)
	assert.Equal(t, []HistogramBucket{
		{Label: "1-5", Count: 1},
		{Label: "5-15", Count: 1},
		{Label: "15-30", Count: 1},
		{Label: "30-60", Count: 1},
		{Label: "60+", Count: 1},
	}, got.MessageHistogram)
}

func TestComputeSessionsDoesNotMarkExactAbandonThresholdsAsAbandoned(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"threshold",
			time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 2, 9, 1, 0, 0, time.UTC)),
			withMainMessages(3),
		),
		testMeta(
			"short",
			time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 2, 10, 0, 59, 0, time.UTC)),
			withMainMessages(3),
		),
		testMeta(
			"few",
			time.Date(2026, 1, 2, 11, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 1, 2, 11, 1, 0, 0, time.UTC)),
			withMainMessages(2),
		),
	}

	got := ComputeSessions(sessions)

	assert.Equal(t, 2, got.AbandonedCount)
	assert.InDelta(t, 66.6667, got.AbandonedRate, 0.0001)
}

func TestComputeSessionsUsesTotalMessageCountForSubagentSessions(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta(
			"subagent",
			time.Date(2026, 3, 22, 9, 0, 0, 0, time.UTC),
			withLastTimestamp(time.Date(2026, 3, 22, 9, 10, 0, 0, time.UTC)),
			withRoleCounts(3, 5),
			func(meta *sessionMeta) {
				meta.IsSubagent = true
				meta.MessageCount = 8
				meta.MainMessageCount = 0
			},
		),
	}

	got := ComputeSessions(sessions)

	assert.InDelta(t, 8.0, got.AverageMessages, 0.0001)
	assert.Zero(t, got.AbandonedCount)
	assert.Equal(t, HistogramBucket{Label: "5-15", Count: 1}, got.MessageHistogram[1])
}

func TestComputeTurnTokenMetricsAveragesPromptAndTurnCostByPosition(t *testing.T) {
	t.Parallel()

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessage(10, 5),
			assistantUsageMessage(30, 10),
			userMessage(),
			assistantUsageMessage(40, 10),
		}),
		testSession("s2", []message{
			userMessage(),
			assistantUsageMessage(20, 10),
			userMessage(),
			assistantUsageMessage(50, 20),
			assistantUsageMessage(60, 20),
		}),
		testSession("s3", []message{
			assistantUsageMessage(25, 5),
			assistantUsageMessage(35, 10),
			userMessage(),
			assistantUsageMessage(70, 10),
			userMessage(),
			assistantUsageMessage(80, 20),
		}),
		testSession("zero", []message{
			userMessage(),
			userMessage(),
		}),
	}

	got := ComputeTurnTokenMetrics(sessions)
	require.Len(t, got, 2)
	assert.Equal(t, 1, got[0].Position)
	assert.Equal(t, 3, got[0].SampleCount)
	assert.InDelta(t, 40.0, got[0].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 55.0, got[0].AverageTurnTokens, 0.0001)
	assert.Equal(t, 2, got[1].Position)
	assert.Equal(t, 3, got[1].SampleCount)
	assert.InDelta(t, 60.0, got[1].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 100.0, got[1].AverageTurnTokens, 0.0001)
}

func TestComputeTurnTokenMetricsUsesRealTurnBoundariesInsteadOfAssistantSteps(t *testing.T) {
	t.Parallel()

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessage(100, 10),
			assistantUsageMessage(200, 20),
			userMessage(),
			assistantUsageMessage(400, 40),
		}),
		testSession("s2", []message{
			userMessage(),
			assistantUsageMessage(150, 15),
			assistantUsageMessage(300, 30),
			userMessage(),
			assistantUsageMessage(500, 50),
		}),
		testSession("s3", []message{
			assistantUsageMessage(50, 5),
			assistantUsageMessage(250, 25),
			userMessage(),
			assistantUsageMessage(600, 60),
			userMessage(),
			assistantUsageMessage(700, 70),
		}),
	}

	got := ComputeTurnTokenMetrics(sessions)
	require.Len(t, got, 2)
	assert.Equal(t, 1, got[0].Position)
	assert.InDelta(t, 366.6666666667, got[0].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 495.0, got[0].AverageTurnTokens, 0.0001)
	assert.Equal(t, 2, got[1].Position)
	assert.InDelta(t, 533.3333333333, got[1].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 586.6666666667, got[1].AverageTurnTokens, 0.0001)
}

func TestComputeTurnTokenMetricsUsesPromptTokensAndFullAssistantTurnCost(t *testing.T) {
	t.Parallel()

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessageWithUsage(conv.TokenUsage{
				InputTokens:              10,
				CacheCreationInputTokens: 5,
				CacheReadInputTokens:     15,
				OutputTokens:             4,
				ReasoningOutputTokens:    6,
			}),
			assistantUsageMessageWithUsage(conv.TokenUsage{
				InputTokens:           20,
				CacheReadInputTokens:  5,
				OutputTokens:          3,
				ReasoningOutputTokens: 1,
			}),
		}),
		testSession("s2", []message{
			userMessage(),
			assistantUsageMessageWithUsage(conv.TokenUsage{
				InputTokens:              30,
				CacheCreationInputTokens: 10,
				OutputTokens:             5,
				ReasoningOutputTokens:    5,
			}),
		}),
		testSession("s3", []message{
			userMessage(),
			assistantUsageMessageWithUsage(conv.TokenUsage{
				InputTokens:              20,
				CacheCreationInputTokens: 10,
				CacheReadInputTokens:     30,
				OutputTokens:             10,
				ReasoningOutputTokens:    5,
			}),
		}),
	}

	got := ComputeTurnTokenMetrics(sessions)
	require.Len(t, got, 1)
	assert.Equal(t, 1, got[0].Position)
	assert.InDelta(t, 43.3333333333, got[0].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 64.6666666667, got[0].AverageTurnTokens, 0.0001)
}

func TestCollectSessionTurnMetricsCapturesUserTurnsPerSession(t *testing.T) {
	t.Parallel()

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessage(100, 10),
			userToolResultMessage(),
			assistantUsageMessage(200, 20),
			userMessage(),
			assistantUsageMessage(300, 30),
		}),
		testSession("empty", []message{
			userMessage(),
			userMessage(),
		}),
	}
	sessions[0].Meta.Provider = conv.ProviderClaude
	sessions[0].Meta.Version = "1.0.0"

	got := CollectSessionTurnMetrics(sessions)
	require.Len(t, got, 1)
	assert.Equal(t, conv.ProviderClaude, got[0].Provider)
	assert.Equal(t, "1.0.0", got[0].Version)
	assert.Equal(t, sessions[0].Meta.Timestamp, got[0].Timestamp)
	assert.Equal(t, []TurnTokens{
		{PromptTokens: 200, TurnTokens: 330},
		{PromptTokens: 300, TurnTokens: 330},
	}, got[0].Turns)
}

func TestCollectSessionTurnMetricsSkipsSubagentsAndNonMainThreadMessages(t *testing.T) {
	t.Parallel()

	sessions := []session{
		testSession("main", []message{
			assistantUsageMessage(100, 10),
			userMessage(),
			assistantUsageMessage(10, 1),
			sidechainUserMessage(),
			sidechainAssistantUsageMessage(50, 5),
			userToolResultMessage(),
			systemMessage("system"),
			assistantUsageMessage(20, 2),
			agentDividerMessage(),
			assistantUsageMessage(30, 3),
			userMessage(),
			assistantUsageMessage(40, 4),
		}),
		subagentSession("sub", []message{
			userMessage(),
			assistantUsageMessage(999, 99),
		}),
	}

	got := CollectSessionTurnMetrics(sessions)
	require.Len(t, got, 1)
	assert.Equal(t, []TurnTokens{
		{PromptTokens: 20, TurnTokens: 33},
		{PromptTokens: 40, TurnTokens: 44},
	}, got[0].Turns)
}

func TestComputeTurnTokenMetricsForRangeReusesCollectedSessionsAcrossDurations(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	sessions := []SessionTurnMetrics{
		{
			Timestamp: now,
			Turns: []TurnTokens{
				{PromptTokens: 10, TurnTokens: 15},
				{PromptTokens: 20, TurnTokens: 30},
			},
		},
		{
			Timestamp: now.AddDate(0, 0, -10),
			Turns: []TurnTokens{
				{PromptTokens: 20, TurnTokens: 25},
				{PromptTokens: 30, TurnTokens: 40},
			},
		},
		{
			Timestamp: now.AddDate(0, 0, -20),
			Turns: []TurnTokens{
				{PromptTokens: 30, TurnTokens: 35},
				{PromptTokens: 40, TurnTokens: 50},
			},
		},
		{
			Timestamp: now.AddDate(0, 0, -50),
			Turns: []TurnTokens{
				{PromptTokens: 40, TurnTokens: 45},
				{PromptTokens: 50, TurnTokens: 60},
			},
		},
	}

	thirtyDay := ComputeTurnTokenMetricsForRange(sessions, TimeRange{
		Start: now.AddDate(0, 0, -29),
		End:   now,
	})
	require.Len(t, thirtyDay, 2)
	assert.InDelta(t, 20, thirtyDay[0].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 25, thirtyDay[0].AverageTurnTokens, 0.0001)
	assert.InDelta(t, 30, thirtyDay[1].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 40, thirtyDay[1].AverageTurnTokens, 0.0001)

	allTime := ComputeTurnTokenMetricsForRange(sessions, TimeRange{})
	require.Len(t, allTime, 2)
	assert.InDelta(t, 25, allTime[0].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 30, allTime[0].AverageTurnTokens, 0.0001)
	assert.InDelta(t, 35, allTime[1].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 45, allTime[1].AverageTurnTokens, 0.0001)
}

func TestCollectSessionTurnMetricsCapturesCacheAndMemoryWrites(t *testing.T) {
	t.Parallel()

	memoryWrite := assistantMemoryWrite("/u/proj/memory/notes.md")
	nonMemoryWrite := conv.Message{
		Role: conv.RoleAssistant,
		ToolCalls: []conv.ToolCall{{
			Name: "Write",
			Action: conv.NormalizedAction{
				Type: conv.NormalizedActionRewrite,
				Targets: []conv.ActionTarget{
					{Type: conv.ActionTargetFilePath, Value: "/u/proj/src/foo.go"},
				},
			},
		}},
	}

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessageWithUsage(conv.TokenUsage{
				InputTokens:              10,
				CacheReadInputTokens:     4000,
				CacheCreationInputTokens: 1200,
				OutputTokens:             3,
			}),
			memoryWrite,
			assistantUsageMessageWithUsage(conv.TokenUsage{
				InputTokens:              5,
				CacheReadInputTokens:     5000,
				CacheCreationInputTokens: 900,
				OutputTokens:             2,
			}),
			userMessage(),
			assistantUsageMessageWithUsage(conv.TokenUsage{
				InputTokens:          20,
				CacheReadInputTokens: 1000,
				OutputTokens:         5,
			}),
			nonMemoryWrite,
		}),
	}

	got := CollectSessionTurnMetrics(sessions)
	require.Len(t, got, 1)
	require.Len(t, got[0].Turns, 2)

	// First-within-turn semantics: the first assistant message's cache state
	// is what defines the turn's entry, not max across messages.
	assert.Equal(t, 4000, got[0].Turns[0].CacheReadTokens)
	assert.Equal(t, 1200, got[0].Turns[0].CacheCreationTokens)
	assert.Equal(t, 1, got[0].Turns[0].MemoryWriteCount)

	assert.Equal(t, 1000, got[0].Turns[1].CacheReadTokens)
	assert.Equal(t, 0, got[0].Turns[1].CacheCreationTokens)
	assert.Equal(t, 0, got[0].Turns[1].MemoryWriteCount)
}

func TestCollectSessionTurnMetricsCacheTokensCaptureTurnEntryEvenWhenFirstMessageIsCold(t *testing.T) {
	t.Parallel()

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessageWithUsage(conv.TokenUsage{
				InputTokens:              100,
				CacheReadInputTokens:     0,
				CacheCreationInputTokens: 50_000,
				OutputTokens:             5,
			}),
			assistantUsageMessageWithUsage(conv.TokenUsage{
				InputTokens:          20,
				CacheReadInputTokens: 50_000,
				OutputTokens:         3,
			}),
		}),
	}

	got := CollectSessionTurnMetrics(sessions)
	require.Len(t, got, 1)
	require.Len(t, got[0].Turns, 1)
	// A cold-started turn stays cold even if later messages read from the
	// cache the first message just created.
	assert.Zero(t, got[0].Turns[0].CacheReadTokens)
	assert.Equal(t, 50_000, got[0].Turns[0].CacheCreationTokens)
}

func TestCollectSessionTurnMetricsActivatesTurnWhenDividerCarriesUserText(t *testing.T) {
	t.Parallel()

	dividerWithText := conv.Message{
		Role:           conv.RoleUser,
		Text:           "Quick exploration",
		IsAgentDivider: true,
	}
	memoryWrite := assistantMemoryWrite("/u/proj/memory/notes.md")

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessage(10, 5),
			dividerWithText,
			memoryWrite,
			assistantUsageMessage(40, 20),
		}),
	}

	got := CollectSessionTurnMetrics(sessions)
	require.Len(t, got, 1)
	require.Len(t, got[0].Turns, 2)
	// First turn closed at the divider with no memory writes.
	assert.Zero(t, got[0].Turns[0].MemoryWriteCount)
	// Post-divider turn picks up the memory write that previously was lost.
	assert.Equal(t, 1, got[0].Turns[1].MemoryWriteCount)
}

func TestCollectSessionTurnMetricsFlushesOnTextlessDivider(t *testing.T) {
	t.Parallel()

	memoryWrite := assistantMemoryWrite("/u/proj/memory/notes.md")
	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessage(10, 5),
			agentDividerMessage(),
			memoryWrite,
			assistantUsageMessage(20, 10),
		}),
	}

	got := CollectSessionTurnMetrics(sessions)
	require.Len(t, got, 1)
	// Only the first turn is recorded — nothing after a textless divider
	// attaches until the next user-text boundary arrives.
	require.Len(t, got[0].Turns, 1)
	assert.Zero(t, got[0].Turns[0].MemoryWriteCount)
}

func TestCollectSessionTurnMetricsCountsMemoryWritesFromUserToolCalls(t *testing.T) {
	t.Parallel()

	userMemoryWrite := conv.Message{
		Role: conv.RoleUser,
		ToolCalls: []conv.ToolCall{{
			Name: "Write",
			Action: conv.NormalizedAction{
				Type: conv.NormalizedActionRewrite,
				Targets: []conv.ActionTarget{
					{Type: conv.ActionTargetFilePath, Value: "/u/proj/memory/MEMORY.md"},
				},
			},
		}},
	}

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			userMemoryWrite,
			assistantUsageMessage(10, 5),
		}),
	}

	got := CollectSessionTurnMetrics(sessions)
	require.Len(t, got, 1)
	require.Len(t, got[0].Turns, 1)
	assert.Equal(t, 1, got[0].Turns[0].MemoryWriteCount)
}

func TestComputeTurnTokenMetricsKeepsSparsePositions(t *testing.T) {
	t.Parallel()

	sessions := []session{
		testSession("s1", []message{
			userMessage(),
			assistantUsageMessage(100, 10),
		}),
		testSession("s2", []message{
			userMessage(),
			assistantUsageMessage(110, 10),
		}),
		testSession("s3", []message{
			userMessage(),
			assistantUsageMessage(120, 10),
			userMessage(),
			assistantUsageMessage(220, 20),
		}),
	}

	got := ComputeTurnTokenMetrics(sessions)

	require.Len(t, got, 2)
	assert.Equal(t, 1, got[0].Position)
	assert.Equal(t, 3, got[0].SampleCount)
	assert.Equal(t, 2, got[1].Position)
	assert.Equal(t, 1, got[1].SampleCount)
}
