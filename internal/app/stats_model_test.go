package app

import (
	"context"
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
	assert.Equal(t, statsTabOverview, m.tab)

	m, _ = m.Update(ctrlKey("b"))
	assert.Equal(t, statsTabTools, m.tab)
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

func TestStatsOverviewHelpExplainsTokenHeavyTableUsesSessions(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)

	section := m.chartHelpSection()
	require.Len(t, section.items, 3)
	assert.Equal(t, "Most Token-Heavy Sessions", section.items[2].key)
	assert.Contains(t, section.items[2].detail, "heaviest individual sessions")
	assert.Contains(t, section.items[2].detail, "not grouped conversations")
}

func TestStatsHelpSectionsListSummaryBeforeCharts(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)

	sections := m.helpSections()
	require.Len(t, sections, 3)
	assert.Equal(t, "Summary Chips", sections[0].title)
	assert.Equal(t, "Charts", sections[1].title)
	assert.Equal(t, "Navigation", sections[2].title)
}

func TestStatsSessionsHelpListsAllChipGroupsBeforeCharts(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)

	m, _ = m.Update(ctrlKey("f"))
	m, _ = m.Update(ctrlKey("f"))

	sections := m.helpSections()
	require.Len(t, sections, 3)
	require.Len(t, sections[0].items, 10)
	assert.Equal(t, []string{
		"avg duration",
		"avg messages",
		"user:assistant",
		"abandoned",
		"context 1-5 avg",
		"context 20+ avg",
		"context multiplier",
		"turn cost 1-5 avg",
		"turn cost 20+ avg",
		"turn cost multiplier",
	}, []string{
		sections[0].items[0].key,
		sections[0].items[1].key,
		sections[0].items[2].key,
		sections[0].items[3].key,
		sections[0].items[4].key,
		sections[0].items[5].key,
		sections[0].items[6].key,
		sections[0].items[7].key,
		sections[0].items[8].key,
		sections[0].items[9].key,
	})
	assert.Equal(t, []string{
		"Session Duration",
		"Messages per Session",
		"Context Growth",
		"Turn Cost",
	}, []string{
		sections[1].items[0].key,
		sections[1].items[1].key,
		sections[1].items[2].key,
		sections[1].items[3].key,
	})
}

func TestStatsChartHelpExplainsStoryAndReadingOrder(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		advance  int
		expected []struct {
			key        string
			substrings []string
		}
	}{
		{
			name:    "overview",
			advance: 0,
			expected: []struct {
				key        string
				substrings []string
			}{
				{
					key:        "Tokens by Model",
					substrings: []string{"driving token use", "Y-axis lists models", "X-axis shows total tokens"},
				},
				{
					key:        "Most Token-Heavy Sessions",
					substrings: []string{"heaviest individual sessions", "not grouped conversations", "Columns read left to right"},
				},
			},
		},
		{
			name:    "activity",
			advance: 1,
			expected: []struct {
				key        string
				substrings []string
			}{
				{
					key:        "Daily Activity",
					substrings: []string{"steady, spiky, or fading", "X-axis is calendar day", "Y-axis is the selected metric"},
				},
				{
					key:        "Activity Heatmap",
					substrings: []string{"Rows are weekdays", "columns are hours", "darker cells mean more sessions"},
				},
			},
		},
		{
			name:    "sessions",
			advance: 2,
			expected: []struct {
				key        string
				substrings []string
			}{
				{
					key:        "Session Duration",
					substrings: []string{"quick checks or long runs", "X-axis is duration bucket", "Y-axis is session count"},
				},
				{
					key: "Context Growth",
					substrings: []string{
						"context tends to accumulate",
						"usage-bearing turn number",
						"average input tokens",
					},
				},
				{
					key: "Turn Cost",
					substrings: []string{
						"prompt and response are counted together",
						"usage-bearing turn number",
						"average input+output tokens",
					},
				},
			},
		},
		{
			name:    "tools",
			advance: 3,
			expected: []struct {
				key        string
				substrings []string
			}{
				{
					key:        "Top Tools",
					substrings: []string{"dominate the workflow", "Y-axis lists tools", "X-axis shows total calls"},
				},
				{
					key:        "Tool Error Rate",
					substrings: []string{"fail often enough to inspect", "Y-axis lists tools", "X-axis shows error rate percent"},
				},
				{
					key: "Rejected Suggestions",
					substrings: []string{
						"which suggested tools users push back on",
						"Y-axis lists tools",
						"X-axis shows rejected-share percent",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			m := newStatsModel(
				[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Now())},
				&fakeBrowserStore{},
				120,
				32,
				newBrowserFilterState(),
			)

			for range testCase.advance {
				m, _ = m.Update(ctrlKey("f"))
			}

			section := m.chartHelpSection()
			for _, item := range testCase.expected {
				detail := helpItemDetail(t, section, item.key)
				for _, part := range item.substrings {
					assert.Contains(t, detail, part)
				}
			}
		})
	}
}

