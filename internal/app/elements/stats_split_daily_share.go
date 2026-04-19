package elements

import (
	"fmt"
	"image/color"
	"slices"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	statspkg "github.com/rkuska/carn/internal/stats"
)

type splitDailyShareBucket struct {
	Start       time.Time
	End         time.Time
	Prompt      int
	Total       int
	HasActivity bool
	Splits      []statspkg.SplitValue
}

func (t *Theme) RenderSplitDailyShareChartBody(
	shares []statspkg.SplitDailyShare,
	width, height int,
	colorByKey map[string]color.Color,
) string {
	if width <= 0 {
		return ""
	}
	if len(shares) == 0 {
		return NoDataLabel
	}

	maxValue := 0.01
	plotWidth := max(width-3, 1)
	buckets := bucketSplitDailyShares(shares, splitDailyShareBucketCount(shares, plotWidth))
	if len(buckets) == 0 {
		return NoDataLabel
	}
	for _, bucket := range buckets {
		if rate := splitDailyShareRate(bucket); rate > maxValue {
			maxValue = rate
		}
	}
	axisLabelWidth := max(
		lipgloss.Width(splitDailyShareAxisLabel(maxValue)),
		lipgloss.Width(splitDailyShareAxisLabel(maxValue/2)),
		lipgloss.Width(splitDailyShareAxisLabel(0)),
	)
	plotWidth = max(width-axisLabelWidth-3, 1)
	buckets = bucketSplitDailyShares(shares, splitDailyShareBucketCount(shares, plotWidth))
	maxValue = 0.01
	for _, bucket := range buckets {
		if rate := splitDailyShareRate(bucket); rate > maxValue {
			maxValue = rate
		}
	}

	slots := splitDailyShareGroupSlots(buckets, plotWidth)
	plotHeight, showLabels := DailyRateChartDimensions(height)
	lines := make([]string, 0, plotHeight+1)
	for level := plotHeight; level >= 1; level-- {
		lines = append(lines, t.renderSplitDailyShareRow(
			buckets,
			slots,
			plotWidth,
			level,
			plotHeight,
			maxValue,
			axisLabelWidth,
			colorByKey,
		))
	}
	if showLabels {
		lines = append(
			lines,
			strings.Repeat(" ", axisLabelWidth+3)+
				renderSplitDailyShareLabels(buckets, plotWidth, slots),
		)
	}
	return strings.Join(lines, "\n")
}

func splitDailyShareAxisLabel(value float64) string {
	return fmt.Sprintf("%.0f%%", value*100)
}

func bucketSplitDailyShares(
	shares []statspkg.SplitDailyShare,
	columnCount int,
) []splitDailyShareBucket {
	if len(shares) == 0 || columnCount <= 0 {
		return nil
	}
	bucketCount := min(len(shares), columnCount)
	buckets := make([]splitDailyShareBucket, 0, bucketCount)
	for i := range bucketCount {
		start := i * len(shares) / bucketCount
		end := (i + 1) * len(shares) / bucketCount
		if end <= start {
			end = start + 1
		}
		buckets = append(buckets, buildSplitDailyShareBucket(shares[start:end]))
	}
	return buckets
}

func splitDailyShareBucketCount(
	shares []statspkg.SplitDailyShare,
	plotWidth int,
) int {
	if len(shares) == 0 || plotWidth <= 0 {
		return 0
	}

	return groupedVerticalBarBucketCount(len(shares), plotWidth, func(bucketCount int) []int {
		return splitDailyShareActiveCounts(bucketSplitDailyShares(shares, bucketCount))
	})
}

func buildSplitDailyShareBucket(chunk []statspkg.SplitDailyShare) splitDailyShareBucket {
	bucket := splitDailyShareBucket{
		Start: chunk[0].Date,
		End:   chunk[len(chunk)-1].Date,
	}
	splitTotals := make(map[string]int)
	for _, day := range chunk {
		if !day.HasActivity {
			continue
		}
		bucket.HasActivity = true
		bucket.Prompt += day.Prompt
		bucket.Total += day.Total
		for _, split := range day.Splits {
			splitTotals[split.Key] += split.Value
		}
	}
	bucket.Splits = sortSplitDailyShareSplits(splitTotals)
	return bucket
}

