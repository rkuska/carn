package stats

import (
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestStatsRenderSessionsGroupedTurnChartsShowVersionLegend(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	store := &fakeBrowserStore{
		turnMetricRows: testClaudeVersionTurnMetricRows(now, 220, 270),
	}

	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-1",
				"alpha",
				testStatsSessionMeta("stats-1", "alpha", now),
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
	m.sessionsLaneCursor = 2
	m.splitBy = statspkg.SplitDimensionVersion
	m.filter.Dimensions[filterDimVersion] = dimensionFilter{
		Selected: map[string]bool{"1.0.0": true, statspkg.UnknownVersionLabel: true},
	}
	m = m.applyFilterChange()

	body := ansi.Strip(m.renderSessionsTab(180))

	assert.Contains(t, body, "Avg Prompt Growth (by Version)")
	assert.Contains(t, body, "Avg Billed Tokens per Turn (by Version)")
	assert.Contains(t, body, "1.0.0")
	assert.Contains(t, body, statspkg.UnknownVersionLabel)
}

func TestStatsRenderSplitSessionsHistogramLanesShowUnsupportedPlaceholders(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-1",
				"alpha",
				testStatsSessionMeta("stats-1", "alpha", now),
			),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.tab = statsTabSessions
	m.splitBy = statspkg.SplitDimensionVersion
	m = m.applyFilterChange()

	body := ansi.Strip(m.renderSessionsTab(120))

	assert.Contains(t, body, "Split by Version is not available for session duration.")
	assert.Contains(t, body, "Split by Version is not available for messages per sessi")
}

func TestStatsRenderSplitSessionsTurnLanesShowUnsupportedPlaceholderForModel(t *testing.T) {
	t.Parallel()

	const claudeOpusModel = "claude-opus-4-1"

	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-1",
				"alpha",
				testStatsSessionMeta("stats-1", "alpha", now, func(meta *conv.SessionMeta) {
					meta.Model = claudeOpusModel
				}),
			),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.tab = statsTabSessions
	m.splitBy = statspkg.SplitDimensionModel
	m = m.applyFilterChange()

	body := ansi.Strip(m.renderSessionsTab(120))

	assert.Contains(t, body, "Split by Model is not available for turn metrics.")
}