func TestStatsSummaryHelpUsesInterpretiveDescriptions(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("stats-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)

	section := m.summaryHelpSection()
	assert.Contains(t, helpItemDetail(t, section, "sessions"), "Sets the size of the slice")
	assert.Contains(t, helpItemDetail(t, section, "tokens"), "Shows overall token burn")
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
	filter.dimensions[filterDimProject] = dimensionFilter{
		selected: map[string]bool{"alpha": true},
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

func TestStatsSessionsTabLoadsTurnMetricsInBackgroundOncePerFilterAndReusesThemAcrossRanges(t *testing.T) {
	t.Parallel()

	store := &fakeBrowserStore{
		loadSessionResults: map[string]conv.Session{
			"stats-1":  testStatsLoadedSession("stats-1"),
			"stats-2a": testStatsLoadedSession("stats-2a"),
			"stats-2b": testStatsLoadedSession("stats-2b"),
			"stats-3":  testStatsLoadedSession("stats-3"),
		},
	}
	now := time.Now()

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

	m, _ = m.Update(ctrlKey("f"))
	m, cmd := m.Update(ctrlKey("f"))

	require.NotNil(t, cmd)
	assert.Nil(t, m.claudeTurnMetrics)
	assert.Zero(t, store.loadCalls)
	assert.Zero(t, store.loadSessionCalls)
	assert.Contains(t, ansi.Strip(m.View()), "Loading")

	firstLoad := requireBatchMsgType[claudeTurnMetricsLoadedMsg](t, cmd())
	m, _ = m.Update(firstLoad)

	require.NotEmpty(t, m.claudeTurnMetrics)
	assert.Equal(t, 4, m.snapshot.Overview.SessionCount)
	assert.Equal(t, 5, store.loadSessionCalls)
	assert.Equal(t, []string{"stats-1", "stats-2a", "stats-2b", "codex-1", "stats-3"}, store.loadSessionIDs)

	m, cmd = m.Update(ctrlKey("b"))
	assert.Nil(t, cmd)

	m, cmd = m.Update(ctrlKey("f"))
	assert.Nil(t, cmd)
	assert.Equal(t, 5, store.loadSessionCalls)

	m, cmd = m.Update(tea.KeyPressMsg{Text: "r"})
	assert.Nil(t, cmd)
	require.NotEmpty(t, m.claudeTurnMetrics)
	assert.Equal(t, statsRangeLabel90d, statsTimeRangeLabel(m.timeRange))
	assert.Equal(t, 5, m.snapshot.Overview.SessionCount)
	assert.Equal(t, 5, store.loadSessionCalls)
}

func TestStatsSessionsTabIgnoresStaleClaudeTurnMetricResults(t *testing.T) {
	t.Parallel()

	store := &fakeBrowserStore{
		loadSessionResults: map[string]conv.Session{
			"stats-1": testStatsLoadedSession("stats-1"),
			"stats-2": testStatsLoadedSession("stats-2"),
		},
	}
	now := time.Now()

	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProvider(conv.ProviderClaude, "stats-1", "alpha", now),
			testStatsConversationWithProvider(conv.ProviderClaude, "stats-2", "beta", now.AddDate(0, 0, -45)),
		},
		store,
		120,
		32,
		newBrowserFilterState(),
	)

	m, _ = m.Update(ctrlKey("f"))
	m, cmd := m.Update(ctrlKey("f"))
	require.NotNil(t, cmd)
	firstLoad := requireBatchMsgType[claudeTurnMetricsLoadedMsg](t, cmd())

	m.filter.dimensions[filterDimProject] = dimensionFilter{
		selected: map[string]bool{"alpha": true},
	}
	m, cmd = m.applyFilterChangeAndMaybeLoad()
	require.NotNil(t, cmd)
	secondLoad := requireBatchMsgType[claudeTurnMetricsLoadedMsg](t, cmd())

	m, _ = m.Update(firstLoad)
	assert.Nil(t, m.claudeTurnMetrics)
	assert.Contains(t, ansi.Strip(m.View()), "Loading")

	m, _ = m.Update(secondLoad)
	assert.Empty(t, m.claudeTurnMetrics)
	assert.False(t, m.claudeTurnMetricsLoading())
	assert.Equal(t, m.claudeTurnMetricsSourceCacheKey(), m.claudeTurnMetricsSourceKey)
	assert.Equal(t, 3, store.loadSessionCalls)
}

func TestStatsToolsTabUsesPersistedToolOutcomeCounts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
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

func TestStatsOverviewDigitShortcutOpensHeavySessionAndBackReturnsToStats(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	store := &fakeBrowserStore{
		loadSessionResults: map[string]conv.Session{
			"heavy": {
				Meta: testStatsSessionMeta("heavy", "alpha", now),
				Messages: []conv.Message{
					{Role: conv.RoleUser, Text: "question"},
					{Role: conv.RoleAssistant, Text: "answer"},
				},
			},
		},
	}
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
		store,
		120,
		32,
		newBrowserFilterState(),
	)
	m.glamourStyle = "dark"
	m.timestampFormat = "2006-01-02 15:04"

	next, cmd := m.Update(tea.KeyPressMsg{Text: "1"})
	require.NotNil(t, cmd)

	loaded, ok := cmd().(statsSessionLoadedMsg)
	require.True(t, ok)

	next, _ = next.Update(loaded)

	assert.True(t, next.viewerOpen)
	assert.Equal(t, []string{"heavy"}, store.loadSessionIDs)
	assert.Equal(t, "heavy", next.viewer.session.Meta.ID)

	next, _ = next.Update(tea.KeyPressMsg{Text: "q"})

	assert.False(t, next.viewerOpen)
	assert.Equal(t, statsTabOverview, next.tab)
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
	require.IsType(t, closeStatsMsg{}, cmd())
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

func helpItemDetail(t *testing.T, section helpSection, key string) string {
	t.Helper()

	for _, item := range section.items {
		if item.key == key {
			return item.detail
		}
	}

	t.Fatalf("help item %q not found", key)
	return ""
}

func testStatsLoadedSession(id string) conv.Session {
	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        id,
			Project:   conv.Project{DisplayName: "alpha"},
			Timestamp: time.Now(),
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "q1", Usage: conv.TokenUsage{InputTokens: 10, OutputTokens: 1}},
			{Role: conv.RoleAssistant, Text: "a1", Usage: conv.TokenUsage{InputTokens: 20, OutputTokens: 2}},
			{Role: conv.RoleUser, Text: "q2", Usage: conv.TokenUsage{InputTokens: 30, OutputTokens: 3}},
		},
	}
	return session
}

func TestStatsOpenAndCloseMessagesSwitchViewState(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	m := newAppModelWithDeps(context.Background(), cfg, testAppConfig(), &fakeBrowserStore{}, stubImportPipeline{})
	m.state = viewBrowser
	m.width = 120
	m.height = 32
	m.browser.mainConversations = []conv.Conversation{
		testStatsConversation("stats-1", "alpha", time.Now()),
	}
	m.browser.filter = newBrowserFilterState()

	next, _ := m.Update(openStatsMsg{})
	app := requireAs[appModel](t, next)
	assert.Equal(t, viewStats, app.state)
	assert.Len(t, app.stats.conversations, 1)

	next, _ = app.Update(closeStatsMsg{})
	app = requireAs[appModel](t, next)
	assert.Equal(t, viewBrowser, app.state)
}
