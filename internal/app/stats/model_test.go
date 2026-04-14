package stats

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestStatsTabNavigationWrapsAcrossTabs(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)

	assert.Equal(t, statsTabOverview, m.tab)

	m, _ = m.Update(ctrlKey("f"))
	assert.Equal(t, statsTabActivity, m.tab)

	m, _ = m.Update(ctrlKey("f"))
	assert.Equal(t, statsTabSessions, m.tab)

	m, _ = m.Update(ctrlKey("f"))
	assert.Equal(t, statsTabTools, m.tab)

	m, _ = m.Update(ctrlKey("f"))
	assert.Equal(t, statsTabCache, m.tab)

	m, _ = m.Update(ctrlKey("f"))
	assert.Equal(t, statsTabPerformance, m.tab)

	m, _ = m.Update(ctrlKey("f"))
	assert.Equal(t, statsTabOverview, m.tab)

	m, _ = m.Update(ctrlKey("b"))
	assert.Equal(t, statsTabPerformance, m.tab)
}

func TestStatsRangeCyclesAndUpdatesTabBar(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)

	assert.Contains(t, ansi.Strip(m.renderTabBar()), "[30d]")

	m, _ = m.Update(tea.KeyPressMsg{Text: "r"})
	assert.Contains(t, ansi.Strip(m.renderTabBar()), "[90d]")

	m, _ = m.Update(tea.KeyPressMsg{Text: "r"})
	assert.Contains(t, ansi.Strip(m.renderTabBar()), "[All]")

	m, _ = m.Update(tea.KeyPressMsg{Text: "r"})
	assert.Contains(t, ansi.Strip(m.renderTabBar()), "[7d]")

	m, _ = m.Update(tea.KeyPressMsg{Text: "r"})
	assert.Contains(t, ansi.Strip(m.renderTabBar()), "[30d]")
}

func TestStatsTabAndRangeChangesResetScroll(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		20,
		newBrowserFilterState(),
	)
	m.viewport.SetContent(strings.Repeat("line\n", 40))
	m.viewport.SetHeight(5)
	m.viewport.SetYOffset(7)

	m, _ = m.Update(ctrlKey("f"))
	assert.Equal(t, 0, m.viewport.YOffset())

	m.viewport.SetYOffset(4)
	m, _ = m.Update(tea.KeyPressMsg{Text: "r"})
	assert.Equal(t, 0, m.viewport.YOffset())
}

func TestStatsHelpToggleAndQClosesHelpBeforeView(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)

	m, _ = m.Update(tea.KeyPressMsg{Text: "?"})
	assert.True(t, m.helpOpen)

	m, cmd := m.Update(tea.KeyPressMsg{Text: "q"})
	assert.False(t, m.helpOpen)
	assert.Nil(t, cmd)

	m, _ = m.Update(tea.KeyPressMsg{Text: "?"})
	assert.True(t, m.helpOpen)

	m, _ = m.Update(tea.KeyPressMsg{Text: "?"})
	assert.False(t, m.helpOpen)
}

func TestStatsViewRendersEmptyStateWhenNoSessionsMatch(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC))},
		&fakeBrowserStore{},
		100,
		24,
		newBrowserFilterState(),
	)

	assert.Contains(t, ansi.Strip(m.View()), "No sessions match")
}

func TestStatsFooterShowsFilteredSessionCountAndBadges(t *testing.T) {
	t.Parallel()

	filter := newBrowserFilterState()
	filter.Dimensions[filterDimProject] = dimensionFilter{
		Selected: map[string]bool{"alpha": true},
	}

	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversation("stats-1", "alpha", time.Now()),
			testStatsConversation("stats-2", "beta", time.Now()),
		},
		&fakeBrowserStore{},
		120,
		32,
		filter,
	)

	view := ansi.Strip(m.View())
	assert.Contains(t, view, "[project:alpha]")
	assert.Contains(t, view, "[stats] 1 sessions")
}

