package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestNormalizeVersionLabelUsesUnknownForBlankValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, UnknownVersionLabel, NormalizeVersionLabel(""))
	assert.Equal(t, UnknownVersionLabel, NormalizeVersionLabel("   "))
	assert.Equal(t, "1.2.3", NormalizeVersionLabel("1.2.3"))
}

func TestComputeTurnTokenMetricsBySplitGroupsByVersionWhenAllowedSet(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)

	got := ComputeTurnTokenMetricsBySplit(
		[]SessionTurnMetrics{
			{
				Provider:  conv.ProviderClaude,
				Version:   "1.0.0",
				Timestamp: now.Add(-2 * time.Hour),
				Turns:     []TurnTokens{{PromptTokens: 100, TurnTokens: 150}},
			},
			{
				Provider:  conv.ProviderClaude,
				Version:   "1.0.0",
				Timestamp: now.Add(-90 * time.Minute),
				Turns:     []TurnTokens{{PromptTokens: 110, TurnTokens: 170}},
			},
			{
				Provider:  conv.ProviderClaude,
				Version:   "1.0.0",
				Timestamp: now.Add(-time.Hour),
				Turns:     []TurnTokens{{PromptTokens: 120, TurnTokens: 190}},
			},
			{
				Provider:  conv.ProviderClaude,
				Version:   "",
				Timestamp: now.Add(-2 * time.Hour),
				Turns:     []TurnTokens{{PromptTokens: 200, TurnTokens: 260}},
			},
			{
				Provider:  conv.ProviderClaude,
				Version:   "",
				Timestamp: now.Add(-90 * time.Minute),
				Turns:     []TurnTokens{{PromptTokens: 220, TurnTokens: 280}},
			},
			{
				Provider:  conv.ProviderClaude,
				Version:   "",
				Timestamp: now.Add(-time.Hour),
				Turns:     []TurnTokens{{PromptTokens: 240, TurnTokens: 300}},
			},
			{
				Provider:  conv.ProviderCodex,
				Version:   "9.9.9",
				Timestamp: now.Add(-time.Hour),
				Turns:     []TurnTokens{{PromptTokens: 999, TurnTokens: 999}},
			},
		},
		TimeRange{Start: now.Add(-24 * time.Hour), End: now},
		SplitDimensionVersion,
		map[string]bool{"1.0.0": true, UnknownVersionLabel: true},
		StatisticModeAverage,
	)

	require.Len(t, got, 2)
	assert.Equal(t, "1.0.0", got[0].Key)
	require.Len(t, got[0].Metrics, 1)
	assert.Equal(t, 3, got[0].Metrics[0].SampleCount)
	assert.InDelta(t, 110.0, got[0].Metrics[0].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 170.0, got[0].Metrics[0].AverageTurnTokens, 0.0001)

	assert.Equal(t, UnknownVersionLabel, got[1].Key)
	require.Len(t, got[1].Metrics, 1)
	assert.Equal(t, 3, got[1].Metrics[0].SampleCount)
	assert.InDelta(t, 220.0, got[1].Metrics[0].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 280.0, got[1].Metrics[0].AverageTurnTokens, 0.0001)
}

func TestComputeTurnTokenMetricsBySplitGroupsByProvider(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)

	got := ComputeTurnTokenMetricsBySplit(
		[]SessionTurnMetrics{
			{
				Provider:  conv.ProviderClaude,
				Timestamp: now.Add(-time.Hour),
				Turns:     []TurnTokens{{PromptTokens: 100, TurnTokens: 200}},
			},
			{
				Provider:  conv.ProviderCodex,
				Timestamp: now.Add(-time.Hour),
				Turns:     []TurnTokens{{PromptTokens: 50, TurnTokens: 80}},
			},
		},
		TimeRange{Start: now.Add(-24 * time.Hour), End: now},
		SplitDimensionProvider,
		nil,
		StatisticModeAverage,
	)

	require.Len(t, got, 2)
	assert.Equal(t, "Claude", got[0].Key)
	assert.Equal(t, "Codex", got[1].Key)
}

func TestComputeTurnTokenMetricsBySplitHonorsStatisticMode(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	sessions := []SessionTurnMetrics{
		{
			Provider:  conv.ProviderClaude,
			Version:   "1.0.0",
			Timestamp: now.Add(-time.Hour),
			Turns:     []TurnTokens{{PromptTokens: 100, TurnTokens: 100}},
		},
		{
			Provider:  conv.ProviderClaude,
			Version:   "1.0.0",
			Timestamp: now.Add(-2 * time.Hour),
			Turns:     []TurnTokens{{PromptTokens: 300, TurnTokens: 300}},
		},
	}
	timeRange := TimeRange{Start: now.Add(-24 * time.Hour), End: now}

	gotAvg := ComputeTurnTokenMetricsBySplit(sessions, timeRange, SplitDimensionVersion, nil, StatisticModeAverage)
	gotMax := ComputeTurnTokenMetricsBySplit(sessions, timeRange, SplitDimensionVersion, nil, StatisticModeMax)

	require.Len(t, gotAvg, 1)
	require.Len(t, gotMax, 1)
	assert.InDelta(t, 200.0, gotAvg[0].Metrics[0].AveragePromptTokens, 0.0001)
	assert.InDelta(t, 300.0, gotMax[0].Metrics[0].AveragePromptTokens, 0.0001)
}

func TestComputeTurnTokenMetricsBySplitReturnsNilForUnsupportedDimension(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)

	got := ComputeTurnTokenMetricsBySplit(
		[]SessionTurnMetrics{
			{
				Provider:  conv.ProviderClaude,
				Version:   "1.0.0",
				Timestamp: now.Add(-time.Hour),
				Turns:     []TurnTokens{{PromptTokens: 100, TurnTokens: 200}},
			},
		},
		TimeRange{Start: now.Add(-24 * time.Hour), End: now},
		SplitDimensionModel,
		nil,
		StatisticModeAverage,
	)

	assert.Nil(t, got)
}
