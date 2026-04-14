package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestStatsHelpSectionsShowNavigationOnly(t *testing.T) {
	t.Parallel()

	m := newStatsHelpModel()

	sections := m.helpSections()

	assert.Len(t, sections, 1)
	assert.Equal(t, "Navigation", sections[0].title)
}

func TestStatsHelpNavigationTracksActiveLaneActions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		mut  func(statsModel) statsModel
		want []string
	}{
		{
			name: "overview default lane",
			mut: func(m statsModel) statsModel {
				return m
			},
			want: []string{"ctrl+f/b", "r", "f", "h/l", "?", "q/esc"},
		},
		{
			name: "overview project lane",
			mut: func(m statsModel) statsModel {
				m.overviewLaneCursor = 1
				return m
			},
			want: []string{"ctrl+f/b", "r", "f", "h/l", "?", "q/esc"},
		},
		{
			name: "overview provider version lane",
			mut: func(m statsModel) statsModel {
				m.overviewLaneCursor = 2
				return m
			},
			want: []string{"ctrl+f/b", "r", "f", "h/l", "?", "q/esc"},
		},
		{
			name: "overview top sessions lane",
			mut: func(m statsModel) statsModel {
				m.overviewLaneCursor = 3
				return m
			},
			want: []string{"ctrl+f/b", "r", "f", "h/l", "m", "enter", "?", "q/esc"},
		},
		{
			name: "activity daily lane",
			mut: func(m statsModel) statsModel {
				m.tab = statsTabActivity
				return m
			},
			want: []string{"ctrl+f/b", "r", "f", "h/l", "m", "?", "q/esc"},
		},
		{
			name: "activity heatmap lane",
			mut: func(m statsModel) statsModel {
				m.tab = statsTabActivity
				m.activityLaneCursor = 1
				return m
			},
			want: []string{"ctrl+f/b", "r", "f", "h/l", "?", "q/esc"},
		},
		{
			name: "sessions",
			mut: func(m statsModel) statsModel {
				m.tab = statsTabSessions
				return m
			},
			want: []string{"ctrl+f/b", "r", "f", "h/l", "v", "?", "q/esc"},
		},
		{
			name: "sessions grouped lane",
			mut: func(m statsModel) statsModel {
				m.tab = statsTabSessions
				m.sessionsLaneCursor = 2
				return m
			},
			want: []string{"ctrl+f/b", "r", "f", "h/l", "v", "?", "q/esc"},
		},
		{
			name: "tools",
			mut: func(m statsModel) statsModel {
				m.tab = statsTabTools
				return m
			},
			want: []string{"ctrl+f/b", "r", "f", "h/l", "v", "?", "q/esc"},
		},
		{
			name: "cache",
			mut: func(m statsModel) statsModel {
				m.tab = statsTabCache
				return m
			},
			want: []string{"ctrl+f/b", "r", "f", "h/l", "v", "m", "?", "q/esc"},
		},
		{
			name: "performance",
			mut: func(m statsModel) statsModel {
				m.tab = statsTabPerformance
				m.snapshot.Performance = testRenderedPerformance(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))
				return m
			},
			want: []string{"ctrl+f/b", "r", "f", "h/l", "m", "?", "q/esc"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			m := testCase.mut(newStatsHelpModel())

			navigation := findHelpSectionByTitle(t, m.helpSections(), "Navigation")
			assert.Equal(t, testCase.want, helpItemKeys(navigation.items))
		})
	}
}

func TestStatsHelpNavigationShowsScrollShortcutsOnlyWhenScrollable(t *testing.T) {
	t.Parallel()

	m := newStatsHelpModel()

	navigation := findHelpSectionByTitle(t, m.helpSections(), "Navigation")
	assert.NotContains(t, helpItemKeys(navigation.items), "j/k")
	assert.NotContains(t, helpItemKeys(navigation.items), "g/G")

	m.viewport.SetContent("line\nline\nline\nline\nline\nline\nline\nline\nline\nline")
	m.viewport.SetHeight(3)

	navigation = findHelpSectionByTitle(t, m.helpSections(), "Navigation")
	assert.Contains(t, helpItemKeys(navigation.items), "j/k")
	assert.Contains(t, helpItemKeys(navigation.items), "g/G")
}

func TestStatsHelpNavigationShowsFixScopeForPerformanceGate(t *testing.T) {
	t.Parallel()

	m := newStatsHelpModel()
	m.tab = statsTabPerformance
	m.snapshot.Performance.Scope = statspkg.PerformanceScope{
		Providers:    []string{"Claude", "Codex"},
		Models:       []string{"claude-opus-4-1", "gpt-5.4"},
		SingleFamily: false,
	}

	navigation := findHelpSectionByTitle(t, m.helpSections(), "Navigation")

	assert.Contains(t, helpItemKeys(navigation.items), "h/l")
	assert.NotContains(t, helpItemKeys(navigation.items), "m")
	assert.Equal(t, "fix scope", navigation.items[2].desc)
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
