package elements

import (
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (t *Theme) RenderSplitTurnGroupedChartBody(
	series []statspkg.SplitTurnSeries,
	width, height int,
	colorByKey map[string]color.Color,
	mode statspkg.StatisticMode,
	value func(statspkg.PositionTokenMetrics) float64,
	emptyState string,
) string {
	if width <= 0 {
		return ""
	}
	if len(series) == 0 {
		return emptyState
	}

	positions := collectSplitTurnPositions(series)
	lookups := make([]map[int]float64, len(series))
	for i, item := range series {
		lookups[i] = splitTurnMetricValueLookup(item.Metrics, value)
	}

	maxValue := splitTurnSeriesMaxValue(series, value)
	axisLabelWidth := TurnBarAxisLabelWidth(maxValue)
	graphWidth := max(width-axisLabelWidth-3, 1)
	bucketCount := groupedTurnMetricBucketCount(positions, lookups, graphWidth, mode)
	buckets := bucketSplitTurnMetrics(positions, lookups, bucketCount, mode)
	if len(buckets) == 0 {
		return emptyState
	}

	maxValue = groupedTurnMetricMaxValue(buckets)
	if maxValue <= 0 {
		maxValue = 1
	}
	axisLabelWidth = TurnBarAxisLabelWidth(maxValue)
	graphWidth = max(width-axisLabelWidth-3, 1)
	bucketCount = groupedTurnMetricBucketCount(positions, lookups, graphWidth, mode)
	buckets = bucketSplitTurnMetrics(positions, lookups, bucketCount, mode)
	slots := groupedTurnMetricBarSlots(buckets, graphWidth)
	if len(slots) == 0 {
		return emptyState
	}

	keys := SplitTurnSeriesKeys(series)
	cellByKey := make([]string, len(keys))
	for i, key := range keys {
		cellByKey[i] = lipgloss.NewStyle().Foreground(colorByKey[key]).Render("█")
	}
	plotHeight := max(height, 1)
	lines := make([]string, 0, plotHeight+3)
	for level := plotHeight; level >= 1; level-- {
		lines = append(lines, t.renderGroupedTurnMetricLevel(
			buckets,
			slots,
			cellByKey,
			level,
			plotHeight,
			maxValue,
			axisLabelWidth,
			graphWidth,
			width,
		))
	}
	lines = append(lines, t.RenderTurnBarAxis(axisLabelWidth, graphWidth, width))
	lines = append(lines, renderGroupedTurnXAxisRows(buckets, slots, axisLabelWidth, graphWidth)...)
	return strings.Join(lines, "\n")
}

func splitTurnSeriesMaxValue(
	series []statspkg.SplitTurnSeries,
	value func(statspkg.PositionTokenMetrics) float64,
) float64 {
	maxValue := 1.0
	for _, item := range series {
		for _, metric := range item.Metrics {
			maxValue = max(maxValue, value(metric))
		}
	}
	return maxValue
}

func groupedTurnMetricMaxValue(buckets []groupedTurnMetricBucket) float64 {
	maxValue := 0.0
	for _, bucket := range buckets {
		for _, value := range bucket.Series {
			if value.HasValue && value.Value > maxValue {
				maxValue = value.Value
			}
		}
	}
	return maxValue
}

func SplitTurnSeriesKeys(series []statspkg.SplitTurnSeries) []string {
	keys := make([]string, 0, len(series))
	for _, item := range series {
		if item.Key == "" {
			continue
		}
		keys = append(keys, item.Key)
	}
	return keys
}

func (t *Theme) renderGroupedTurnMetricLevel(
	buckets []groupedTurnMetricBucket,
	slots []groupedTurnMetricBarSlot,
	cellByKey []string,
	level, plotHeight int,
	maxValue float64,
	axisLabelWidth, graphWidth int,
	width int,
) string {
	cells := make([]string, graphWidth)
	for i := range cells {
		cells[i] = " "
	}
	for i, slot := range slots {
		if i >= len(buckets) {
			continue
		}
		for j, seriesIndex := range slot.SeriesIndexes {
			if j >= len(slot.Bars) ||
				seriesIndex >= len(buckets[i].Series) ||
				seriesIndex >= len(cellByKey) {
				continue
			}
			bucketValue := buckets[i].Series[seriesIndex]
			if !bucketValue.HasValue {
				continue
			}
			if MonotonicScaledHeight(bucketValue.Value, maxValue, max(plotHeight, 1)) < level {
				continue
			}
			fillGroupedTurnBar(cells, slot.Bars[j], cellByKey[seriesIndex])
		}
	}

	label := TurnBarLevelLabel(level, plotHeight, maxValue)
	prefix := FitToWidth(t.HistogramAxisLabel(label), axisLabelWidth) + " " + t.HistogramAxisLine("│") + " "
	return ansi.Truncate(prefix+strings.Join(cells, ""), width, "…")
}

func fillGroupedTurnBar(cells []string, bar DailyRateBarSlot, cell string) {
	if cell == "" || bar.Start < 0 || bar.End <= bar.Start || bar.Start >= len(cells) {
		return
	}
	for i := bar.Start; i < min(bar.End, len(cells)); i++ {
		cells[i] = cell
	}
}

func renderGroupedTurnXAxisRows(
	buckets []groupedTurnMetricBucket,
	slots []groupedTurnMetricBarSlot,
	axisLabelWidth, graphWidth int,
) []string {
	placements := make([]claudeTurnAxisLabelPlacement, 0, len(buckets))
	for i, bucket := range buckets {
		if i >= len(slots) {
			continue
		}
		placements = append(placements, claudeTurnAxisLabelPlacement{
			Anchor: slots[i].Anchor,
			Label:  groupedTurnBucketLabel(bucket),
		})
	}
	rows := claudeTurnAxisLabelGrid(graphWidth, placements)
	return renderClaudeTurnAxisRows(strings.Repeat(" ", axisLabelWidth+3), rows)
}

func groupedTurnBucketLabel(bucket groupedTurnMetricBucket) string {
	if bucket.StartPosition == bucket.EndPosition {
		return strconv.Itoa(bucket.StartPosition)
	}
	return strconv.Itoa(bucket.StartPosition) + "-" + strconv.Itoa(bucket.EndPosition)
}
