package app

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestStatsRenderTabBarShowsActiveTabAndRangeWithinWidth(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(64, 32)
	m.tab = statsTabTools
	m.timeRange = statsRange7d()

	bar := ansi.Strip(m.renderTabBar())

	assert.Contains(t, bar, "▸ Tools")
	assert.Contains(t, bar, "[7d]")
	assert.NotContains(t, bar, "[30d]")
	assert.Equal(t, framedFooterContentWidth(64), lipgloss.Width(bar))
}

func TestStatsRenderSummaryChipsWrapAndUseAbbreviatedValues(t *testing.T) {
	t.Parallel()

	got := ansi.Strip(renderSummaryChips([]chip{
		{Label: "messages", Value: statspkg.FormatNumber(99999)},
		{Label: "tokens", Value: statspkg.FormatNumber(8200000)},
	}, 24))

	assert.Contains(t, got, "messages 99,999")
	assert.Contains(t, got, "tokens 8.2M")
	assert.Contains(t, got, "\n")
}

func TestStatsRenderOverviewCentersCappedTableInWideViewport(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversation("stats-1", "alpha", now),
			testStatsConversation("stats-2", "beta", now.Add(-time.Hour)),
			testStatsConversation("stats-3", "gamma", now.Add(-2*time.Hour)),
			testStatsConversation("stats-4", "delta", now.Add(-3*time.Hour)),
			testStatsConversation("stats-5", "epsilon", now.Add(-4*time.Hour)),
			testStatsConversation("stats-6", "zeta", now.Add(-5*time.Hour)),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)

	body := ansi.Strip(m.renderOverviewTab(120))
	titleLine := findRenderedLine(t, body, "Most Token-Heavy Sessions")
	headerLine := findRenderedLine(t, body, "Project")

	assert.Greater(t, lipgloss.Width(titleLine), 72)
	assert.Greater(t, lipgloss.Width(headerLine), 72)
	assert.Greater(t, strings.Index(titleLine, "Most Token-Heavy Sessions"), 0)
	assert.Greater(t, strings.Index(headerLine, "Project"), 0)
}

func TestStatsRenderOverviewStylesTokenHeavySessionValuesWithTokenColor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-1",
				"alpha",
				testStatsSessionMeta("stats-1", "alpha", now, func(meta *conv.SessionMeta) {
					meta.TotalUsage.InputTokens = 1234567
					meta.TotalUsage.OutputTokens = 890
					meta.TotalUsage.CacheReadInputTokens = 0
					meta.TotalUsage.CacheCreationInputTokens = 0
				}),
			),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)

	body := m.renderOverviewTab(120)

	assert.Contains(t, body, renderTokenValue(statspkg.FormatNumber(1235457)))
}

func TestStatsRenderOverviewShowsTokenTrendForFiniteRanges(t *testing.T) {
	restoreNow := setStatsNowForTest(func() time.Time {
		return time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	})
	defer restoreNow()

	store := &fakeBrowserStore{
		dailyTokenRows: []conv.DailyTokenRow{
			{
				Date:                  time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC),
				SessionCount:          1,
				MessageCount:          4,
				UserMessageCount:      2,
				AssistantMessageCount: 2,
				InputTokens:           1000,
			},
			{
				Date:                  time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
				SessionCount:          1,
				MessageCount:          4,
				UserMessageCount:      2,
				AssistantMessageCount: 2,
				InputTokens:           1250,
			},
		},
	}

	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-1",
				"alpha",
				testStatsSessionMeta("prev", "alpha", time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC), func(meta *conv.SessionMeta) {
					meta.TotalUsage.InputTokens = 1000
					meta.TotalUsage.OutputTokens = 0
				}),
				testStatsSessionMeta("curr", "alpha", time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC), func(meta *conv.SessionMeta) {
					meta.TotalUsage.InputTokens = 1250
					meta.TotalUsage.OutputTokens = 0
				}),
			),
		},
		store,
		120,
		32,
		newBrowserFilterState(),
	)
	m.archiveDir = t.TempDir()
	m.timeRange = statsRange7d()
	m = m.applyFilterChange()

	body := ansi.Strip(m.renderOverviewTab(120))

	assert.Contains(t, body, "tokens 1,250 +25%")
}

