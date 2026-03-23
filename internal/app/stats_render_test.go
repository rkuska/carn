package app

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

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
	t.Parallel()

	restoreNow := setStatsNowForTest(func() time.Time {
		return time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	})
	defer restoreNow()

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
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
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

	assert.Equal(t, 0, strings.Index(usageRow, "Tool Calls/Session"))
	assert.Contains(t, usageRow, "Top Tools")
	assert.Contains(t, usageRow, "│")
	assert.GreaterOrEqual(t, strings.Index(usageRow, "│"), 55)
	assert.LessOrEqual(t, strings.Index(usageRow, "│"), 65)
	assert.Greater(t, strings.Index(usageRow, "Top Tools"), strings.Index(usageRow, "│"))

	assert.Equal(t, 0, strings.Index(qualityRow, "Tool Error Rate"))
	assert.Contains(t, qualityRow, "Rejected Suggestions")
	assert.Contains(t, qualityRow, "│")
	assert.GreaterOrEqual(t, strings.Index(qualityRow, "│"), 55)
	assert.LessOrEqual(t, strings.Index(qualityRow, "│"), 65)
}

func TestToolCallsChartHeightTracksToolErrorRateRows(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, toolCallsChartHeight(0))
	assert.Equal(t, 1, toolCallsChartHeight(2))
	assert.Equal(t, 1, toolCallsChartHeight(3))
	assert.Equal(t, 4, toolCallsChartHeight(6))
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

func TestStatsFooterHelpRowShowsMetricOnlyOnActivityTab(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(80, 20)

	assert.NotContains(t, ansi.Strip(m.footerHelpRow()), "metric")

	m.tab = statsTabActivity
	assert.Contains(t, ansi.Strip(m.footerHelpRow()), "metric")
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
