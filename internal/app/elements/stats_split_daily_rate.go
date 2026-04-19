package elements

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"

	statspkg "github.com/rkuska/carn/internal/stats"
)

type groupedDailyValueBucket struct {
	Start  time.Time
	End    time.Time
	Series []DailyValueBucket
}

func (t *Theme) RenderSplitDailyValueChartBody(
	series []statspkg.SplitDailyValueSeries,
	width, height int,
	colorByKey map[string]color.Color,
	yFormatter linechart.LabelFormatter,
	minValue float64,
) string {
	if width <= 0 {
		return ""
	}
	if len(series) == 0 {
		return NoDataLabel
	}

	maxValue := splitDailyValueSeriesMax(series, minValue)
	axisLabelWidth := dailyRateAxisLabelWidth(maxValue, yFormatter)
	plotWidth := max(width-axisLabelWidth-3, 1)
	keys := make([]string, 0, len(series))
	for _, item := range series {
		keys = append(keys, item.Key)
	}

	buckets := bucketSplitDailyValues(
		series,
		groupedDailyValueBucketCount(series, plotWidth),
	)
	if len(buckets) == 0 {
		return NoDataLabel
	}

	slots := GroupedDailyValueBarSlots(buckets, plotWidth)
	plotHeight, showLabels := DailyRateChartDimensions(height)
	inactiveStyle := lipgloss.NewStyle().Foreground(t.ColorNormalDesc)
	stylesByKey := make([]lipgloss.Style, len(keys))
	for i, key := range keys {
		stylesByKey[i] = lipgloss.NewStyle().Foreground(colorByKey[key])
	}
	lines := make([]string, 0, plotHeight+1)
	for level := plotHeight; level >= 1; level-- {
		lines = append(lines, t.renderGroupedDailyValueRow(
			buckets,
			slots,
			level,
			plotHeight,
			maxValue,
			plotWidth,
			axisLabelWidth,
			yFormatter,
			stylesByKey,
			inactiveStyle,
		))
	}
	if showLabels {
		lines = append(
			lines,
			strings.Repeat(" ", axisLabelWidth+3)+
				renderGroupedDailyValueLabels(buckets, plotWidth, slots),
		)
	}
	return strings.Join(lines, "\n")
}

func (t *Theme) RenderSplitDailyRateChartBody(
	series []statspkg.SplitDailyValueSeries,
	width, height int,
	colorByKey map[string]color.Color,
) string {
	return t.RenderSplitDailyValueChartBody(
		series,
		width,
		height,
		colorByKey,
		splitDailyRatePercentYLabel(),
		0.01,
	)
}

func splitDailyValueSeriesMax(
	series []statspkg.SplitDailyValueSeries,
	minValue float64,
) float64 {
	maxValue := minValue
	for _, item := range series {
		for _, value := range item.Values {
			if value.HasValue && value.Value > maxValue {
				maxValue = value.Value
			}
		}
	}
	return maxValue
}

func bucketSplitDailyValues(
	series []statspkg.SplitDailyValueSeries,
	bucketCount int,
) []groupedDailyValueBucket {
	if len(series) == 0 || len(series[0].Values) == 0 || bucketCount <= 0 {
		return nil
	}

	totalDays := len(series[0].Values)
	buckets := make([]groupedDailyValueBucket, 0, bucketCount)
	for i := range bucketCount {
		start := i * totalDays / bucketCount
		end := (i + 1) * totalDays / bucketCount
		if end <= start {
			end = start + 1
		}

		group := groupedDailyValueBucket{
			Start:  series[0].Values[start].Date,
			End:    series[0].Values[end-1].Date,
			Series: make([]DailyValueBucket, 0, len(series)),
		}
		for _, item := range series {
			group.Series = append(group.Series, buildDailyValueBucket(item.Values[start:end]))
		}
		buckets = append(buckets, group)
	}
	return buckets
}

func (t *Theme) renderGroupedDailyValueRow(
	buckets []groupedDailyValueBucket,
	slots []groupedDailyValueBarSlot,
	level, plotHeight int,
	maxValue float64,
	plotWidth int,
	axisLabelWidth int,
	yFormatter linechart.LabelFormatter,
	stylesByKey []lipgloss.Style,
	inactiveStyle lipgloss.Style,
) string {
	label := dailyRateAxisLabel(level, plotHeight, maxValue, yFormatter)
	prefix := FitToWidth(t.HistogramAxisLabel(label), axisLabelWidth) +
		" " + t.HistogramAxisLine("│") + " "
	cells := BlankDailyRateCells(plotWidth)
	for i, slot := range slots {
		if len(slot.SeriesIndexes) == 0 {
			if level == 1 {
				writeDailyRateSlot(
					cells,
					buildDailyRateBarSlot(slot.Start, slot.End),
					inactiveStyle.Render("·"),
					false,
				)
			}
			continue
		}

		for j, seriesIndex := range slot.SeriesIndexes {
			if j >= len(slot.Bars) || seriesIndex >= len(stylesByKey) || i >= len(buckets) {
				continue
			}
			seriesBucket := buckets[i].Series[seriesIndex]
			cell, fill := renderDailyRateBucketLevel(
				seriesBucket,
				level,
				plotHeight,
				maxValue,
				stylesByKey[seriesIndex],
				inactiveStyle,
			)
			writeDailyRateSlot(cells, slot.Bars[j], cell, fill)
		}
	}
	return prefix + strings.Join(cells, "")
}

func renderGroupedDailyValueLabels(
	buckets []groupedDailyValueBucket,
	plotWidth int,
	slots []groupedDailyValueBarSlot,
) string {
	labels := make([]DailyRateBucket, 0, len(buckets))
	for _, bucket := range buckets {
		labels = append(labels, DailyRateBucket{Start: bucket.Start, End: bucket.End})
	}
	return RenderDailyRateLabelLine(labels, plotWidth, groupedDailyValueLabelSlots(slots))
}

func groupedDailyValueLabelSlots(slots []groupedDailyValueBarSlot) []DailyRateBarSlot {
	labelSlots := make([]DailyRateBarSlot, 0, len(slots))
	for _, slot := range slots {
		labelSlots = append(labelSlots, DailyRateBarSlot{
			Start:  slot.Start,
			End:    slot.End,
			Anchor: slot.Anchor,
		})
	}
	return labelSlots
}

func splitDailyRatePercentYLabel() linechart.LabelFormatter {
	return func(_ int, v float64) string {
		return fmt.Sprintf("%.0f%%", v*100)
	}
}