func TestStatsRenderOverviewOmitsTokenTrendForAllRange(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-1",
				"alpha",
				testStatsSessionMeta("prev", "alpha", time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC), func(meta *conv.SessionMeta) {
					meta.TotalUsage.InputTokens = 1000
				}),
				testStatsSessionMeta("curr", "alpha", time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC), func(meta *conv.SessionMeta) {
					meta.TotalUsage.InputTokens = 1250
				}),
			),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.timeRange = statspkg.TimeRange{}
	m = m.applyFilterChange()

	body := ansi.Strip(m.renderOverviewTab(120))

	assert.Contains(t, body, "tokens 2,410")
	assert.NotContains(t, body, "+")
	assert.NotContains(t, body, "~")
}

func TestStatsRenderToolsUsesShareChipsInsteadOfCompoundRatio(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-1",
				"alpha",
				testStatsSessionMeta(
					"stats-1",
					"alpha",
					time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC),
					func(meta *conv.SessionMeta) {
						meta.ToolCounts = map[string]int{
							"Read":  4,
							"Write": 2,
							"Bash":  1,
						}
					},
				),
			),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.tab = statsTabTools

	body := ansi.Strip(m.renderToolsTab(120))

	assert.Contains(t, body, "read 57%")
	assert.Contains(t, body, "write 29%")
	assert.Contains(t, body, "bash 14%")
	assert.NotContains(t, body, "read:write:bash")
}

func TestStatsRenderToolsUsesGridRowsForUsageAndQualityCharts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-1",
				"alpha",
				testStatsSessionMeta("stats-1", "alpha", now, func(meta *conv.SessionMeta) {
					meta.ToolCounts = map[string]int{
						"Read":  12,
						"Write": 6,
						"Bash":  5,
					}
					meta.ToolErrorCounts = map[string]int{"Bash": 3}
					meta.ToolRejectCounts = map[string]int{"Write": 2}
				}),
			),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.tab = statsTabTools

	body := ansi.Strip(m.renderToolsTab(120))
	usageRow := findRenderedLine(t, body, "Tool Calls/Session")
	qualityRow := findRenderedLine(t, body, "Tool Error Rate")

	assert.GreaterOrEqual(t, strings.Index(usageRow, "Tool Calls/Session"), 0)
	assert.Contains(t, usageRow, "Top Tools")
	assert.Contains(t, usageRow, "│")
	usageSeparator := strings.Index(usageRow, "│")
	require.NotEqual(t, -1, usageSeparator)
	assert.GreaterOrEqual(t, lipgloss.Width(usageRow[:usageSeparator]), 55)
	assert.LessOrEqual(t, lipgloss.Width(usageRow[:usageSeparator]), 65)
	assert.Greater(t, strings.Index(usageRow, "Top Tools"), strings.Index(usageRow, "│"))

	assert.GreaterOrEqual(t, strings.Index(qualityRow, "Tool Error Rate"), 0)
	assert.Contains(t, qualityRow, "Rejected Suggestions")
	assert.Contains(t, qualityRow, "│")
	qualitySeparator := strings.Index(qualityRow, "│")
	require.NotEqual(t, -1, qualitySeparator)
	assert.GreaterOrEqual(t, lipgloss.Width(qualityRow[:qualitySeparator]), 55)
	assert.LessOrEqual(t, lipgloss.Width(qualityRow[:qualitySeparator]), 65)
}

func TestRenderToolsHistogramKeepsVisibleBarsWhenErrorRatesAreSparse(t *testing.T) {
	t.Parallel()

	buckets := []histBucket{
		{Label: "0-20", Count: 21},
		{Label: "21-50", Count: 41},
		{Label: "51-100", Count: 36},
		{Label: "101-200", Count: 36},
		{Label: "201+", Count: 48},
	}

	got := ansi.Strip(renderVerticalHistogram(
		"Tool Calls/Session",
		buckets,
		56,
		toolCallsChartHeight(0),
	))

	assert.Contains(t, got, "Tool Calls/Session")
	assert.Contains(t, got, "█")
}

