package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestStatsHelpOverviewShowsOverviewCharts(t *testing.T) {
	t.Parallel()

	m := newStatsHelpModel()

	charts := findHelpSectionByTitle(t, m.helpSections(), "Charts")

	assert.Equal(t, []string{
		"Tokens by Model",
		"Tokens by Project",
		"Most Token-Heavy Sessions",
	}, helpItemKeys(charts.items))
	assert.Contains(t, helpItemDetail(t, charts, "Tokens by Model"), "driving token use")
}

func TestStatsHelpActivityNavigationIncludesMetricShortcut(t *testing.T) {
	t.Parallel()

	m := newStatsHelpModel()
	m.tab = statsTabActivity

	navigation := findHelpSectionByTitle(t, m.helpSections(), "Navigation")

	assert.Contains(t, helpItemKeys(navigation.items), "m")
	assert.Contains(t, helpItemDetail(t, navigation, "m"), "cycle the daily chart")
}

func TestStatsHelpShowsTabSpecificChartDescriptions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		tab  statsTab
		want []string
	}{
		{
			name: "sessions",
			tab:  statsTabSessions,
			want: []string{"Session Duration", "Messages per Session", "Context Growth", "Turn Cost"},
		},
		{
			name: "tools",
			tab:  statsTabTools,
			want: []string{"Top Tools", "Tool Calls/Session", "Tool Error Rate", "Rejected Suggestions"},
		},
		{
			name: "performance",
			tab:  statsTabPerformance,
			want: []string{"Lane Cards", "Detailed Trends", "Diagnostics"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			m := newStatsHelpModel()
			m.tab = testCase.tab

			charts := findHelpSectionByTitle(t, m.helpSections(), "Charts")
			assert.Equal(t, testCase.want, helpItemKeys(charts.items))
		})
	}
}

func TestStatsHelpNavigationSectionIsConsistentAcrossTabs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		tab  statsTab
		want []string
	}{
		{
			name: "overview",
			tab:  statsTabOverview,
			want: []string{"ctrl+f/b", "r", "f", "j/k", "g/G", "?", "q/esc", "1-5"},
		},
		{
			name: "activity",
			tab:  statsTabActivity,
			want: []string{"ctrl+f/b", "r", "f", "j/k", "g/G", "?", "q/esc", "m"},
		},
		{
			name: "sessions",
			tab:  statsTabSessions,
			want: []string{"ctrl+f/b", "r", "f", "j/k", "g/G", "?", "q/esc"},
		},
		{
			name: "tools",
			tab:  statsTabTools,
			want: []string{"ctrl+f/b", "r", "f", "j/k", "g/G", "?", "q/esc"},
		},
		{
			name: "performance",
			tab:  statsTabPerformance,
			want: []string{"ctrl+f/b", "r", "f", "j/k", "g/G", "?", "q/esc"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			m := newStatsHelpModel()
			m.tab = testCase.tab

			navigation := findHelpSectionByTitle(t, m.helpSections(), "Navigation")
			assert.Equal(t, testCase.want, helpItemKeys(navigation.items))
		})
	}
}

func newStatsHelpModel() statsModel {
	return newStatsModel(
		[]conv.Conversation{
			testStatsConversation("stats-1", "alpha", time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
}

func findHelpSectionByTitle(tb testing.TB, sections []helpSection, title string) helpSection {
	tb.Helper()

	for _, section := range sections {
		if section.title == title {
			return section
		}
	}

	tb.Fatalf("section %q not found", title)
	return helpSection{}
}
