package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func claudeSession(version string, turns ...TurnTokens) conv.SessionTurnMetrics {
	return conv.SessionTurnMetrics{
		Provider: conv.ProviderClaude,
		Version:  version,
		Turns:    turns,
	}
}

func TestComputeCacheFirstTurnByVersionDropsSparseGroupsAndSortsByVersion(t *testing.T) {
	t.Parallel()

	series := []conv.SessionTurnMetrics{
		claudeSession("1.2.0", TurnTokens{CacheReadTokens: 0}),
		claudeSession("1.2.0", TurnTokens{CacheReadTokens: 0}),
		claudeSession("1.2.0", TurnTokens{CacheReadTokens: 500}),
		claudeSession("1.10.0", TurnTokens{CacheReadTokens: 0}),
		claudeSession("1.10.0", TurnTokens{CacheReadTokens: 100}),
		claudeSession("1.10.0", TurnTokens{CacheReadTokens: 200}),
		claudeSession("1.10.0", TurnTokens{CacheReadTokens: 300}),
		claudeSession("sparse", TurnTokens{CacheReadTokens: 0}),
	}

	got := ComputeCacheFirstTurnByVersion(series)
	require.Len(t, got, 2)
	assert.Equal(t, "1.2.0", got[0].Version)
	assert.Equal(t, 3, got[0].SessionCount)
	assert.InDelta(t, 2.0/3.0, got[0].ZeroReadRate, 0.0001)
	assert.Equal(t, 0, got[0].MedianFirstRead)

	assert.Equal(t, "1.10.0", got[1].Version)
	assert.Equal(t, 4, got[1].SessionCount)
	assert.InDelta(t, 0.25, got[1].ZeroReadRate, 0.0001)
	assert.Equal(t, 150, got[1].MedianFirstRead)
}

func TestComputeCacheFirstTurnByVersionSkipsNonClaudeAndEmptyTurnSessions(t *testing.T) {
	t.Parallel()

	series := []conv.SessionTurnMetrics{
		claudeSession("1.0.0", TurnTokens{CacheReadTokens: 0}),
		claudeSession("1.0.0", TurnTokens{CacheReadTokens: 0}),
		claudeSession("1.0.0"), // empty turns
		{Provider: conv.ProviderCodex, Version: "1.0.0", Turns: []TurnTokens{{CacheReadTokens: 0}}},
	}

	got := ComputeCacheFirstTurnByVersion(series)
	assert.Empty(t, got)
}

func TestComputeCacheFirstTurnByVersionComputesMedianForOddCount(t *testing.T) {
	t.Parallel()

	series := []conv.SessionTurnMetrics{
		claudeSession("2.0.0", TurnTokens{CacheReadTokens: 10}),
		claudeSession("2.0.0", TurnTokens{CacheReadTokens: 20}),
		claudeSession("2.0.0", TurnTokens{CacheReadTokens: 30}),
	}

	got := ComputeCacheFirstTurnByVersion(series)
	require.Len(t, got, 1)
	assert.Equal(t, 20, got[0].MedianFirstRead)
}

func TestCompareVersionLabel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		left, right string
		want        int
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2.0", "1.10.0", -1},
		{"2.0", "1.9.9", 1},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.0", "unknown", -1},
		{"alpha", "beta", -1},
	}
	for _, c := range cases {
		got := compareVersionLabel(c.left, c.right)
		switch {
		case c.want < 0:
			assert.Negative(t, got, "left=%q right=%q", c.left, c.right)
		case c.want > 0:
			assert.Positive(t, got, "left=%q right=%q", c.left, c.right)
		default:
			assert.Zero(t, got, "left=%q right=%q", c.left, c.right)
		}
	}
}