func TestToolCallsChartHeightTracksToolErrorRateRows(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 3, toolCallsChartHeight(0))
	assert.Equal(t, 3, toolCallsChartHeight(2))
	assert.Equal(t, 3, toolCallsChartHeight(3))
	assert.Equal(t, 4, toolCallsChartHeight(6))
}

func TestStatsRenderToolsFilterKeepsHistogramBarsForCodexAndGPT54Slices(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	conversations := []conv.Conversation{
		testStatsConversationWithProviderAndSessions(
			conv.ProviderCodex,
			"codex-1",
			"alpha",
			testStatsSessionMeta("codex-1", "alpha", now, func(meta *conv.SessionMeta) {
				meta.Model = "gpt-5.4"
				meta.ToolCounts = map[string]int{"exec_command": 24}
				meta.ToolErrorCounts = nil
			}),
		),
		testStatsConversationWithProviderAndSessions(
			conv.ProviderClaude,
			"claude-1",
			"beta",
			testStatsSessionMeta("claude-1", "beta", now, func(meta *conv.SessionMeta) {
				meta.Model = "claude-opus-4-1"
				meta.ToolCounts = map[string]int{"Read": 8}
				meta.ToolErrorCounts = map[string]int{"Read": 3}
			}),
		),
	}

	tests := []struct {
		name   string
		filter browserFilterState
	}{
		{
			name: "provider codex",
			filter: func() browserFilterState {
				filter := newBrowserFilterState()
				filter.dimensions[filterDimProvider] = dimensionFilter{
					selected: map[string]bool{"Codex": true},
				}
				return filter
			}(),
		},
		{
			name: "model gpt-5.4",
			filter: func() browserFilterState {
				filter := newBrowserFilterState()
				filter.dimensions[filterDimModel] = dimensionFilter{
					selected: map[string]bool{"gpt-5.4": true},
				}
				return filter
			}(),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			m := newStatsModel(
				conversations,
				&fakeBrowserStore{},
				120,
				32,
				testCase.filter,
			)
			m.tab = statsTabTools

			body := ansi.Strip(m.renderToolsTab(120))
			usageLeft := renderedToolsUsageLeftColumn(t, body, 120)

			assert.Contains(t, usageLeft, "Tool Calls/Session")
			assert.Contains(t, usageLeft, "█")
		})
	}
}

func TestStatsFooterStatusRowShowsSessionCountAndScrollPercentWhenScrollable(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(80, 20)
	m.viewport.SetContent(strings.Repeat("line\n", 40))
	m.viewport.SetHeight(5)
	m.viewport.SetYOffset(6)

	row := ansi.Strip(m.footerStatusRow())

	assert.Contains(t, row, "[stats] 1 sessions")
	assert.Contains(t, row, "%")
}

func TestStatsFooterStatusRowHidesScrollPercentWhenContentFits(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(80, 20)
	m.viewport.SetContent("line\nline")
	m.viewport.SetHeight(5)

	row := ansi.Strip(m.footerStatusRow())

	assert.Contains(t, row, "[stats] 1 sessions")
	assert.NotContains(t, row, "%")
}

func TestStatsFooterStatusRowShowsDegradedBadge(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(80, 20)
	m.statsQueryFailures = statsQueryFailureDailyTokens

	row := ansi.Strip(m.footerStatusRow())

	assert.Contains(t, row, statsDegradedBadgeText)
	assert.Contains(t, row, "[stats] 1 sessions")
}

func TestStatsFooterHelpRowTracksActiveLaneActions(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(80, 20)

	row := ansi.Strip(m.footerHelpRow())
	assert.Contains(t, row, "h/l lane")
	assert.NotContains(t, row, "m metric")
	assert.NotContains(t, row, "enter open")

	m.overviewLaneCursor = 2
	row = ansi.Strip(m.footerHelpRow())
	assert.Contains(t, row, "m session")
	assert.Contains(t, row, "enter open")

	m.tab = statsTabActivity
	row = ansi.Strip(m.footerHelpRow())
	assert.Contains(t, row, "m metric")

	m.activityLaneCursor = 1
	row = ansi.Strip(m.footerHelpRow())
	assert.NotContains(t, row, "m metric")
}

