package elements

import (
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestResolveVerticalBarGroupSlotsUsesUniformBarsAndMarkerGroups(t *testing.T) {
	t.Parallel()

	slots := resolveVerticalBarGroupSlots([]int{2, 0, 1}, 12)

	require.Len(t, slots, 3)
	require.Len(t, slots[0].Bars, 2)
	require.Len(t, slots[2].Bars, 1)
	assert.Empty(t, slots[1].Bars)
	assert.Equal(t, 1, slots[1].End-slots[1].Start)

	barWidth := slots[0].Bars[0].End - slots[0].Bars[0].Start
	assert.Equal(t, barWidth, slots[0].Bars[1].End-slots[0].Bars[1].Start)
	assert.Equal(t, barWidth, slots[2].Bars[0].End-slots[2].Bars[0].Start)
	assert.GreaterOrEqual(t, slots[1].Start-slots[0].End, 1)
	assert.GreaterOrEqual(t, slots[2].Start-slots[1].End, 1)
}

func TestGroupedVerticalBarBucketCountReducesWhenGroupsDoNotFit(t *testing.T) {
	t.Parallel()

	got := groupedVerticalBarBucketCount(5, 6, func(bucketCount int) []int {
		counts := make([]int, bucketCount)
		for i := range counts {
			counts[i] = 2
		}
		return counts
	})

	assert.Equal(t, 2, got)
}

func TestDailyRateBarSlotsKeepUniformWidthsAndFillPlot(t *testing.T) {
	t.Parallel()

	slots := DailyRateBarSlots(7, 40)

	require.Len(t, slots, 7)
	assert.Equal(t, 40, DailyRatePlotWidth(slots))
	assert.Equal(t, slots[0].End-slots[0].Start, slots[3].End-slots[3].Start)
	assert.Equal(t, slots[3].End-slots[3].Start, slots[6].End-slots[6].Start)
	assert.Greater(t, slots[1].Start-slots[0].End, 0)
}

func TestRenderSplitDailyShareChartBodyShowsBarsAndEmptyBucketMarker(t *testing.T) {
	t.Parallel()

	dayOne := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)
	theme := NewTheme(true)
	got := ansi.Strip(theme.RenderSplitDailyShareChartBody(
		[]statspkg.SplitDailyShare{
			{
				Date:        dayOne,
				Prompt:      10,
				Total:       5,
				HasActivity: true,
				Splits: []statspkg.SplitValue{
					{Key: "Claude", Value: 3},
					{Key: "Codex", Value: 2},
				},
			},
			{
				Date:        dayOne.AddDate(0, 0, 1),
				Prompt:      0,
				Total:       0,
				HasActivity: false,
			},
		},
		24,
		6,
		map[string]color.Color{
			"Claude": theme.ColorPrimary,
			"Codex":  theme.ColorChartBar,
		},
	))

	assert.Contains(t, got, "█")
	assert.Equal(t, 1, strings.Count(got, "·"))
	assert.Contains(t, got, "04/17")
	assert.Contains(t, got, "04/18")
}

func TestTurnBarColumnsMergeAdjacentPositionsWhenWidthIsTight(t *testing.T) {
	t.Parallel()

	metrics := []statspkg.PositionTokenMetrics{
		{Position: 1, AveragePromptTokens: 10},
		{Position: 2, AveragePromptTokens: 20},
		{Position: 10, AveragePromptTokens: 30},
	}

	columns := TurnBarColumns(metrics, 4, 4, 30, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AveragePromptTokens
	})

	require.Len(t, columns, 2)
	assert.Equal(t, 1, columns[0].StartPosition)
	assert.Equal(t, 1, columns[0].EndPosition)
	assert.Equal(t, 2, columns[1].StartPosition)
	assert.Equal(t, 10, columns[1].EndPosition)

	labels := ansi.Strip(strings.Join(RenderTurnBarXAxisRows(columns, 2, 4), "\n"))
	assert.Contains(t, labels, "1")
	assert.Contains(t, labels, "2-10")
}
