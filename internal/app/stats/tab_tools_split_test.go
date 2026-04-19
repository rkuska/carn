package stats

import (
	"image/color"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rkuska/carn/internal/app/testutil"
	conv "github.com/rkuska/carn/internal/conversation"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestSplitToolRateRowsUseOverallRateScaleAndNumeratorSegments(t *testing.T) {
	t.Parallel()

	theme := testutil.NewTestTheme()
	rows := splitToolRateRows(
		theme,
		[]statspkg.SplitRateStat{{
			Name:  "Read",
			Count: 6,
			Total: 10,
			Rate:  60,
			Splits: []statspkg.SplitValue{
				{Key: "1.0.0", Value: 4},
				{Key: "2.0.0", Value: 2},
			},
		}},
		map[string]color.Color{
			"1.0.0": theme.ColorChartToken,
			"2.0.0": theme.ColorChartBar,
		},
		true,
	)

	require.Len(t, rows, 1)
	assert.Equal(t, 60.0, rows[0].Scale)
	require.Len(t, rows[0].Segments, 2)
	assert.Equal(t, 4, rows[0].Segments[0].Value)
	assert.Equal(t, 2, rows[0].Segments[1].Value)
}

func TestSplitToolsErrorLegendKeysUseOnlyErrorContributors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	sessions := []conv.SessionMeta{
		testStatsSessionMeta("stats-1", "alpha", now, func(meta *conv.SessionMeta) {
			meta.Version = "1.0.0"
			meta.ToolCounts = map[string]int{"Read": 10}
		}),
		testStatsSessionMeta("stats-2", "alpha", now.Add(-time.Hour), func(meta *conv.SessionMeta) {
			meta.Version = "2.0.0"
			meta.ToolCounts = map[string]int{"Read": 10}
			meta.ToolErrorCounts = map[string]int{"Read": 3}
		}),
	}

	grouped := statspkg.ComputeToolsBySplit(
		sessions,
		statspkg.TimeRange{},
		statspkg.SplitDimensionVersion,
		nil,
	)

	assert.Equal(t, []string{"2.0.0"}, presentSplitKeys(grouped.ToolErrorRates, rateStatSplits))
}
