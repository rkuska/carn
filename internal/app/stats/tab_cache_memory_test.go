package stats

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestRenderCacheTabIncludesFirstTurnLaneWithPopulatedData(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 40)
	m.tab = statsTabCache
	m.snapshot.Cache.FirstTurnByVersion = []statspkg.CacheFirstTurnVersionStat{
		{Version: "1.0.0", SessionCount: 5, ZeroReadRate: 0.6, MedianFirstRead: 0},
		{Version: "1.1.0", SessionCount: 4, ZeroReadRate: 0.25, MedianFirstRead: 500},
	}

	body := ansi.Strip(m.renderCacheTab(120, 40))
	assert.Contains(t, body, "First-Turn Cold Cache by Version (Claude)")
	assert.Contains(t, body, "1.0.0")
	assert.Contains(t, body, "1.1.0")
}

func TestRenderCacheTabFirstTurnLaneShowsNoDataWhenEmpty(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 40)
	m.tab = statsTabCache

	body := ansi.Strip(m.renderCacheTab(120, 40))
	assert.Contains(t, body, "First-Turn Cold Cache by Version (Claude)")
	assert.GreaterOrEqual(t, strings.Count(body, "No data"), 1)
}

func TestRenderCacheFirstTurnMetricDetailShowsPopulatedChips(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 40)
	m.tab = statsTabCache
	m.cacheLaneCursor = 4
	m.snapshot.Cache.FirstTurnByVersion = []statspkg.CacheFirstTurnVersionStat{
		{Version: "1.0.0", SessionCount: 4, ZeroReadRate: 0.75},
		{Version: "1.1.0", SessionCount: 4, ZeroReadRate: 0.25},
	}

	detail := ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "First-Turn Cold Cache by Version (Claude)")
	assert.Contains(t, detail, "worst version")
	assert.Contains(t, detail, "1.0.0")
	assert.Contains(t, detail, "versions compared")
}

func TestRenderCacheFirstTurnMetricDetailDegradesWhenEmpty(t *testing.T) {
	t.Parallel()

	m := newStatsRenderModel(120, 40)
	m.tab = statsTabCache
	m.cacheLaneCursor = 4

	detail := ansi.Strip(m.renderActiveMetricDetail(120))
	assert.Contains(t, detail, "First-Turn Cold Cache by Version (Claude)")
	assert.Contains(t, detail, "No data")
}
