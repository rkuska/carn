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
