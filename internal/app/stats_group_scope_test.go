package app

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestStatsOverviewProviderVersionLaneUsesFilteredSnapshotInsteadOfGroupScope(t *testing.T) {
	t.Parallel()

	const version = "1.0.0"

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"claude-1",
				"alpha",
				testStatsSessionMeta("claude-1", "alpha", now, func(meta *conv.SessionMeta) {
					meta.Provider = conv.ProviderClaude
					meta.Version = version
					meta.TotalUsage.InputTokens = 200
					meta.TotalUsage.OutputTokens = 50
				}),
			),
			testStatsConversationWithProviderAndSessions(
				conv.ProviderCodex,
				"codex-1",
				"beta",
				testStatsSessionMeta("codex-1", "beta", now.Add(-time.Hour), func(meta *conv.SessionMeta) {
					meta.Provider = conv.ProviderCodex
					meta.TotalUsage.InputTokens = 120
					meta.TotalUsage.OutputTokens = 30
				}),
			),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.overviewLaneCursor = 1

	view := ansi.Strip(m.renderOverviewTab(120))
	assert.Contains(t, view, "Tokens by (Provider, Version)")
	assert.Contains(t, view, "Claude 1.0.0")
	assert.Contains(t, view, "Codex unknown")
	assert.Contains(t, view, "Tokens by Project")

	m.overviewLaneCursor = 2
	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	assert.False(t, m.groupScope.active)

	m.groupScope.provider = conv.ProviderClaude
	view = ansi.Strip(m.renderOverviewTab(120))
	assert.Contains(t, view, "Claude "+version)
	assert.Contains(t, view, "Codex unknown")
}

func TestStatsSessionsVersionToggleClearsSharedScope(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.tab = statsTabSessions
	m.sessionsLaneCursor = 2

	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	assert.True(t, m.sessionsGrouped)
	assert.True(t, m.groupScope.active)

	m.groupScope.active = false
	m.groupScope.provider = conv.ProviderClaude
	m.groupScope.versions = map[string]bool{"1.0.0": true}

	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	assert.False(t, m.sessionsGrouped)
	assert.False(t, m.groupScope.active)
	assert.Equal(t, conv.Provider(""), m.groupScope.provider)
	assert.Empty(t, m.groupScope.versions)
}

func TestStatsSessionsVersionToggleWorksWithoutSelectingTurnLane(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.tab = statsTabSessions
	m.sessionsLaneCursor = 0

	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})

	assert.True(t, m.sessionsGrouped)
	assert.True(t, m.groupScope.active)
}

func TestStatsSessionsVersionScopeSelectionRefreshesRenderedContent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	store := &fakeBrowserStore{
		turnMetricRows: testClaudeVersionTurnMetricRows(now, 300, 360),
	}
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-1",
				"alpha",
				testStatsSessionMeta("stats-1", "alpha", now, func(meta *conv.SessionMeta) {
					meta.Provider = conv.ProviderClaude
				}),
			),
		},
		store,
		120,
		32,
		newBrowserFilterState(),
	)
	m.archiveDir = t.TempDir()
	m = m.applyFilterChange()
	m.tab = statsTabSessions
	m.sessionsLaneCursor = 0

	m, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	require.True(t, m.groupScope.active)

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = m.Update(tea.KeyPressMsg{Text: "q"})

	assert.False(t, m.groupScope.active)
	assert.Contains(t, ansi.Strip(m.renderedTabContent), "1.0.0")
}

