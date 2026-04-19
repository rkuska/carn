package stats

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestSplitDimensionSupportsTab(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tab  statsTab
		want bool
	}{
		{statsTabOverview, false},
		{statsTabActivity, true},
		{statsTabSessions, true},
		{statsTabTools, true},
		{statsTabCache, true},
		{statsTabPerformance, false},
	}
	for _, tc := range cases {
		t.Run(tabLabel(tc.tab), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, splitDimensionSupportsTab(tc.tab))
		})
	}
}

func TestStatsModelSplitActiveRequiresActiveDimAndSupportedTab(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("split-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)

	assert.False(t, m.splitActive())

	m.splitBy = statspkg.SplitDimensionVersion
	m.tab = statsTabSessions
	assert.True(t, m.splitActive())

	m.tab = statsTabOverview
	assert.False(t, m.splitActive())
}

func TestStatsFilterOverlayShowsSplitRow(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("split-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m = m.openFilterOverlay()

	view := ansi.Strip(m.renderStatsFilterOverlay())
	assert.Contains(t, view, "Split by")
	assert.Contains(t, view, "off")
}

func TestStatsFilterSplitRowSpaceOpensOptions(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("split-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m = m.openFilterOverlay()
	m.filter.Cursor = splitRowCursor

	m, _ = m.Update(tea.KeyPressMsg{Text: " "})

	assert.True(t, m.splitExpanded)

	view := ansi.Strip(m.renderStatsFilterOverlay())
	assert.Contains(t, view, "Provider")
	assert.Contains(t, view, "Version")
	assert.Contains(t, view, "Model")
	assert.Contains(t, view, "Project")
}

func TestStatsFilterSplitOptionSpaceSelectsAndReplaces(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("split-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m = m.openFilterOverlay()
	m.filter.Cursor = splitRowCursor
	m.splitExpanded = true

	m.splitExpandedCursor = currentSplitOptionIndex(statspkg.SplitDimensionVersion)
	m, _ = m.Update(tea.KeyPressMsg{Text: " "})
	assert.Equal(t, statspkg.SplitDimensionVersion, m.splitBy)

	m.splitExpandedCursor = currentSplitOptionIndex(statspkg.SplitDimensionProvider)
	m, _ = m.Update(tea.KeyPressMsg{Text: " "})
	assert.Equal(t, statspkg.SplitDimensionProvider, m.splitBy)

	m, _ = m.Update(tea.KeyPressMsg{Text: " "})
	assert.Equal(t, statspkg.SplitDimensionNone, m.splitBy)
}

func TestStatsFilterSplitClearWithX(t *testing.T) {
	t.Parallel()

	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("split-1", "alpha", time.Now())},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.splitBy = statspkg.SplitDimensionVersion
	m = m.openFilterOverlay()
	m.filter.Cursor = splitRowCursor

	m, _ = m.Update(tea.KeyPressMsg{Text: "x"})
	assert.Equal(t, statspkg.SplitDimensionNone, m.splitBy)
}

func TestStatsRenderUnsupportedTabShowsPlaceholder(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	m := newStatsModel(
		[]conv.Conversation{testStatsConversation("split-1", "alpha", now)},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.tab = statsTabPerformance
	m.splitBy = statspkg.SplitDimensionVersion
	m = m.applyFilterChange()

	body := ansi.Strip(m.renderActiveTab())
	assert.Contains(t, body, "Split by Version is not supported on the Performance tab.")
}

func TestPresentSplitKeysReturnsOnlyKeysWithNonZeroValues(t *testing.T) {
	t.Parallel()

	buckets := []statspkg.SplitHistogramBucket{
		{Label: "0-20", Total: 3, Splits: []statspkg.SplitValue{
			{Key: "1.0.0", Value: 2},
			{Key: "2.0.0", Value: 1},
		}},
		{Label: "21-50", Total: 0, Splits: nil},
	}

	keys := presentSplitKeys(buckets, histBucketSplits)

	assert.Equal(t, []string{"1.0.0", "2.0.0"}, keys)
}

func TestPresentSplitKeysSkipsKeysWithZeroOrNegativeValues(t *testing.T) {
	t.Parallel()

	stats := []statspkg.SplitNamedStat{
		{Name: "Read", Total: 10, Splits: []statspkg.SplitValue{
			{Key: "1.0.0", Value: 10},
			{Key: "2.0.0", Value: 0},
		}},
		{Name: "Write", Total: 5, Splits: []statspkg.SplitValue{
			{Key: "1.0.0", Value: 5},
		}},
	}

	keys := presentSplitKeys(stats, namedStatSplits)

	assert.Equal(t, []string{"1.0.0"}, keys)
}

func TestRenderSplitToolsBodyLegendOnlyListsKeysPresentInData(t *testing.T) {
	t.Parallel()

	const versionWithData = "1.0.0"
	const versionWithoutData = "9.9.9"

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	conversation := testStatsConversationWithProviderAndSessions(
		conv.ProviderClaude,
		"split-1",
		"alpha",
		testStatsSessionMeta("split-1", "alpha", now, func(meta *conv.SessionMeta) {
			meta.Provider = conv.ProviderClaude
			meta.Version = versionWithData
			meta.ToolCounts = map[string]int{"Read": 10}
		}),
	)

	m := newStatsModel(
		[]conv.Conversation{conversation},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.tab = statsTabTools
	m.splitBy = statspkg.SplitDimensionVersion
	// User selected versionWithoutData even though no session exists for it.
	// The legend must reflect the chart contents, not the filter selection.
	m.filter.Dimensions[filterDimVersion] = dimensionFilter{
		Selected: map[string]bool{versionWithData: true, versionWithoutData: true},
	}
	m = m.applyFilterChange()

	body := ansi.Strip(m.renderSplitToolsBody(120))

	assert.Contains(t, body, versionWithData, "legend should show keys with data")
	assert.NotContains(t, body, versionWithoutData, "legend should hide keys with no data")
}

func TestMonotonicScaledHeightDistinguishesAdjacentSmallValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		value, maxValue  float64
		height, wantSlot int
	}{
		{"zero stays zero", 0, 8, 5, 0},
		{"one of eight at height 5", 1, 8, 5, 1},
		{"two of eight at height 5", 2, 8, 5, 2},
		{"four of eight at height 5", 4, 8, 5, 3},
		{"eight of eight at height 5", 8, 8, 5, 5},
		{"tiny non-zero gets one row", 1, 1000, 5, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.wantSlot, el.MonotonicScaledHeight(tc.value, tc.maxValue, tc.height))
		})
	}
}

func TestMonotonicScaledHeightIsMonotonic(t *testing.T) {
	t.Parallel()

	// Skewed dataset: the first few buckets dominate the max while later
	// buckets have very small counts. Round-to-nearest used to collapse two
	// adjacent small buckets onto the same row; the monotonic helper now
	// preserves the ordering whenever the resolution allows.
	values := []float64{8, 4, 2, 1}
	maxValue := 8.0
	height := 5
	heights := make([]int, len(values))
	for i, v := range values {
		heights[i] = el.MonotonicScaledHeight(v, maxValue, height)
	}
	assert.Equal(t, []int{5, 3, 2, 1}, heights)
	for i := 1; i < len(heights); i++ {
		assert.Less(t, heights[i], heights[i-1],
			"each smaller bucket should map to a strictly smaller row count",
		)
	}
}

func TestSplitSessionsLaneSupportsMetricCycling(t *testing.T) {
	t.Parallel()

	const version = "1.0.0"

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	conversation := testStatsConversationWithProviderAndSessions(
		conv.ProviderClaude,
		"split-1",
		"alpha",
		testStatsSessionMeta("split-1", "alpha", now, func(meta *conv.SessionMeta) {
			meta.Provider = conv.ProviderClaude
			meta.Version = version
		}),
	)
	m := newStatsModel(
		[]conv.Conversation{conversation},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.tab = statsTabSessions
	m.splitBy = statspkg.SplitDimensionVersion
	m = m.applyFilterChange()

	m.sessionsLaneCursor = 2 // statsLaneSessionsContext
	require.True(t, m.activeLaneSupportsMetric(),
		"split mode must keep the metric cycling action enabled for sessions lanes",
	)

	before := m.sessionsPromptMode
	m, _ = m.Update(tea.KeyPressMsg{Text: "m"})
	assert.NotEqual(t, before, m.sessionsPromptMode,
		"pressing m should advance the prompt mode even when split is active",
	)
}

func TestSplitTurnSeriesPicksUpStatisticMode(t *testing.T) {
	t.Parallel()

	const version = "1.0.0"

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	store := &fakeBrowserStore{}
	conversation := testStatsConversationWithProviderAndSessions(
		conv.ProviderClaude,
		"split-1",
		"alpha",
		testStatsSessionMeta("split-1", "alpha", now, func(meta *conv.SessionMeta) {
			meta.Provider = conv.ProviderClaude
			meta.Version = version
		}),
	)
	store.turnMetricRowsByKey = map[string][]conv.SessionTurnMetrics{
		conversation.CacheKey(): {
			{
				Provider:  conv.ProviderClaude,
				Version:   version,
				Timestamp: now.Add(-time.Hour),
				Turns:     []conv.TurnTokens{{PromptTokens: 100, TurnTokens: 100}},
			},
			{
				Provider:  conv.ProviderClaude,
				Version:   version,
				Timestamp: now.Add(-2 * time.Hour),
				Turns:     []conv.TurnTokens{{PromptTokens: 300, TurnTokens: 300}},
			},
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
	m.tab = statsTabSessions
	m.splitBy = statspkg.SplitDimensionVersion
	m = m.applyFilterChange()

	avgSeries := m.splitTurnSeries(statspkg.StatisticModeAverage)
	maxSeries := m.splitTurnSeries(statspkg.StatisticModeMax)

	require.Len(t, avgSeries, 1)
	require.Len(t, maxSeries, 1)
	assert.InDelta(t, 200.0, avgSeries[0].Metrics[0].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 300.0, maxSeries[0].Metrics[0].AveragePromptTokens, 0.0001)
}

func TestSessionTurnLaneTitleKeepsBadgeAsPrefixWithSplit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		split string
		mode  statspkg.StatisticMode
		want  string
	}{
		{"non-split avg", "", statspkg.StatisticModeAverage, "Avg Prompt Growth"},
		{"non-split p99", "", statspkg.StatisticModeP99, "p99 Prompt Growth"},
		{"split avg", "Version", statspkg.StatisticModeAverage, "Avg Prompt Growth (by Version)"},
		{"split p99", "Version", statspkg.StatisticModeP99, "p99 Prompt Growth (by Version)"},
		{"split max", "Provider", statspkg.StatisticModeMax, "max Prompt Growth (by Provider)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, buildSessionTurnLaneTitle("Prompt Growth", tc.mode, tc.split))
		})
	}
}

func TestSplitMetricDetailExplainsGroupedTurnBehavior(t *testing.T) {
	t.Parallel()

	const version = "1.0.0"

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	store := &fakeBrowserStore{
		turnMetricRowsByKey: map[string][]conv.SessionTurnMetrics{},
	}
	conversation := testStatsConversationWithProviderAndSessions(
		conv.ProviderClaude,
		"split-1",
		"alpha",
		testStatsSessionMeta("split-1", "alpha", now, func(meta *conv.SessionMeta) {
			meta.Provider = conv.ProviderClaude
			meta.Version = version
		}),
	)
	store.turnMetricRowsByKey[conversation.CacheKey()] = []conv.SessionTurnMetrics{
		{
			Provider: conv.ProviderClaude, Version: version, Timestamp: now.Add(-time.Hour),
			Turns: []conv.TurnTokens{{PromptTokens: 100, TurnTokens: 200}},
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
	m.tab = statsTabSessions
	m.splitBy = statspkg.SplitDimensionVersion
	m.sessionsTurnCostMode = statspkg.StatisticModeP99
	m.sessionsLaneCursor = 3 // statsLaneSessionsTurnCost
	m = m.applyFilterChange()

	body := ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, body, "The X-axis is main-thread user turn bucket",
		"split detail must describe grouped turn buckets",
	)
	assert.Contains(t, body, "p99 total assistant tokens per turn",
		"split detail must include the cycled statistic mode in the Y-axis description",
	)
	assert.Contains(t, body, "groups only split series with values",
		"split detail must explain grouped split behavior",
	)
}

func TestStatsModelSplitKeysReflectsSelectedFilterValues(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	conversation := testStatsConversationWithProviderAndSessions(
		conv.ProviderClaude,
		"split-1",
		"alpha",
		testStatsSessionMeta("split-1", "alpha", now, func(meta *conv.SessionMeta) {
			meta.Provider = conv.ProviderClaude
			meta.Version = "1.0.0"
		}),
		testStatsSessionMeta("split-2", "alpha", now.Add(-time.Hour), func(meta *conv.SessionMeta) {
			meta.Provider = conv.ProviderClaude
			meta.Version = "2.0.0"
		}),
	)
	m := newStatsModel(
		[]conv.Conversation{conversation},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.splitBy = statspkg.SplitDimensionVersion
	m = m.applyFilterChange()

	require.Equal(t, []string{"1.0.0", "2.0.0"}, m.splitKeys())

	m.filter.Dimensions[filterDimVersion] = dimensionFilter{
		Selected: map[string]bool{"2.0.0": true},
	}
	m = m.applyFilterChange()
	require.Equal(t, []string{"2.0.0"}, m.splitKeys())
}
