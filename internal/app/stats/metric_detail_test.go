package stats

import (
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestStatsRenderOverviewIncludesSelectedLaneMetricDetail(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)

	detail := ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "Tokens by Model")
	assert.Contains(t, detail, "leading model")

	m.overviewLaneCursor = 1
	detail = ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "Tokens by Project")
	assert.Contains(t, detail, "leading project")

	m.overviewLaneCursor = 2
	detail = ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "Tokens by (Provider, Version)")
	assert.Contains(t, detail, "provider/version")

	m.overviewLaneCursor = 3
	detail = ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "Most Token-Heavy Sessions")
	assert.Contains(t, detail, "Enter opens the selected session")
}

func TestStatsRenderActivityMetricDetailFollowsSelectedLane(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabActivity

	detail := ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "Daily Activity")
	assert.Contains(t, detail, "peak day")

	m.activityLaneCursor = 1
	detail = ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "Activity Heatmap")
	assert.Contains(t, detail, "busiest slot")
}

func TestStatsRenderSessionsMetricDetailFollowsSelectedLane(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabSessions

	detail := ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "Session Duration")
	assert.Contains(t, detail, "dominant bucket")

	m.sessionsLaneCursor = 2
	detail = ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "Prompt Growth")
	assert.Contains(t, detail, "prompt multiplier")
	assert.Contains(t, detail, "main-thread user turn number")
}

func TestStatsRenderToolsMetricDetailFollowsSelectedLane(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabTools

	detail := ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "Tool Calls/Session")
	assert.Contains(t, detail, "dominant bucket")

	m.toolsLaneCursor = 2
	detail = ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "Tool Error Rate")
	assert.Contains(t, detail, "top rate")
}

func TestStatsRenderMetricDetailWrapsLongReadingText(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(72, 32)
	m.tab = statsTabSessions
	m.sessionsLaneCursor = 2
	m.snapshot.Sessions.ClaudeTurnMetrics = []statspkg.PositionTokenMetrics{{
		Position:            1,
		AveragePromptTokens: 120,
		AverageTurnTokens:   180,
		SampleCount:         3,
	}}

	detail := ansi.Strip(m.renderActiveMetricDetail(72))

	assert.Contains(t, detail, "main-thread user turn")
	assert.NotContains(t, detail, "...")
}

func TestStatsRenderCacheDailyMetricDetailOmitsToggleCopy(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabCache

	body := ansi.Strip(m.renderCacheTab(120, 32))

	assert.NotContains(t, body, "Press m to toggle")
}

func TestStatsRenderPerformanceScopeGateIncludesMetricDetail(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabPerformance
	m.snapshot.Performance = statspkg.Performance{
		Scope: statspkg.PerformanceScope{
			SessionCount:         12,
			BaselineSessionCount: 10,
			Providers:            []string{"Claude", "Codex"},
			Models:               []string{"claude-opus-4-1", "gpt-5.4"},
			CurrentRange: statspkg.TimeRange{
				Start: time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC),
				End:   time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC),
			},
			BaselineRange: statspkg.TimeRange{
				Start: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
				End:   time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	body := ansi.Strip(m.renderPerformanceTab(120))
	detail := ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, body, "Performance preview")
	assert.Contains(t, body, "Outcome")
	assert.Contains(t, detail, "Outcome")

	m.performanceLaneCursor = 2
	detail = ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "Efficiency")
}
