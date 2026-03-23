package app

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func TestRenderHorizontalBarsUsesProportionalWidthsAndTruncatesLabels(t *testing.T) {
	t.Parallel()

	got := renderHorizontalBars("Top", []barItem{
		{Label: "very-long-label-that-should-truncate", Value: 10},
		{Label: "short", Value: 5},
	}, 48, colorChartBar)

	stripped := ansi.Strip(got)
	lines := strings.Split(stripped, "\n")
	require.GreaterOrEqual(t, len(lines), 3)
	assert.Contains(t, stripped, "very-long-label")
	assert.Contains(t, stripped, "10")
	assert.Contains(t, stripped, "5")
	assert.Greater(t, strings.Count(lines[len(lines)-2], "█"), strings.Count(lines[len(lines)-1], "█"))
}

func TestRenderHorizontalBarsHandlesSingleAndEmptyInputs(t *testing.T) {
	t.Parallel()

	single := ansi.Strip(renderHorizontalBars("One", []barItem{{Label: "alpha", Value: 7}}, 36, colorChartBar))
	assert.Contains(t, single, "alpha")
	assert.Contains(t, single, "7")

	empty := ansi.Strip(renderHorizontalBars("None", nil, 36, colorChartBar))
	assert.Contains(t, empty, "None")
	assert.Contains(t, empty, "No data")
}

func TestRenderToolRateChartShowsDecimalPercentages(t *testing.T) {
	t.Parallel()

	got := ansi.Strip(renderToolRateChart("Tool Error Rate", []statspkg.ToolRateStat{
		{Name: "Bash", Rate: 12.5, Count: 67},
		{Name: "Read", Rate: 2.4, Count: 6},
	}, 48, colorChartError, true))

	assert.Contains(t, got, "12.5% (67)")
	assert.Contains(t, got, "2.4% (6)")
}

func TestRenderToolRateChartCanOmitAbsoluteCounts(t *testing.T) {
	t.Parallel()

	got := ansi.Strip(renderToolRateChart("Rejected Suggestions", []statspkg.ToolRateStat{
		{Name: "Bash", Rate: 30, Count: 3},
	}, 40, colorPrimary, false))

	assert.Contains(t, got, "30.0%")
	assert.NotContains(t, got, "(3)")
}

func TestRenderVerticalHistogramKeepsWidthsAndLabelsAligned(t *testing.T) {
	t.Parallel()

	got := renderVerticalHistogram("Durations", []histBucket{
		{Label: "A", Count: 1},
		{Label: "B", Count: 3},
		{Label: "C", Count: 0},
	}, 30, 6)

	for line := range strings.SplitSeq(ansi.Strip(got), "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), 30)
	}
	assert.Contains(t, ansi.Strip(got), "A")
	assert.Contains(t, ansi.Strip(got), "B")
	assert.Contains(t, ansi.Strip(got), "C")
	assert.Contains(t, ansi.Strip(got), "█")
}

func TestRenderVerticalHistogramShowsValueLabelsAboveBars(t *testing.T) {
	t.Parallel()

	got := ansi.Strip(renderVerticalHistogram("Durations", []histBucket{
		{Label: "A", Count: 8},
		{Label: "B", Count: 5},
		{Label: "C", Count: 2},
	}, 34, 6))

	assert.Contains(t, got, "5")
	assert.Contains(t, got, "2")
}

func TestRenderActivityHeatmapUsesGridSizingAndIntensityLevels(t *testing.T) {
	t.Parallel()

	var cells [7][24]int
	cells[1][1] = 1
	cells[2][2] = 2
	cells[3][3] = 4
	cells[4][4] = 8

	got := renderActivityHeatmap("Heatmap", cells, 56)
	stripped := ansi.Strip(got)

	assert.Contains(t, stripped, "Heatmap")
	assert.Contains(t, stripped, "Mon")
	assert.Contains(t, stripped, "Sun")
	assert.Contains(t, stripped, "···")
	assert.Contains(t, stripped, "01")
	assert.Contains(t, stripped, "04")
	assert.Contains(t, stripped, "░")
	assert.Contains(t, stripped, "▒")
	assert.Contains(t, stripped, "▓")
	assert.Contains(t, stripped, "█")
}

