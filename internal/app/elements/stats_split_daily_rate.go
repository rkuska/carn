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

type groupedDailyRateBucket struct {
	Start  time.Time
	End    time.Time
	Series []DailyRateBucket
}

func (t *Theme) RenderSplitDailyRateChartBody(
	series []statspkg.SplitDailyRateSeries,
	width, height int,
	colorByKey map[string]color.Color,
) string {
	if width <= 0 {
		return ""
	}
	if len(series) == 0 {
		return NoDataLabel
	}

	maxValue := splitDailyRateSeriesMax(series)
	yFormatter := splitDailyRatePercentYLabel()
	axisLabelWidth := dailyRateAxisLabelWidth(maxValue, yFormatter)
	plotWidth := max(width-axisLabelWidth-3, 1)
	keys := make([]string, 0, len(series))
	for _, item := range series {
		keys = append(keys, item.Key)
	}
	buckets := bucketSplitDailyRates(
		series,
		groupedDailyRateBucketCount(series, plotWidth),
	)
	if len(buckets) == 0 {
		return NoDataLabel
	}

	slots := GroupedDailyRateBarSlots(buckets, plotWidth)
	plotHeight, showLabels := DailyRateChartDimensions(height)
	inactiveStyle := lipgloss.NewStyle().Foreground(t.ColorNormalDesc)
	lines := make([]string, 0, plotHeight+1)
	for level := plotHeight; level >= 1; level-- {
		lines = append(lines, t.renderGroupedDailyRateRow(
			buckets,
			slots,
			level,
			plotHeight,
			maxValue,
			plotWidth,
			axisLabelWidth,
			yFormatter,
			keys,
			colorByKey,
			inactiveStyle,
		))
	}
	if showLabels {
		lines = append(lines, strings.Repeat(" ", axisLabelWidth+3)+renderGroupedDailyRateLabels(buckets, plotWidth, slots))
	}
	return strings.Join(lines, "\n")
}

func splitDailyRateSeriesMax(series []statspkg.SplitDailyRateSeries) float64 {
	maxValue := 0.01
	for _, item := range series {
		for _, rate := range item.Rates {
			if rate.HasActivity && rate.Rate > maxValue {
				maxValue = rate.Rate
			}
		}
	}
	return maxValue
}

func bucketSplitDailyRates(
	series []statspkg.SplitDailyRateSeries,
	bucketCount int,
) []groupedDailyRateBucket {
	if len(series) == 0 || len(series[0].Rates) == 0 || bucketCount <= 0 {
		return nil
	}
	totalDays := len(series[0].Rates)
	buckets := make([]groupedDailyRateBucket, 0, bucketCount)
	for i := range bucketCount {
		start := i * totalDays / bucketCount
		end := (i + 1) * totalDays / bucketCount
		if end <= start {
			end = start + 1
		}
		group := groupedDailyRateBucket{
			Start:  series[0].Rates[start].Date,
			End:    series[0].Rates[end-1].Date,
			Series: make([]DailyRateBucket, 0, len(series)),
		}
		for _, item := range series {
			group.Series = append(group.Series, buildDailyRateBucket(item.Rates[start:end]))
		}
		buckets = append(buckets, group)
	}
	return buckets
}

func (t *Theme) renderGroupedDailyRateRow(
	buckets []groupedDailyRateBucket,
	slots []groupedDailyRateBarSlot,
	level, plotHeight int,
	maxValue float64,
	plotWidth int,
	axisLabelWidth int,
	yFormatter linechart.LabelFormatter,
	keys []string,
	colorByKey map[string]color.Color,
	inactiveStyle lipgloss.Style,
) string {
	label := dailyRateAxisLabel(level, plotHeight, maxValue, yFormatter)
	prefix := FitToWidth(t.HistogramAxisLabel(label), axisLabelWidth) +
		" " + t.HistogramAxisLine("│") + " "
	cells := BlankDailyRateCells(plotWidth)
	for i, slot := range slots {
		if len(slot.SeriesIndexes) == 0 {
			if level == 1 {
				writeDailyRateSlot(cells, DailyRateBarSlot{
					Start:  slot.Start,
					End:    slot.End,
					Anchor: slot.Anchor,
				}, inactiveStyle.Render("·"), false)
			}
			continue
		}
		for j, seriesIndex := range slot.SeriesIndexes {
			if j >= len(slot.Bars) || seriesIndex >= len(keys) || i >= len(buckets) {
				continue
			}
			seriesBucket := buckets[i].Series[seriesIndex]
			barStyle := lipgloss.NewStyle().Foreground(colorByKey[keys[seriesIndex]])
			cell, fill := renderDailyRateBucketLevel(
				seriesBucket,
				level,
				plotHeight,
				maxValue,
				barStyle,
				inactiveStyle,
			)
			writeDailyRateSlot(cells, slot.Bars[j], cell, fill)
		}
	}
	return prefix + strings.Join(cells, "")
}

func renderGroupedDailyRateLabels(
	buckets []groupedDailyRateBucket,
	plotWidth int,
	slots []groupedDailyRateBarSlot,
) string {
	labels := make([]DailyRateBucket, 0, len(buckets))
	for _, bucket := range buckets {
		labels = append(labels, DailyRateBucket{Start: bucket.Start, End: bucket.End})
	}
	return RenderDailyRateLabelLine(labels, plotWidth, groupedDailyRateLabelSlots(slots))
}

func groupedDailyRateLabelSlots(slots []groupedDailyRateBarSlot) []DailyRateBarSlot {
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