func TestNewModelLoadsPrecomputedStatsWithArchiveDir(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	restoreNow := setStatsNowForTest(func() time.Time {
		return now
	})
	defer restoreNow()

	store := &fakeBrowserStore{
		turnMetricRows: []conv.SessionTurnMetrics{
			{
				Timestamp: now.Add(-time.Hour),
				Turns: []conv.TurnTokens{{
					PromptTokens: 100,
					TurnTokens:   150,
				}},
			},
			{
				Timestamp: now.Add(-2 * time.Hour),
				Turns: []conv.TurnTokens{{
					PromptTokens: 120,
					TurnTokens:   180,
				}},
			},
		},
		activityBucketRows: []conv.ActivityBucketRow{{
			BucketStart:           now,
			SessionCount:          2,
			MessageCount:          8,
			UserMessageCount:      4,
			AssistantMessageCount: 4,
			InputTokens:           60,
			OutputTokens:          20,
		}},
	}

	m := NewModel(
		context.Background(),
		t.TempDir(),
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", now)},
		store,
		120,
		32,
		newBrowserFilterState(),
	)

	require.Len(t, m.snapshot.Sessions.ClaudeTurnMetrics, 1)
	assert.Equal(t, 2, m.snapshot.Sessions.ClaudeTurnMetrics[0].SampleCount)
	require.NotEmpty(t, m.snapshot.Activity.DailySessions)
	totalSessions := 0
	for _, day := range m.snapshot.Activity.DailySessions {
		totalSessions += day.Count
	}
	assert.Equal(t, 2, totalSessions)
}

func TestStatsSessionsTabUsesPrecomputedTurnMetricsAcrossRanges(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	restoreNow := setStatsNowForTest(func() time.Time {
		return now
	})
	defer restoreNow()

	store := &fakeBrowserStore{
		turnMetricRows: []conv.SessionTurnMetrics{
			{
				Timestamp: now.Add(-time.Hour),
				Turns: []conv.TurnTokens{{
					PromptTokens: 100,
					TurnTokens:   150,
				}},
			},
			{
				Timestamp: now.Add(-2 * time.Hour),
				Turns: []conv.TurnTokens{{
					PromptTokens: 120,
					TurnTokens:   180,
				}},
			},
			{
				Timestamp: now.Add(-3 * time.Hour),
				Turns: []conv.TurnTokens{{
					PromptTokens: 140,
					TurnTokens:   210,
				}},
			},
			{
				Timestamp: now.AddDate(0, 0, -45),
				Turns: []conv.TurnTokens{{
					PromptTokens: 160,
					TurnTokens:   240,
				}},
			},
		},
	}

	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProvider(conv.ProviderClaude, "stats-1", "alpha", now),
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-2",
				"beta",
				testStatsSessionMeta("stats-2a", "beta", now),
				testStatsSessionMeta("stats-2b", "beta", now.Add(15*time.Minute)),
			),
			testStatsConversationWithProvider(conv.ProviderCodex, "codex-1", "gamma", now),
			testStatsConversationWithProvider(
				conv.ProviderClaude,
				"stats-3",
				"delta",
				now.AddDate(0, 0, -45),
			),
		},
		store,
		120,
		32,
		newBrowserFilterState(),
	)
	m.archiveDir = t.TempDir()
	m = m.applyFilterChange()

	m, _ = m.Update(ctrlKey("f"))
	m, cmd := m.Update(ctrlKey("f"))

	assert.Nil(t, cmd)
	assert.Zero(t, store.loadSessionCalls)
	assert.Equal(t, 4, m.snapshot.Overview.SessionCount)
	require.Len(t, m.snapshot.Sessions.ClaudeTurnMetrics, 1)
	assert.Equal(t, 3, m.snapshot.Sessions.ClaudeTurnMetrics[0].SampleCount)
	assert.NotContains(t, ansi.Strip(m.View()), "Loading")

	m, cmd = m.Update(ctrlKey("b"))
	assert.Nil(t, cmd)

	m, cmd = m.Update(ctrlKey("f"))
	assert.Nil(t, cmd)
	require.Len(t, m.snapshot.Sessions.ClaudeTurnMetrics, 1)
	assert.Equal(t, 3, m.snapshot.Sessions.ClaudeTurnMetrics[0].SampleCount)

	m, cmd = m.Update(tea.KeyPressMsg{Text: "r"})
	assert.Nil(t, cmd)
	assert.Equal(t, statsRangeLabel90d, statsTimeRangeLabel(m.timeRange))
	assert.Equal(t, 5, m.snapshot.Overview.SessionCount)
	require.Len(t, m.snapshot.Sessions.ClaudeTurnMetrics, 1)
	assert.Equal(t, 4, m.snapshot.Sessions.ClaudeTurnMetrics[0].SampleCount)
	assert.Zero(t, store.loadSessionCalls)
}