func TestStatsSessionsGroupedTurnSeriesKeepLateTurnsForSelectedProvider(t *testing.T) {
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
			},
			claudeConversation.CacheKey(): {
				testTurnMetricRowWithLength(now.Add(-time.Minute), conv.ProviderClaude, "1.0.0", 12),
				testTurnMetricRowWithLength(now.Add(-2*time.Minute), conv.ProviderClaude, "1.0.0", 12),
			},
		},
	}

	m := newStatsModel(
		[]conv.Conversation{codexConversation, claudeConversation},
		store,
		120,
		32,
		newBrowserFilterState(),
	)
	m.archiveDir = t.TempDir()
	m = m.applyFilterChange()
	m.tab = statsTabSessions
	m.sessionsGrouped = true
	m.groupScope.provider = conv.ProviderCodex
	m.groupScope.versions = map[string]bool{"2.0.0": true}

	series := m.groupedTurnSeries()

	require.Len(t, series, 1)
	require.Len(t, series[0].Metrics, 12)
	assert.Equal(t, 1, series[0].Metrics[0].Position)
	assert.Equal(t, 12, series[0].Metrics[len(series[0].Metrics)-1].Position)
	assert.Equal(t, 1, series[0].Metrics[11].SampleCount)
}

func TestStatsVersionFilterScopesOverviewTrendActivityAndTurnMetrics(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	restoreNow := setStatsNowForTest(func() time.Time {
		return now
	})
	defer restoreNow()

	store := &fakeBrowserStore{
		turnMetricRowsByKey:     map[string][]conv.SessionTurnMetrics{},
		activityBucketRowsByKey: map[string][]conv.ActivityBucketRow{},
	}

	conversation := testStatsConversationWithProviderAndSessions(
		conv.ProviderClaude,
		"stats-1",
		"alpha",
		testStatsSessionMeta("v1", "alpha", time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC), func(meta *conv.SessionMeta) {
			meta.Provider = conv.ProviderClaude
			meta.Version = "1.0.0"
			meta.TotalUsage.InputTokens = 1200
			meta.TotalUsage.OutputTokens = 0
		}),
		testStatsSessionMeta("unknown", "alpha", time.Date(2026, 3, 20, 13, 0, 0, 0, time.UTC), func(meta *conv.SessionMeta) {
			meta.Provider = conv.ProviderClaude
			meta.Version = ""
			meta.TotalUsage.InputTokens = 3000
			meta.TotalUsage.OutputTokens = 0
		}),
	)

	store.turnMetricRowsByKey[conversation.CacheKey()] = testClaudeVersionTurnMetricRows(now, 300, 360)
	store.activityBucketRowsByKey[conversation.CacheKey()] = []conv.ActivityBucketRow{
		{
			BucketStart:           time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC),
			Provider:              "claude",
			Version:               "1.0.0",
			SessionCount:          1,
			MessageCount:          4,
			UserMessageCount:      2,
			AssistantMessageCount: 2,
			InputTokens:           1000,
		},
		{
			BucketStart:           time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
			Provider:              "claude",
			Version:               "1.0.0",
			SessionCount:          1,
			MessageCount:          4,
			UserMessageCount:      2,
			AssistantMessageCount: 2,
			InputTokens:           1200,
		},
		{
			BucketStart:           time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
			Provider:              "claude",
			Version:               "",
			SessionCount:          1,
			MessageCount:          4,
			UserMessageCount:      2,
			AssistantMessageCount: 2,
			InputTokens:           3000,
		},
	}

	m := newStatsModel(
		[]conv.Conversation{conversation},
		store,
		120,
		32,
		newBrowserFilterState(),
	)
	m.archiveDir = t.TempDir()
	m.timeRange = statsRange7d()
	m.versionFilter.selected = map[string]bool{"1.0.0": true}
	m = m.applyFilterChange()

	assert.Equal(t, 1, m.snapshot.Overview.SessionCount)
	assert.Contains(t, ansi.Strip(m.renderOverviewTab(120)), "tokens 1,200 +20%")

	totalTokens := 0
	for _, day := range m.snapshot.Activity.DailyTokens {
		totalTokens += day.Count
	}
	assert.Equal(t, 1200, totalTokens)

	require.Len(t, m.snapshot.Sessions.ClaudeTurnMetrics, 1)
	assert.InDelta(t, 120.0, m.snapshot.Sessions.ClaudeTurnMetrics[0].AveragePromptTokens, 0.0001)
}
