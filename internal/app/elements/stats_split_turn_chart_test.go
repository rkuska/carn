package elements

import (
	"image/color"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestBucketSplitTurnMetricsAveragesOnlyVisiblePositions(t *testing.T) {
	t.Parallel()

	series := []statspkg.SplitTurnSeries{
		{
			Key: "Claude",
			Metrics: []statspkg.PositionTokenMetrics{
				{Position: 1, AveragePromptTokens: 10},
				{Position: 3, AveragePromptTokens: 30},
			},
		},
	}

	value := func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AveragePromptTokens
	}
	positions, lookups := splitTurnBucketInputs(series, value)

	gotAverage := bucketSplitTurnMetrics(positions, lookups, 1, statspkg.StatisticModeAverage)
	gotMax := bucketSplitTurnMetrics(positions, lookups, 1, statspkg.StatisticModeMax)

	require.Len(t, gotAverage, 1)
	require.Len(t, gotAverage[0].Series, 1)
	assert.Equal(t, 20.0, gotAverage[0].Series[0].Value)
	assert.True(t, gotAverage[0].Series[0].HasValue)

	require.Len(t, gotMax, 1)
	require.Len(t, gotMax[0].Series, 1)
	assert.Equal(t, 30.0, gotMax[0].Series[0].Value)
	assert.True(t, gotMax[0].Series[0].HasValue)
}

func TestGroupedTurnMetricBucketCountReducesToPreserveGroupGaps(t *testing.T) {
	t.Parallel()

	series := sparseSplitTurnSeries(
		[]int{1},
		[]int{2},
		[]int{3},
		[]int{4},
	)

	positions, lookups := splitTurnBucketInputs(series, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AveragePromptTokens
	})
	assert.Equal(t, 3, groupedTurnMetricBucketCount(positions, lookups, 6, statspkg.StatisticModeAverage))
}

func splitTurnBucketInputs(
	series []statspkg.SplitTurnSeries,
	value func(statspkg.PositionTokenMetrics) float64,
) ([]int, []map[int]float64) {
	lookups := make([]map[int]float64, len(series))
	for i, item := range series {
		lookups[i] = splitTurnMetricValueLookup(item.Metrics, value)
	}
	return collectSplitTurnPositions(series), lookups
}

func TestGroupedTurnBarSlotsKeepUniformWidthsAndVisibleGroupGaps(t *testing.T) {
	t.Parallel()

	buckets := []groupedTurnMetricBucket{
		{Series: []groupedTurnMetricValue{{HasValue: true}, {HasValue: true}, {HasValue: false}}},
		{Series: []groupedTurnMetricValue{{HasValue: true}, {HasValue: false}, {HasValue: false}}},
		{Series: []groupedTurnMetricValue{{HasValue: false}, {HasValue: true}, {HasValue: true}}},
	}

	slots := groupedTurnMetricBarSlots(buckets, 24)

	require.Len(t, slots, len(buckets))
	groupGap := slots[1].Start - slots[0].End
	barWidth := slots[0].Bars[0].End - slots[0].Bars[0].Start
	barGap := slots[0].Bars[1].Start - slots[0].Bars[0].End
	assert.GreaterOrEqual(t, groupGap, 1)
	assert.Equal(t, 0, barGap)
	assert.Greater(t, groupGap, barGap)

	for i, slot := range slots {
		if i > 0 {
			assert.Equal(t, groupGap, slot.Start-slots[i-1].End)
		}
		for j, bar := range slot.Bars {
			assert.Equal(t, barWidth, bar.End-bar.Start)
			if j > 0 {
				assert.Equal(t, barGap, bar.Start-slot.Bars[j-1].End)
			}
		}
	}
}

func TestRenderSplitTurnGroupedChartBodyUsesObservedPositionsOnly(t *testing.T) {
	t.Parallel()

	theme := NewTheme(true)
	got := ansi.Strip(theme.RenderSplitTurnGroupedChartBody(
		[]statspkg.SplitTurnSeries{
			{
				Key: "Claude",
				Metrics: []statspkg.PositionTokenMetrics{
					{Position: 11, AveragePromptTokens: 10},
					{Position: 33, AveragePromptTokens: 30},
				},
			},
		},
		36,
		8,
		map[string]color.Color{"Claude": theme.ColorChartToken},
		statspkg.StatisticModeAverage,
		func(metric statspkg.PositionTokenMetrics) float64 {
			return metric.AveragePromptTokens
		},
		"No main-thread turn metrics",
	))

	assert.Contains(t, got, "11")
	assert.Contains(t, got, "33")
	assert.NotContains(t, got, "22")
}

func sparseSplitTurnSeries(positionGroups ...[]int) []statspkg.SplitTurnSeries {
	series := make([]statspkg.SplitTurnSeries, 0, len(positionGroups))
	for i, positions := range positionGroups {
		metrics := make([]statspkg.PositionTokenMetrics, 0, len(positions))
		for _, position := range positions {
			metrics = append(metrics, statspkg.PositionTokenMetrics{
				Position:            position,
				AveragePromptTokens: 10,
			})
		}
		series = append(series, statspkg.SplitTurnSeries{
			Key:     string(rune('a' + i)),
			Metrics: metrics,
		})
	}
	return series
}