func TestStatsHasPlansFilterScopesActivityAndSessionTurnMetrics(t *testing.T) {
	store := &fakeBrowserStore{
		turnMetricRowsByKey:     map[string][]conv.SessionTurnMetrics{},
		activityBucketRowsByKey: map[string][]conv.ActivityBucketRow{},
	}
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	restoreNow := setStatsNowForTest(func() time.Time {
		return now
	})
	defer restoreNow()

	withPlans := testStatsConversationWithProvider(
		conv.ProviderClaude,
		"stats-1",
		"alpha",
		now,
	)
	withPlans.PlanCount = 1

	withoutPlans := testStatsConversationWithProvider(
		conv.ProviderClaude,
		"stats-2",
		"beta",
		now.Add(-time.Hour),
	)

	store.turnMetricRowsByKey[withPlans.CacheKey()] = []conv.SessionTurnMetrics{
		{
			Timestamp: now.Add(-time.Hour),
			Turns: []conv.TurnTokens{{
				PromptTokens: 100,
				TurnTokens:   150,
			}},
		},
		{
			Timestamp: now.Add(-2 * time.Hour),
			Turns: []conv.TurnTokens{{
				PromptTokens: 120,
				TurnTokens:   180,
			}},
		},
		{
			Timestamp: now.Add(-3 * time.Hour),
			Turns: []conv.TurnTokens{{
				PromptTokens: 140,
				TurnTokens:   210,
			}},
		},
	}
	store.turnMetricRowsByKey[withoutPlans.CacheKey()] = []conv.SessionTurnMetrics{
		{
			Timestamp: now.Add(-4 * time.Hour),
			Turns: []conv.TurnTokens{{
				PromptTokens: 200,
				TurnTokens:   260,
			}},
		},
		{
			Timestamp: now.Add(-5 * time.Hour),
			Turns: []conv.TurnTokens{{
				PromptTokens: 220,
				TurnTokens:   280,
			}},
		},
		{
			Timestamp: now.Add(-6 * time.Hour),
			Turns: []conv.TurnTokens{{
				PromptTokens: 240,
				TurnTokens:   300,
			}},
		},
	}

	store.activityBucketRowsByKey[withPlans.CacheKey()] = []conv.ActivityBucketRow{{
		BucketStart:           now,
		SessionCount:          1,
		MessageCount:          4,
		UserMessageCount:      2,
		AssistantMessageCount: 2,
		InputTokens:           30,
		OutputTokens:          10,
	}}
	store.activityBucketRowsByKey[withoutPlans.CacheKey()] = []conv.ActivityBucketRow{{
		BucketStart:           now.AddDate(0, 0, -2),
		SessionCount:          1,
		MessageCount:          6,
		UserMessageCount:      3,
		AssistantMessageCount: 3,
		InputTokens:           70,
		OutputTokens:          20,
	}}

	m := newStatsModel(
		[]conv.Conversation{withPlans, withoutPlans},
		store,
		120,
		32,
		newBrowserFilterState(),
	)
	m.archiveDir = t.TempDir()
	m = m.applyFilterChange()

	m.filter.Dimensions[filterDimHasPlans] = dimensionFilter{BoolState: boolFilterYes}
	m = m.applyFilterChange()

	assert.Equal(t, 1, m.snapshot.Overview.SessionCount)
	assert.Equal(t, 1, m.snapshot.Activity.ActiveDays)
	assert.Equal(t, 1, m.snapshot.Activity.CurrentStreak)

	totalTokens := 0
	for _, day := range m.snapshot.Activity.DailyTokens {
		totalTokens += day.Count
	}
	assert.Equal(t, 40, totalTokens)

	require.Len(t, m.snapshot.Sessions.ClaudeTurnMetrics, 1)
	assert.Equal(t, 3, m.snapshot.Sessions.ClaudeTurnMetrics[0].SampleCount)
	assert.InDelta(t, 120.0, m.snapshot.Sessions.ClaudeTurnMetrics[0].AveragePromptTokens, 0.0001)
	assert.Zero(t, store.loadSessionCalls)
}