func TestStatsFooterHelpRowShowsRebuildHintWhenDegraded(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(80, 20)
	m.statsQueryFailures = statsQueryFailureTurnMetrics

	row := ansi.Strip(m.footerHelpRow())

	assert.Contains(t, row, statsDegradedHintText)
	assert.Contains(t, row, "h/l lane")
}

func TestStatsFooterHelpRowPromptsToFixPerformanceScopeWhenGateIsActive(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(80, 20)
	m.tab = statsTabPerformance
	m.snapshot.Performance.Scope = statspkg.PerformanceScope{
		Providers:    []string{"Claude", "Codex"},
		Models:       []string{"claude-opus-4-1", "gpt-5.4"},
		SingleFamily: false,
	}

	row := ansi.Strip(m.footerHelpRow())

	assert.Contains(t, row, "f fix scope")
	assert.Contains(t, row, "need 1 provider + 1 model")
	assert.Contains(t, row, "h/l lane")
	assert.NotContains(t, row, "m metric")
}

func TestStatsFooterHelpRowPrefersRebuildHintOverScopeGateWhenDegraded(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(80, 20)
	m.tab = statsTabPerformance
	m.snapshot.Performance.Scope = statspkg.PerformanceScope{
		Providers:    []string{"Claude", "Codex"},
		Models:       []string{"claude-opus-4-1", "gpt-5.4"},
		SingleFamily: false,
	}
	m.statsQueryFailures = statsQueryFailurePerformanceSequence

	row := ansi.Strip(m.footerHelpRow())

	assert.Contains(t, row, statsDegradedHintText)
	assert.NotContains(t, row, "need 1 provider + 1 model")
}

func TestStatsBodyRowsUseStyledSideBorders(t *testing.T) {
	t.Parallel()

	border := lipgloss.NewStyle().Foreground(colorPrimary).Render("│")

	line := renderBodyLine("tabs", 8, colorPrimary)
	assert.True(t, strings.HasPrefix(line, border))
	assert.True(t, strings.HasSuffix(line, border))

	rows := renderBodyContent("alpha\nbeta", 8, 3, colorPrimary)
	assert.Len(t, rows, 3)
	for _, row := range rows {
		assert.True(t, strings.HasPrefix(row, border))
		assert.True(t, strings.HasSuffix(row, border))
	}
}

func TestRenderStatsTitleUsesPrimaryStyle(t *testing.T) {
	t.Parallel()

	title := renderStatsTitle("Tokens by Model")

	assert.Equal(t, "Tokens by Model", ansi.Strip(title))
	assert.NotEqual(t, "Tokens by Model", title)
}

func TestRenderStatsLanePairUsesBorderedCardsAndAlignedHeights(t *testing.T) {
	t.Parallel()

	rendered := renderStatsLanePair(
		80,
		30,
		"Left Lane",
		true,
		func(width int) string {
			return fitToWidth("short", width)
		},
		"Right Lane",
		false,
		func(width int) string {
			return strings.Join([]string{
				fitToWidth("first", width),
				fitToWidth("second", width),
				fitToWidth("third", width),
			}, "\n")
		},
	)
	stripped := ansi.Strip(rendered)
	titleLine := findRenderedLine(t, stripped, "Left Lane")

	assert.Contains(t, titleLine, "▸ Left Lane")
	assert.Contains(t, titleLine, "Right Lane")
	assert.Contains(t, titleLine, "│")

	lines := strings.Split(stripped, "\n")
	require.Len(t, lines, 5)
	for _, line := range lines {
		assert.Equal(t, 80, lipgloss.Width(line))
	}
}

