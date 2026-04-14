package app

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

	body := ansi.Strip(m.renderOverviewTab(120))
	assert.Contains(t, body, "Metric detail")
	assert.Contains(t, body, "Tokens by Model")
	assert.Contains(t, body, "leading model")

	m.overviewLaneCursor = 1
	body = ansi.Strip(m.renderOverviewTab(120))
	assert.Contains(t, body, "Tokens by Project")
	assert.Contains(t, body, "leading project")

	m.overviewLaneCursor = 2
	body = ansi.Strip(m.renderOverviewTab(120))
	assert.Contains(t, body, "Tokens by (Provider, Version)")
	assert.Contains(t, body, "provider/version")

	m.overviewLaneCursor = 3
	body = ansi.Strip(m.renderOverviewTab(120))
	assert.Contains(t, body, "Most Token-Heavy Sessions")
	assert.Contains(t, body, "Enter opens the selected session")
}

func TestStatsRenderActivityMetricDetailFollowsSelectedLane(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabActivity

	body := ansi.Strip(m.renderActivityTab(120, 25))
	assert.Contains(t, body, "Metric detail")
	assert.Contains(t, body, "Daily Activity")
	assert.Contains(t, body, "peak day")

	m.activityLaneCursor = 1
	body = ansi.Strip(m.renderActivityTab(120, 25))
	assert.Contains(t, body, "Activity Heatmap")
	assert.Contains(t, body, "busiest slot")
}

func TestStatsRenderSessionsMetricDetailFollowsSelectedLane(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabSessions

	body := ansi.Strip(m.renderSessionsTab(120))
	assert.Contains(t, body, "Session Duration")
	assert.Contains(t, body, "dominant bucket")

	m.sessionsLaneCursor = 2
	body = ansi.Strip(m.renderSessionsTab(120))
	assert.Contains(t, body, "Prompt Growth")
	assert.Contains(t, body, "prompt multiplier")
	assert.Contains(t, body, "main-thread user turn number")
}

func TestStatsRenderToolsMetricDetailFollowsSelectedLane(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 32)
	m.tab = statsTabTools

	body := ansi.Strip(m.renderToolsTab(120))
	assert.Contains(t, body, "Tool Calls/Session")
	assert.Contains(t, body, "dominant bucket")

	m.toolsLaneCursor = 2
	body = ansi.Strip(m.renderToolsTab(120))
	assert.Contains(t, body, "Tool Error Rate")
	assert.Contains(t, body, "top rate")
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
	assert.Contains(t, body, "Performance preview")
	assert.Contains(t, body, "Metric detail")
	assert.Contains(t, body, "Outcome")

	m.performanceLaneCursor = 2
	body = ansi.Strip(m.renderPerformanceTab(120))
	assert.Contains(t, body, "Efficiency")
}