func TestStatsQueryFailureShowsNotificationAndKeepsSuccessfulRows(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	restoreNow := setStatsNowForTest(func() time.Time {
		return now
	})
	defer restoreNow()

	store := &fakeBrowserStore{
		sequenceErr: errors.New("sequence boom"),
		turnMetricRows: []conv.SessionTurnMetrics{
			{
				Timestamp: now.Add(-time.Hour),
				Turns: []conv.TurnTokens{{
					PromptTokens: 100,
					TurnTokens:   150,
				}},
			},
			{
				Timestamp: now.Add(-2 * time.Hour),
				Turns: []conv.TurnTokens{{
					PromptTokens: 120,
					TurnTokens:   180,
				}},
			},
			{
				Timestamp: now.Add(-3 * time.Hour),
				Turns: []conv.TurnTokens{{
					PromptTokens: 140,
					TurnTokens:   210,
				}},
			},
		},
		activityBucketRows: []conv.ActivityBucketRow{{
			BucketStart:           now,
			SessionCount:          1,
			MessageCount:          4,
			UserMessageCount:      2,
			AssistantMessageCount: 2,
			InputTokens:           30,
			OutputTokens:          10,
		}},
	}

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", now)},
		store,
		120,
		32,
		newBrowserFilterState(),
	)
	m.archiveDir = t.TempDir()
	m = m.applyFilterChange()

	assert.Equal(t, notificationError, m.notification.Kind)
	assert.Contains(t, m.notification.Text, "couldn't load sequence metrics")
	assert.Contains(t, m.notification.Text, "Press q, then R")
	assert.True(t, m.statsQueryFailures.degraded())
	assert.False(t, m.snapshot.Performance.Scope.SequenceLoaded)

	require.Len(t, m.snapshot.Sessions.ClaudeTurnMetrics, 1)
	assert.Equal(t, 3, m.snapshot.Sessions.ClaudeTurnMetrics[0].SampleCount)

	totalTokens := 0
	for _, day := range m.snapshot.Activity.DailyTokens {
		totalTokens += day.Count
	}
	assert.Equal(t, 40, totalTokens)

	view := ansi.Strip(m.View())
	assert.Contains(t, view, statsDegradedBadgeText)
	assert.Contains(t, view, statsDegradedHintText)
}

func TestStatsSessionsTurnMetricsKeepLateTurnsWhenProviderFiltered(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	codexConversation := testStatsConversationWithProviderAndSessions(
		conv.ProviderCodex,
		"codex-1",
		"alpha",
		testStatsSessionMeta("codex-1", "alpha", now, func(meta *conv.SessionMeta) {
			meta.Provider = conv.ProviderCodex
		}),
	)
	claudeConversation := testStatsConversationWithProviderAndSessions(
		conv.ProviderClaude,
		"claude-1",
		"beta",
		testStatsSessionMeta("claude-1", "beta", now.Add(-time.Hour), func(meta *conv.SessionMeta) {
			meta.Provider = conv.ProviderClaude
		}),
	)

	store := &fakeBrowserStore{
		turnMetricRowsByKey: map[string][]conv.SessionTurnMetrics{
			codexConversation.CacheKey(): {
				testTurnMetricRowWithLength(now, conv.ProviderCodex, "2.0.0", 12),
				testTurnMetricRowWithLength(now.Add(-time.Minute), conv.ProviderCodex, "2.0.0", 12),
			},
			claudeConversation.CacheKey(): {
				testTurnMetricRowWithLength(now.Add(-2*time.Minute), conv.ProviderClaude, "1.0.0", 12),
			},
		},
	}

	filter := newBrowserFilterState()
	filter.Dimensions[filterDimProvider] = dimensionFilter{
		Selected: map[string]bool{conv.ProviderCodex.Label(): true},
	}

	m := newStatsModel(
		[]conv.Conversation{codexConversation, claudeConversation},
		store,
		120,
		32,
		filter,
	)
	m.archiveDir = t.TempDir()
	m = m.applyFilterChange()

	require.Len(t, m.snapshot.Sessions.ClaudeTurnMetrics, 12)
	assert.Equal(t, 1, m.snapshot.Sessions.ClaudeTurnMetrics[0].Position)
	assert.Equal(t, 12, m.snapshot.Sessions.ClaudeTurnMetrics[len(m.snapshot.Sessions.ClaudeTurnMetrics)-1].Position)
	assert.Equal(t, 2, m.snapshot.Sessions.ClaudeTurnMetrics[11].SampleCount)
}