func TestHeatmapDisplayRowsCompressEmptyHourRanges(t *testing.T) {
	t.Parallel()

	var cells [7][24]int
	cells[0][8] = 1
	cells[4][9] = 1
	cells[2][16] = 1

	assert.Equal(t, []int{-1, 8, 9, -1, 16, -1}, heatmapDisplayRows(cells))
}

func TestRenderActivityHeatmapOmitsFullyEmptyHours(t *testing.T) {
	t.Parallel()

	var cells [7][24]int
	cells[0][8] = 1
	cells[2][16] = 2

	got := ansi.Strip(renderActivityHeatmap("Heatmap", cells, 56))

	assert.Contains(t, got, "08")
	assert.Contains(t, got, "16")
	assert.Contains(t, got, "···")
	assert.NotContains(t, got, "07")
	assert.NotContains(t, got, "17")
}

func TestRenderSideBySideSplitsAtNormalWidthAndStacksWhenNarrow(t *testing.T) {
	t.Parallel()

	sideBySide := ansi.Strip(renderSideBySide("left", "right", 80))
	assert.Contains(t, sideBySide, "│")
	assert.NotContains(t, sideBySide, "\n\nleft")

	stacked := ansi.Strip(renderSideBySide("left", "right", 40))
	assert.NotContains(t, stacked, "│")
	assert.Contains(t, stacked, "left\n\nright")
}

func TestRenderRankedTableCentersTitleWhenContentIsCapped(t *testing.T) {
	t.Parallel()

	got := ansi.Strip(renderRankedTable("Most Token-Heavy Sessions", []tableRow{
		{Columns: []string{"Project", "Slug", "Date", "Msgs", "Duration", "Tokens"}},
		{Columns: []string{"alpha", "session-a", "2026-03-22", "12", "15m", "1200"}},
	}, 120))

	lines := strings.Split(got, "\n")
	require.NotEmpty(t, lines)
	assert.Equal(t, 120, lipgloss.Width(lines[0]))
	assert.Contains(t, lines[0], "Most Token-Heavy Sessions")
}

func TestRenderRankedTableKeepsCenteredRowsAlignedByColumnStart(t *testing.T) {
	t.Parallel()

	got := ansi.Strip(renderRankedTable("Most Token-Heavy Sessions", []tableRow{
		{Columns: []string{"Project", "Slug", "Date", "Msgs", "Duration", "Tokens"}},
		{Columns: []string{"claude-search", "019ce1da-4e2", "2026-03-12", "225", "3h 31m", "166.2M"}},
		{Columns: []string{"claude-search", "019d15fd-172", "2026-03-22", "156", "1h 44m", "99.6M"}},
	}, 120))

	lines := strings.Split(got, "\n")
	require.GreaterOrEqual(t, len(lines), 4)

	firstStart := strings.Index(lines[2], "claude-search")
	secondStart := strings.Index(lines[3], "claude-search")
	require.NotEqual(t, -1, firstStart)
	require.NotEqual(t, -1, secondStart)
	assert.Equal(t, firstStart, secondStart)
}

func TestRenderVerticalHistogramAddsYAxisWithoutUnitCaption(t *testing.T) {
	t.Parallel()

	got := ansi.Strip(renderVerticalHistogram("Durations", []histBucket{
		{Label: "A", Count: 1},
		{Label: "B", Count: 3},
		{Label: "C", Count: 0},
	}, 34, 6))

	assert.Contains(t, got, "│")
	assert.Contains(t, got, "0")
	assert.Contains(t, got, "3")
	assert.NotContains(t, got, "y:")
}

func TestHistogramValueLabelPlacementUsesAboveBarByDefault(t *testing.T) {
	t.Parallel()

	level, inside := histogramValueLabelPlacement(3, 8)

	assert.Equal(t, 4, level)
	assert.False(t, inside)
}

func TestHistogramValueLabelPlacementUsesInsideBarWhenBarReachesTop(t *testing.T) {
	t.Parallel()

	level, inside := histogramValueLabelPlacement(8, 8)

	assert.Equal(t, 8, level)
	assert.True(t, inside)
}

