package stats

import (
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestStatsRenderGroupedCacheShowsProviderTitlesAndVersionLegend(t *testing.T) {
	t.Parallel()

	const (
		versionOne = "1.0.0"
		versionTwo = "2.0.0"
	)

	now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"stats-1",
				"alpha",
				testStatsSessionMeta("stats-1", "alpha", now, func(meta *conv.SessionMeta) {
					meta.Provider = conv.ProviderClaude
					meta.Version = versionOne
					meta.TotalUsage = conv.TokenUsage{
						InputTokens:              100,
						CacheCreationInputTokens: 100,
						CacheReadInputTokens:     800,
					}
				}),
				testStatsSessionMeta("stats-2", "alpha", now.Add(-time.Hour), func(meta *conv.SessionMeta) {
					meta.Provider = conv.ProviderClaude
					meta.Version = versionTwo
					meta.TotalUsage = conv.TokenUsage{
						InputTokens:              250,
						CacheCreationInputTokens: 50,
						CacheReadInputTokens:     200,
					}
				}),
			),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.tab = statsTabCache
	m.splitBy = statspkg.SplitDimensionVersion
	m = m.applyFilterChange()

	body := ansi.Strip(m.renderCacheTab(120, 32))

	assert.Contains(t, body, "Daily Cache Read Rate (by Version)")
	assert.Contains(t, body, "Main vs Subagent (by Version)")
	assert.Contains(t, body, "Cache Write by Duration (by Version)")
	assert.Contains(t, body, "Cache Read by Duration (by Version)")
	assert.Contains(t, body, "1.0.0 hit")
	assert.Contains(t, body, "80.0%")
	assert.Contains(t, body, "2.0.0 hit")
	assert.Contains(t, body, "40.0%")
	assert.Contains(t, body, versionOne)
	assert.Contains(t, body, versionTwo)
}

func TestStatsRenderGroupedCacheShowsProviderHitRateChipsInProviderSplit(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	m := newStatsModel(
		[]conv.Conversation{
			testStatsConversationWithProviderAndSessions(
				conv.ProviderClaude,
				"claude-1",
				"alpha",
				testStatsSessionMeta("claude-1", "alpha", now, func(meta *conv.SessionMeta) {
					meta.Provider = conv.ProviderClaude
					meta.TotalUsage = conv.TokenUsage{
						InputTokens:              10,
						CacheCreationInputTokens: 10,
						CacheReadInputTokens:     80,
					}
				}),
			),
			testStatsConversationWithProviderAndSessions(
				conv.ProviderCodex,
				"codex-1",
				"beta",
				testStatsSessionMeta("codex-1", "beta", now.Add(-time.Hour), func(meta *conv.SessionMeta) {
					meta.Provider = conv.ProviderCodex
					meta.TotalUsage = conv.TokenUsage{
						InputTokens:              20,
						CacheCreationInputTokens: 20,
						CacheReadInputTokens:     60,
					}
				}),
			),
		},
		&fakeBrowserStore{},
		120,
		32,
		newBrowserFilterState(),
	)
	m.tab = statsTabCache
	m.splitBy = statspkg.SplitDimensionProvider
	m = m.applyFilterChange()

	body := ansi.Strip(m.renderCacheTab(120, 32))

	assert.Contains(t, body, "Daily Cache Read Rate (by Provider)")
	assert.Contains(t, body, "overall hit")
	assert.Contains(t, body, "overall write")
	assert.Contains(t, body, "Claude hit")
	assert.Contains(t, body, "80.0%")
	assert.Contains(t, body, "Claude write")
	assert.Contains(t, body, "10.0%")
	assert.Contains(t, body, "Codex hit")
	assert.Contains(t, body, "60.0%")
	assert.Contains(t, body, "Codex write")
	assert.Contains(t, body, "20.0%")
}