func TestStatsQueryFailureClearsAfterSuccessfulRecompute(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	restoreNow := setStatsNowForTest(func() time.Time {
		return now
	})
	defer restoreNow()

	store := &fakeBrowserStore{
		sequenceErr: errors.New("sequence boom"),
		activityBucketRows: []conv.ActivityBucketRow{{
			BucketStart:           now,
			SessionCount:          1,
			MessageCount:          4,
			UserMessageCount:      2,
			AssistantMessageCount: 2,
			InputTokens:           30,
			OutputTokens:          10,
		}},
	}

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", now)},
		store,
		120,
		32,
		newBrowserFilterState(),
	)
	m.archiveDir = t.TempDir()
	m = m.applyFilterChange()

	assert.True(t, m.statsQueryFailures.degraded())
	assert.Equal(t, notificationError, m.notification.Kind)

	store.sequenceErr = nil
	store.sequenceRows = []conv.PerformanceSequenceSession{{
		Timestamp:         now,
		Mutated:           true,
		FirstPassResolved: true,
		MutationCount:     1,
		ActionCount:       1,
	}}

	m = m.applyFilterChange()

	assert.False(t, m.statsQueryFailures.degraded())
	assert.Equal(t, notification{}, m.notification)
	assert.True(t, m.snapshot.Performance.Scope.SequenceLoaded)

	view := ansi.Strip(m.View())
	assert.NotContains(t, view, statsDegradedBadgeText)
	assert.NotContains(t, view, statsDegradedHintText)
}

func TestStatsQueryFailureNotificationUpdatesWhenFailingQueryChanges(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	restoreNow := setStatsNowForTest(func() time.Time {
		return now
	})
	defer restoreNow()

	store := &fakeBrowserStore{
		sequenceErr: errors.New("sequence boom"),
	}

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", now)},
		store,
		120,
		32,
		newBrowserFilterState(),
	)
	m.archiveDir = t.TempDir()
	m = m.applyFilterChange()

	assert.Contains(t, m.notification.Text, "sequence metrics")

	store.sequenceErr = nil
	store.activityBucketErr = errors.New("activity boom")
	store.sequenceRows = []conv.PerformanceSequenceSession{{
		Timestamp:         now,
		Mutated:           true,
		FirstPassResolved: true,
		MutationCount:     1,
		ActionCount:       1,
	}}
	m = m.applyFilterChange()

	assert.True(t, m.statsQueryFailures.degraded())
	assert.Contains(t, m.notification.Text, "activity buckets")
	assert.NotContains(t, m.notification.Text, "sequence metrics")
}

func TestStatsToolsTabUsesPersistedToolOutcomeCounts(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	restoreNow := setStatsNowForTest(func() time.Time {
		return now
	})
	defer restoreNow()
	store := &fakeBrowserStore{}

	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"tools-1",
				"alpha",
				testStatsSessionMeta("tools-1", "alpha", now.AddDate(0, 0, -10), func(meta *conv.SessionMeta) {
					meta.ToolCounts = map[string]int{"Bash": 5}
					meta.ToolRejectCounts = map[string]int{"Bash": 2}
				}),
			),
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"tools-2",
				"alpha",
				testStatsSessionMeta("tools-2", "alpha", now.AddDate(0, 0, -45), func(meta *conv.SessionMeta) {
					meta.ToolCounts = map[string]int{"Bash": 5}
					meta.ToolErrorCounts = map[string]int{"Bash": 1}
				}),
			),
		},
		store,
		120,
		32,
		newBrowserFilterState(),
	)

	m, _ = m.Update(ctrlKey("f"))
	m, _ = m.Update(ctrlKey("f"))
	m, cmd := m.Update(ctrlKey("f"))

	assert.Nil(t, cmd)
	assert.Zero(t, store.loadSessionCalls)
	assert.InDelta(t, 0.0, m.snapshot.Tools.ErrorRate, 0.0001)
	assert.InDelta(t, 40.0, m.snapshot.Tools.RejectionRate, 0.0001)
	assert.Contains(t, ansi.Strip(m.View()), "Rejected Suggestions")
	assert.NotContains(t, ansi.Strip(m.View()), "Loading...")

	m, cmd = m.Update(tea.KeyPressMsg{Text: "r"})
	assert.Nil(t, cmd)
	assert.InDelta(t, 10.0, m.snapshot.Tools.ErrorRate, 0.0001)
	assert.InDelta(t, 20.0, m.snapshot.Tools.RejectionRate, 0.0001)
	assert.Zero(t, store.loadSessionCalls)
}