func TestStatsRenderOverviewUsesBorderedLaneCards(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)

	body := ansi.Strip(m.renderOverviewTab(120))
	titleLine := findRenderedLine(t, body, "Tokens by Model")
	tableLine := findRenderedLine(t, body, "Most Token-Heavy Sessions")

	assert.Contains(t, titleLine, "▸ Tokens by Model")
	assert.Contains(t, titleLine, "Tokens by Project")
	assert.Contains(t, titleLine, "╭")
	assert.Contains(t, titleLine, "│")
	assert.True(t, strings.HasPrefix(strings.TrimSpace(tableLine), "╭"))
}

func TestStatsRenderActivityUsesBorderedLaneCards(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabActivity

	body := ansi.Strip(m.renderActivityTab(120, 25))
	dailyLine := findRenderedLine(t, body, "Daily Sessions")
	heatmapLine := findRenderedLine(t, body, "Activity Heatmap")

	assert.True(t, strings.HasPrefix(strings.TrimSpace(dailyLine), "╭"))
	assert.Contains(t, dailyLine, "▸ Daily Sessions")
	assert.True(t, strings.HasPrefix(strings.TrimSpace(heatmapLine), "╭"))
}

func TestStatsRenderSessionsShowsBorderedTurnMetricCards(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabSessions
	m.sessionsLaneCursor = 2
	m.snapshot.Sessions.ClaudeTurnMetrics = []statspkg.PositionTokenMetrics{{
		Position:           1,
		AverageInputTokens: 120,
		AverageTurnTokens:  180,
		SampleCount:        3,
	}}

	body := ansi.Strip(m.renderSessionsTab(120))
	histogramLine := findRenderedLine(t, body, "Session Duration")
	turnMetricLine := findRenderedLine(t, body, statsClaudeContextGrowthTitle)

	assert.Contains(t, histogramLine, "╭")
	assert.Contains(t, histogramLine, "Messages per Session")
	assert.Contains(t, turnMetricLine, "▸ "+statsClaudeContextGrowthTitle)
	assert.Contains(t, turnMetricLine, statsClaudeTurnCostTitle)
	assert.NotContains(t, body, "Computing turn charts...")
}

func TestStatsRenderToolsUsesBorderedLaneCards(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabTools
	m.toolsLaneCursor = 1

	body := ansi.Strip(m.renderToolsTab(120))
	usageLine := findRenderedLine(t, body, "Top Tools")
	qualityLine := findRenderedLine(t, body, "Tool Error Rate")

	assert.Contains(t, usageLine, "Tool Calls/Session")
	assert.Contains(t, usageLine, "▸ Top Tools")
	assert.Contains(t, usageLine, "╭")
	assert.Contains(t, usageLine, "│")
	assert.Contains(t, qualityLine, statsRejectedSuggestionsTitle)
	assert.Contains(t, qualityLine, "╭")
}

func newStatsRenderModel(width, height int) statsModel {
	return newStatsModel(
		[]conv.Conversation{
			testStatsConversation("stats-1", "alpha", time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)),
		},
		&fakeBrowserStore{},
		width,
		height,
		newBrowserFilterState(),
	)
}

func findRenderedLine(tb testing.TB, content, needle string) string {
	tb.Helper()

	for line := range strings.SplitSeq(content, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}

	tb.Fatalf("line containing %q not found", needle)
	return ""
}

func renderedToolsUsageLeftColumn(tb testing.TB, content string, width int) string {
	tb.Helper()

	sections := strings.Split(content, "\n\n")
	if len(sections) < 2 {
		tb.Fatalf("usage section not found")
	}

	_, leftWidth, stacked := statsColumnWidths(width, 1, 1, 30)
	if stacked {
		tb.Fatalf("usage section stacked unexpectedly")
	}

	lines := strings.Split(sections[1], "\n")
	leftLines := make([]string, 0, len(lines))
	for _, line := range lines {
		runes := []rune(line)
		if len(runes) > leftWidth {
			leftLines = append(leftLines, strings.TrimRight(string(runes[:leftWidth]), " "))
			continue
		}
		leftLines = append(leftLines, strings.TrimRight(line, " "))
	}

	return strings.Join(leftLines, "\n")
}