func TestHistogramValueLabelPlacementUsesBaselineForZeroBars(t *testing.T) {
	t.Parallel()

	level, inside := histogramValueLabelPlacement(0, 8)

	assert.Equal(t, 1, level)
	assert.False(t, inside)
}

func TestActivityChartRangeExpandsSingleDaySeries(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)

	start, end := activityChartRange([]statspkg.DailyCount{
		{Date: day, Count: 2},
	})

	assert.Equal(t, day, start)
	assert.Equal(t, day.Add(24*time.Hour), end)
}

func TestClaudeTurnChartPointsUseActualTurnPositions(t *testing.T) {
	t.Parallel()

	points := claudeTurnChartPoints([]statspkg.PositionTokenMetrics{
		{Position: 1, AverageInputTokens: 100},
		{Position: 67, AverageInputTokens: 200},
		{Position: 413, AverageInputTokens: 300},
	}, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AverageInputTokens
	})

	require.Len(t, points, 3)
	assert.Equal(t, 1.0, points[0].X)
	assert.Equal(t, 67.0, points[1].X)
	assert.Equal(t, 413.0, points[2].X)
}

func TestClaudeTurnChartRangeLeavesTrailingTurnAfterLastPosition(t *testing.T) {
	t.Parallel()

	minX, maxX := claudeTurnChartRange([]statspkg.PositionTokenMetrics{
		{Position: 1},
		{Position: 67},
		{Position: 413},
	})

	assert.Equal(t, 1.0, minX)
	assert.Equal(t, 414.0, maxX)
}

func TestClaudeTurnChartRangeExpandsSinglePoint(t *testing.T) {
	t.Parallel()

	minX, maxX := claudeTurnChartRange([]statspkg.PositionTokenMetrics{
		{Position: 7},
	})

	assert.Equal(t, 7.0, minX)
	assert.Equal(t, 8.0, maxX)
}

func TestRenderClaudeTurnChartRespectsActualTurnGapSpacing(t *testing.T) {
	t.Parallel()

	compact := ansi.Strip(renderClaudeTurnChart(
		"Context Growth",
		[]statspkg.PositionTokenMetrics{
			{Position: 1, AverageInputTokens: 10},
			{Position: 2, AverageInputTokens: 20},
			{Position: 10, AverageInputTokens: 30},
		},
		40,
		8,
		colorChartToken,
		func(metric statspkg.PositionTokenMetrics) float64 {
			return metric.AverageInputTokens
		},
	))
	wideGap := ansi.Strip(renderClaudeTurnChart(
		"Context Growth",
		[]statspkg.PositionTokenMetrics{
			{Position: 1, AverageInputTokens: 10},
			{Position: 6, AverageInputTokens: 20},
			{Position: 10, AverageInputTokens: 30},
		},
		40,
		8,
		colorChartToken,
		func(metric statspkg.PositionTokenMetrics) float64 {
			return metric.AverageInputTokens
		},
	))

	assert.NotEqual(t, compact, wideGap)
}

func TestClaudeTurnAxisStepTargetsReadableTickDensity(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, claudeTurnAxisStep(0, 6))
	assert.Equal(t, 1, claudeTurnAxisStep(5, 6))
	assert.Equal(t, 11, claudeTurnAxisStep(56, 6))
}

func TestRenderClaudeTurnChartLeavesRightPaddingForFinalXAxisLabel(t *testing.T) {
	t.Parallel()

	metrics := make([]statspkg.PositionTokenMetrics, 0, 30)
	for i := 1; i <= 30; i++ {
		metrics = append(metrics, statspkg.PositionTokenMetrics{
			Position:           i,
			AverageInputTokens: float64(i) * 10000,
		})
	}

	rendered := ansi.Strip(renderClaudeTurnChart(
		"Context Growth",
		metrics,
		38,
		8,
		colorChartToken,
		func(metric statspkg.PositionTokenMetrics) float64 {
			return metric.AverageInputTokens
		},
	))
	lines := strings.Split(rendered, "\n")
	require.NotEmpty(t, lines)

	lastLine := lines[len(lines)-1]
	assert.Equal(t, 38, lipgloss.Width(lastLine))

	lastRune, _ := utf8.DecodeLastRuneInString(lastLine)
	assert.Equal(t, ' ', lastRune)
}