func TestStatsOverviewSelectedSessionReturnsOpenSessionRequest(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-1",
				"alpha",
				testStatsSessionMeta("heavy", "alpha", now, func(meta *conv.SessionMeta) {
					meta.TotalUsage.InputTokens = 4000
					meta.TotalUsage.OutputTokens = 500
					meta.FilePath = "/tmp/heavy.jsonl"
				}),
				testStatsSessionMeta("light", "alpha", now.Add(-time.Hour), func(meta *conv.SessionMeta) {
					meta.TotalUsage.InputTokens = 300
					meta.TotalUsage.OutputTokens = 50
					meta.FilePath = "/tmp/light.jsonl"
				}),
			),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.overviewLaneCursor = 3

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg, ok := cmd().(OpenSessionRequestedMsg)
	require.True(t, ok)
	assert.Equal(t, "heavy", msg.SessionMeta.ID)
	assert.Equal(t, "stats-1", msg.Conversation.ID())
}

func TestStatsOverviewMetricKeyCyclesSelectedSessionRow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-1",
				"alpha",
				testStatsSessionMeta("heavy", "alpha", now, func(meta *conv.SessionMeta) {
					meta.TotalUsage.InputTokens = 4000
					meta.TotalUsage.OutputTokens = 500
				}),
				testStatsSessionMeta("heavier", "alpha", now.Add(-time.Hour), func(meta *conv.SessionMeta) {
					meta.TotalUsage.InputTokens = 4500
					meta.TotalUsage.OutputTokens = 600
				}),
			),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.overviewLaneCursor = 3

	assert.Equal(t, 0, m.overviewSessionCursor)

	m, _ = m.Update(tea.KeyPressMsg{Text: "m"})
	assert.Equal(t, 1, m.overviewSessionCursor)

	m, _ = m.Update(tea.KeyPressMsg{Text: "m"})
	assert.Equal(t, 0, m.overviewSessionCursor)
}

func TestStatsCloseReturnsCloseMessage(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)

	_, cmd := m.Update(tea.KeyPressMsg{Text: "q"})
	require.NotNil(t, cmd)
	require.IsType(t, CloseRequestedMsg{}, cmd())
}

func ctrlKey(letter string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: rune(letter[0]), Mod: tea.ModCtrl}
}

func testStatsConversation(id, project string, ts time.Time) conv.Conversation {
	return testStatsConversationWithProvider(conv.ProviderClaude, id, project, ts)
}

func testStatsConversationWithProvider(
	provider conv.Provider,
	id, project string,
	ts time.Time,
) conv.Conversation {
	return testStatsConversationWithProviderAndSessions(
		provider,
		id,
		project,
		testStatsSessionMeta(id, project, ts),
	)
}

func testStatsConversationWithProviderAndSessions(
	provider conv.Provider,
	id, project string,
	sessions ...conv.SessionMeta,
) conv.Conversation {
	return conv.Conversation{
		Ref:      conv.Ref{Provider: provider, ID: id},
		Name:     id,
		Project:  conv.Project{DisplayName: project},
		Sessions: sessions,
	}
}

func testStatsSessionMeta(
	id, project string,
	ts time.Time,
	options ...func(*conv.SessionMeta),
) conv.SessionMeta {
	meta := conv.SessionMeta{
		ID:                    id,
		Slug:                  id,
		Project:               conv.Project{DisplayName: project},
		Timestamp:             ts,
		LastTimestamp:         ts.Add(10 * time.Minute),
		Model:                 "claude-opus-4-1",
		MainMessageCount:      4,
		UserMessageCount:      2,
		AssistantMessageCount: 2,
		TotalUsage: conv.TokenUsage{
			InputTokens:  120,
			OutputTokens: 80,
		},
		ToolCounts: map[string]int{
			"Read": 1,
		},
	}
	for _, option := range options {
		option(&meta)
	}
	return meta
}
