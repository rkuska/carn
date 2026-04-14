package app

import (
	"math"
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

func TestRenderToolRateChartShowsLessThanPointOneForTinyNonZeroRate(t *testing.T) {
	t.Parallel()

	got := ansi.Strip(renderToolRateChart("Rejected Suggestions", []statspkg.ToolRateStat{
		{Name: "Bash", Rate: 0.04, Count: 1},
	}, 40, colorPrimary, false))

	assert.Contains(t, got, "<0.1%")
	assert.NotContains(t, got, "0.0%")
}

func TestToolRateChipValueShowsLessThanPointOneForTinyNonZeroRate(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "<0.1%", toolRateChipValue(0.04))
	assert.Equal(t, "0.0%", toolRateChipValue(0))
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

func TestResolveHistogramLayoutFillsAvailableWidthExactly(t *testing.T) {
	t.Parallel()

	layout := resolveHistogramLayout(17, 5)

	total := 0
	for _, width := range layout.bucketWidths {
		total += width
	}
	total += layout.gapWidth * (len(layout.bucketWidths) - 1)

	assert.Equal(t, 17, total)
	assert.Equal(t, 17, layout.graphWidth)
	assert.LessOrEqual(t, math.Abs(float64(layout.bucketWidths[0]-layout.bucketWidths[len(layout.bucketWidths)-1])), 1.0)
}

func TestResolveHistogramLayoutDropsGapsBeforeLeavingSlack(t *testing.T) {
	t.Parallel()

	layout := resolveHistogramLayout(6, 5)

	total := 0
	for _, width := range layout.bucketWidths {
		total += width
	}
	total += layout.gapWidth * (len(layout.bucketWidths) - 1)

	assert.Equal(t, 0, layout.gapWidth)
	assert.Equal(t, 6, total)
	assert.Equal(t, 6, layout.graphWidth)
}

func TestResolveStackedBarWidthsFillsAvailableWidthExactly(t *testing.T) {
	t.Parallel()

	widths := resolveStackedBarWidths(17, []int{7, 5, 3, 2})

	total := 0
	for _, width := range widths {
		total += width
	}

	assert.Equal(t, 17, total)
	assert.Len(t, widths, 4)
}

func TestResolveFloatSegmentHeightsFillsRequestedHeightExactly(t *testing.T) {
	t.Parallel()

	heights := resolveFloatSegmentHeights(9, []float64{7, 5, 3, 2})

	total := 0
	for _, height := range heights {
		total += height
	}

	assert.Equal(t, 9, total)
	assert.Len(t, heights, 4)
}

func TestRenderActivityHeatmapUsesGridSizingAndIntensityLevels(t *testing.T) {
	t.Parallel()

	var cells [7][24]int
	cells[0][0] = 1
	cells[1][4] = 2
	cells[2][8] = 4
	cells[3][12] = 8

	got := renderActivityHeatmap("Heatmap", cells, 56)
	stripped := ansi.Strip(got)

	assert.Contains(t, stripped, "Heatmap")
	assert.Contains(t, stripped, "Mon")
	assert.Contains(t, stripped, "Sun")
	assert.Contains(t, stripped, "00-03")
	assert.Contains(t, stripped, "12-15")
	assert.Contains(t, stripped, "20-23")
	assert.Contains(t, stripped, "░")
	assert.Contains(t, stripped, "▒")
	assert.Contains(t, stripped, "▓")
	assert.Contains(t, stripped, "█")
	assert.NotContains(t, stripped, "\n01 ")
	assert.NotContains(t, stripped, "\n13 ")
}

func TestHeatmapIntervalCellsAggregateHourlyCounts(t *testing.T) {
	t.Parallel()

	var cells [7][24]int
	cells[0][0] = 1
	cells[0][1] = 2
	cells[0][4] = 3
	cells[3][23] = 5

	intervals := heatmapIntervalCells(cells)

	assert.Equal(t, 3, intervals[0][0])
	assert.Equal(t, 3, intervals[0][1])
	assert.Equal(t, 5, intervals[3][5])
}

func TestRenderActivityHeatmapUsesCurrentPalette(t *testing.T) {
	initPaletteForTest(true)
	t.Cleanup(func() {
		initPaletteForTest(true)
	})

	var cells [7][24]int
	cells[0][12] = 8

	got := renderActivityHeatmapBody(cells, 56)
	expected := lipgloss.NewStyle().
		Foreground(colorHeatmap4).
		Render(strings.Repeat("█", heatmapCellWidth(56)))

	assert.Contains(t, got, expected)
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

func TestStatsColumnWidthsUseWeightsAndStackWhenNeeded(t *testing.T) {
	t.Parallel()

	leftWidth, rightWidth, stacked := statsColumnWidths(120, 8, 4, 30)
	assert.Equal(t, 78, leftWidth)
	assert.Equal(t, 39, rightWidth)
	assert.False(t, stacked)

	leftWidth, rightWidth, stacked = statsColumnWidths(60, 8, 4, 30)
	assert.Zero(t, leftWidth)
	assert.Zero(t, rightWidth)
	assert.True(t, stacked)
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
		{Position: 1, AveragePromptTokens: 100},
		{Position: 67, AveragePromptTokens: 200},
		{Position: 413, AveragePromptTokens: 300},
	}, func(metric statspkg.PositionTokenMetrics) float64 {
		return metric.AveragePromptTokens
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

func TestTurnBarColumnsLeaveGapsWhenWidthAllows(t *testing.T) {
	t.Parallel()

	columns := turnBarColumns(
		[]statspkg.PositionTokenMetrics{
			{Position: 1, AveragePromptTokens: 10},
			{Position: 2, AveragePromptTokens: 20},
			{Position: 10, AveragePromptTokens: 30},
		},
		8,
		30,
		30,
		func(metric statspkg.PositionTokenMetrics) float64 {
			return metric.AveragePromptTokens
		},
	)

	require.Len(t, columns, 3)
	assert.Less(t, columns[0].End, columns[1].Start)
	assert.Less(t, columns[1].End, columns[2].Start)
	assert.Equal(t, columns[0].End-columns[0].Start, columns[1].End-columns[1].Start)
	assert.Equal(t, columns[1].End-columns[1].Start, columns[2].End-columns[2].Start)
}

func TestLayoutStackedTurnColumnsUsesUniformWidthsWhenSpaceAllows(t *testing.T) {
	t.Parallel()

	columns := layoutStackedTurnColumns([]stackedTurnBarColumn{
		{turnBarColumn: turnBarColumn{Position: 1, Height: 3}},
		{turnBarColumn: turnBarColumn{Position: 2, Height: 4}},
		{turnBarColumn: turnBarColumn{Position: 3, Height: 5}},
		{turnBarColumn: turnBarColumn{Position: 4, Height: 6}},
	}, 20)

	require.Len(t, columns, 4)
	width := columns[0].End - columns[0].Start
	for _, column := range columns[1:] {
		assert.Equal(t, width, column.End-column.Start)
	}
}

func TestRenderClaudeTurnChartShowsSampledTurnNumbersOnXAxis(t *testing.T) {
	t.Parallel()

	rendered := ansi.Strip(renderClaudeTurnChart(
		"Prompt Growth",
		[]statspkg.PositionTokenMetrics{
			{Position: 1, AveragePromptTokens: 10},
			{Position: 67, AveragePromptTokens: 20},
			{Position: 413, AveragePromptTokens: 30},
		},
		50,
		8,
		colorChartToken,
		func(metric statspkg.PositionTokenMetrics) float64 {
			return metric.AveragePromptTokens
		},
	))

	assert.Contains(t, rendered, "67")
	assert.Contains(t, rendered, "413")
}

func TestRenderClaudeTurnChartUsesMultipleXAxisRowsWhenNeeded(t *testing.T) {
	t.Parallel()

	rendered := ansi.Strip(renderClaudeTurnChart(
		"Prompt Growth",
		[]statspkg.PositionTokenMetrics{
			{Position: 11, AveragePromptTokens: 10},
			{Position: 22, AveragePromptTokens: 20},
			{Position: 33, AveragePromptTokens: 30},
			{Position: 44, AveragePromptTokens: 40},
			{Position: 55, AveragePromptTokens: 50},
			{Position: 66, AveragePromptTokens: 60},
			{Position: 77, AveragePromptTokens: 70},
		},
		24,
		8,
		colorChartToken,
		func(metric statspkg.PositionTokenMetrics) float64 {
			return metric.AveragePromptTokens
		},
	))

	assert.Contains(t, rendered, "11")
	assert.Contains(t, rendered, "22")
	assert.Contains(t, rendered, "33")
	assert.Contains(t, rendered, "44")
	assert.Contains(t, rendered, "55")
	assert.Contains(t, rendered, "66")
	assert.Contains(t, rendered, "77")
}

func TestRenderClaudeTurnChartLeavesRightPaddingForFinalXAxisLabel(t *testing.T) {
	t.Parallel()

	metrics := make([]statspkg.PositionTokenMetrics, 0, 30)
	for i := 1; i <= 30; i++ {
		metrics = append(metrics, statspkg.PositionTokenMetrics{
			Position:            i,
			AveragePromptTokens: float64(i) * 10000,
		})
	}

	rendered := ansi.Strip(renderClaudeTurnChart(
		"Prompt Growth",
		metrics,
		38,
		8,
		colorChartToken,
		func(metric statspkg.PositionTokenMetrics) float64 {
			return metric.AveragePromptTokens
		},
	))
	lines := strings.Split(rendered, "\n")
	require.NotEmpty(t, lines)

	lastLine := lines[len(lines)-1]
	assert.Equal(t, 38, lipgloss.Width(lastLine))

	lastRune, _ := utf8.DecodeLastRuneInString(lastLine)
	assert.Equal(t, ' ', lastRune)
}

func TestRenderVersionedTurnChartBodyShowsLegendAndTurnLabels(t *testing.T) {
	t.Parallel()

	rendered := ansi.Strip(renderVersionedTurnChartBody(
		[]statspkg.VersionTurnSeries{
			{
				Version: "1.0.0",
				Metrics: []statspkg.PositionTokenMetrics{
					{Position: 1, AveragePromptTokens: 120},
					{Position: 2, AveragePromptTokens: 180},
					{Position: 3, AveragePromptTokens: 240},
				},
			},
			{
				Version: statspkg.UnknownVersionLabel,
				Metrics: []statspkg.PositionTokenMetrics{
					{Position: 1, AveragePromptTokens: 90},
					{Position: 2, AveragePromptTokens: 110},
					{Position: 3, AveragePromptTokens: 130},
				},
			},
		},
		80,
		8,
		versionColorMap([]string{"1.0.0", statspkg.UnknownVersionLabel}),
		func(metric statspkg.PositionTokenMetrics) float64 {
			return metric.AveragePromptTokens
		},
	))

	assert.Contains(t, rendered, "1.0.0")
	assert.Contains(t, rendered, statspkg.UnknownVersionLabel)
	assert.Contains(t, rendered, "1")
	assert.Contains(t, rendered, "2")
	assert.Contains(t, rendered, "3")
	assert.Contains(t, rendered, "█")
}

func TestRenderVerticalStackedHistogramBodyShowsSingleZeroAxisLabel(t *testing.T) {
	t.Parallel()

	rendered := ansi.Strip(renderVerticalStackedHistogramBody(
		[]stackedHistBucket{
			{
				Label: "0-20",
				Total: 6,
				Segments: []stackedHistSegment{
					{Value: 2, Color: colorChartBar},
					{Value: 4, Color: colorPrimary},
				},
			},
			{
				Label: "21-50",
				Total: 12,
				Segments: []stackedHistSegment{
					{Value: 5, Color: colorChartBar},
					{Value: 7, Color: colorPrimary},
				},
			},
		},
		32,
		4,
		statspkg.FormatNumber,
	))

	lines := strings.Split(rendered, "\n")
	zeroLines := 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "0 ") ||
			strings.HasPrefix(strings.TrimSpace(line), "0└") {
			zeroLines++
		}
	}
	assert.Equal(t, 1, zeroLines)
}

func TestRenderChartWithVersionLegendBuildsChartAtReducedWidth(t *testing.T) {
	t.Parallel()

	versionLabels := []string{"1.0.0", "2.0.0"}
	var builtWidth int

	rendered := renderChartWithVersionLegend(
		64,
		versionLabels,
		versionColorMap(versionLabels),
		24,
		func(chartWidth int) string {
			builtWidth = chartWidth
			return strings.Repeat("X", chartWidth)
		},
	)

	assert.Less(t, builtWidth, 64)
	assert.NotContains(t, ansi.Strip(rendered), "…")
}