func sortSplitDailyShareSplits(values map[string]int) []statspkg.SplitValue {
	items := make([]statspkg.SplitValue, 0, len(values))
	for key, value := range values {
		if value <= 0 {
			continue
		}
		items = append(items, statspkg.SplitValue{Key: key, Value: value})
	}
	slices.SortFunc(items, func(left, right statspkg.SplitValue) int {
		switch {
		case left.Key < right.Key:
			return -1
		case left.Key > right.Key:
			return 1
		default:
			return right.Value - left.Value
		}
	})
	return items
}

func splitDailyShareRate(bucket splitDailyShareBucket) float64 {
	if bucket.Prompt <= 0 {
		return 0
	}
	return float64(bucket.Total) / float64(bucket.Prompt)
}

func (t *Theme) renderSplitDailyShareRow(
	buckets []splitDailyShareBucket,
	slots []verticalBarGroupSlot,
	plotWidth int,
	level, plotHeight int,
	maxValue float64,
	axisLabelWidth int,
	colorByKey map[string]color.Color,
) string {
	label := ""
	switch level {
	case plotHeight:
		label = splitDailyShareAxisLabel(maxValue)
	case max((plotHeight+1)/2, 1):
		label = splitDailyShareAxisLabel(maxValue / 2)
	case 1:
		label = splitDailyShareAxisLabel(0)
	}
	prefix := FitToWidth(t.HistogramAxisLabel(label), axisLabelWidth) +
		" " + t.HistogramAxisLine("│") + " "
	cells := BlankDailyRateCells(plotWidth)
	for i, bucket := range buckets {
		t.writeSplitDailyShareSlot(
			cells,
			dailyRateRenderSlot(slots[i], bucket.HasActivity),
			bucket,
			level,
			plotHeight,
			maxValue,
			colorByKey,
		)
	}
	return ansi.Truncate(prefix+strings.Join(cells, ""), axisLabelWidth+3+len(cells), "…")
}

func (t *Theme) writeSplitDailyShareSlot(
	cells []string,
	slot DailyRateBarSlot,
	bucket splitDailyShareBucket,
	level, plotHeight int,
	maxValue float64,
	colorByKey map[string]color.Color,
) {
	if !bucket.HasActivity {
		if level == 1 && slot.Anchor < len(cells) {
			cells[slot.Anchor] = lipgloss.NewStyle().Foreground(t.ColorNormalDesc).Render("·")
		}
		return
	}
	rate := splitDailyShareRate(bucket)
	totalHeight := ScaledFloatWidth(rate, maxValue, plotHeight)
	if totalHeight < level {
		return
	}
	values := make([]float64, 0, len(bucket.Splits))
	for _, split := range bucket.Splits {
		values = append(values, float64(split.Value))
	}
	segmentHeights := ResolveFloatSegmentHeights(totalHeight, values)
	remaining := level
	for i, split := range bucket.Splits {
		segmentHeight := segmentHeights[i]
		if remaining <= segmentHeight {
			fillSplitDailySlot(cells, slot, colorByKey[split.Key])
			return
		}
		remaining -= segmentHeight
	}
}

func fillSplitDailySlot(cells []string, slot DailyRateBarSlot, fillColor color.Color) {
	for i := slot.Start; i < min(slot.End, len(cells)); i++ {
		cells[i] = lipgloss.NewStyle().Foreground(fillColor).Render("█")
	}
}

func renderSplitDailyShareLabels(
	buckets []splitDailyShareBucket,
	plotWidth int,
	slots []verticalBarGroupSlot,
) string {
	if len(buckets) == 0 {
		return ""
	}
	dayBuckets := make([]DailyRateBucket, 0, len(buckets))
	for _, bucket := range buckets {
		dayBuckets = append(dayBuckets, DailyRateBucket{Start: bucket.Start, End: bucket.End})
	}
	return RenderDailyRateLabelLine(dayBuckets, plotWidth, dailyRateLabelSlots(slots))
}

func splitDailyShareActiveCounts(buckets []splitDailyShareBucket) []int {
	counts := make([]int, 0, len(buckets))
	for _, bucket := range buckets {
		if bucket.HasActivity {
			counts = append(counts, 1)
			continue
		}
		counts = append(counts, 0)
	}
	return counts
}

func splitDailyShareGroupSlots(
	buckets []splitDailyShareBucket,
	plotWidth int,
) []verticalBarGroupSlot {
	return resolveVerticalBarGroupSlots(splitDailyShareActiveCounts(buckets), plotWidth)
}
