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

type groupedDailyShareBucket struct {
	Start       time.Time
	End         time.Time
	Prompt      int
	Total       int
	HasActivity bool
	Versions    []statspkg.VersionValue
}

func RenderGroupedDailyShareChartBody(
	shares []statspkg.GroupedDailyShare,
	width, height int,
	colorByVersion map[string]color.Color,
) string {
	if width <= 0 {
		return ""
	}
	if len(shares) == 0 {
		return NoDataLabel
	}

	maxValue := 0.01
	plotWidth := max(width-3, 1)
	buckets := bucketGroupedDailyShares(shares, plotWidth)
	if len(buckets) == 0 {
		return NoDataLabel
	}
	for _, bucket := range buckets {
		if rate := groupedDailyShareRate(bucket); rate > maxValue {
			maxValue = rate
		}
	}
	axisLabelWidth := max(
		lipgloss.Width(groupedDailyShareAxisLabel(maxValue)),
		lipgloss.Width(groupedDailyShareAxisLabel(maxValue/2)),
		lipgloss.Width(groupedDailyShareAxisLabel(0)),
	)
	plotWidth = max(width-axisLabelWidth-3, 1)
	buckets = bucketGroupedDailyShares(shares, plotWidth)
	maxValue = 0.01
	for _, bucket := range buckets {
		if rate := groupedDailyShareRate(bucket); rate > maxValue {
			maxValue = rate
		}
	}

	slots := DailyRateBarSlots(len(buckets), plotWidth)
	plotHeight, showLabels := DailyRateChartDimensions(height)
	lines := make([]string, 0, plotHeight+1)
	for level := plotHeight; level >= 1; level-- {
		lines = append(lines, renderGroupedDailyShareRow(
			buckets,
			slots,
			level,
			plotHeight,
			maxValue,
			axisLabelWidth,
			colorByVersion,
		))
	}
	if showLabels {
		lines = append(lines, strings.Repeat(" ", axisLabelWidth+3)+renderGroupedDailyShareLabels(buckets, plotWidth, slots))
	}
	return strings.Join(lines, "\n")
}

func groupedDailyShareAxisLabel(value float64) string {
	return fmt.Sprintf("%.0f%%", value*100)
}

func bucketGroupedDailyShares(
	shares []statspkg.GroupedDailyShare,
	columnCount int,
) []groupedDailyShareBucket {
	if len(shares) == 0 || columnCount <= 0 {
		return nil
	}
	bucketCount := min(len(shares), columnCount)
	buckets := make([]groupedDailyShareBucket, 0, bucketCount)
	for i := range bucketCount {
		start := i * len(shares) / bucketCount
		end := (i + 1) * len(shares) / bucketCount
		if end <= start {
			end = start + 1
		}
		buckets = append(buckets, buildGroupedDailyShareBucket(shares[start:end]))
	}
	return buckets
}

func buildGroupedDailyShareBucket(chunk []statspkg.GroupedDailyShare) groupedDailyShareBucket {
	bucket := groupedDailyShareBucket{
		Start: chunk[0].Date,
		End:   chunk[len(chunk)-1].Date,
	}
	versionTotals := make(map[string]int)
	for _, day := range chunk {
		if !day.HasActivity {
			continue
		}
		bucket.HasActivity = true
		bucket.Prompt += day.Prompt
		bucket.Total += day.Total
		for _, version := range day.Versions {
			versionTotals[version.Version] += version.Value
		}
	}
	bucket.Versions = sortGroupedDailyShareVersions(versionTotals)
	return bucket
}

func sortGroupedDailyShareVersions(values map[string]int) []statspkg.VersionValue {
	items := make([]statspkg.VersionValue, 0, len(values))
	for version, value := range values {
		if value <= 0 {
			continue
		}
		items = append(items, statspkg.VersionValue{Version: version, Value: value})
	}
	slices.SortFunc(items, func(left, right statspkg.VersionValue) int {
		switch {
		case left.Version < right.Version:
			return -1
		case left.Version > right.Version:
			return 1
		default:
			return right.Value - left.Value
		}
	})
	return items
}

func groupedDailyShareRate(bucket groupedDailyShareBucket) float64 {
	if bucket.Prompt <= 0 {
		return 0
	}
	return float64(bucket.Total) / float64(bucket.Prompt)
}

func renderGroupedDailyShareRow(
	buckets []groupedDailyShareBucket,
	slots []DailyRateBarSlot,
	level, plotHeight int,
	maxValue float64,
	axisLabelWidth int,
	colorByVersion map[string]color.Color,
) string {
	label := ""
	switch level {
	case plotHeight:
		label = groupedDailyShareAxisLabel(maxValue)
	case max((plotHeight+1)/2, 1):
		label = groupedDailyShareAxisLabel(maxValue / 2)
	case 1:
		label = groupedDailyShareAxisLabel(0)
	}
	prefix := FitToWidth(HistogramAxisLabel(label), axisLabelWidth) +
		" " + HistogramAxisLine("│") + " "
	cells := BlankDailyRateCells(DailyRatePlotWidth(slots))
	for i, bucket := range buckets {
		writeGroupedDailyShareSlot(cells, slots[i], bucket, level, plotHeight, maxValue, colorByVersion)
	}
	return ansi.Truncate(prefix+strings.Join(cells, ""), axisLabelWidth+3+len(cells), "…")
}

func writeGroupedDailyShareSlot(
	cells []string,
	slot DailyRateBarSlot,
	bucket groupedDailyShareBucket,
	level, plotHeight int,
	maxValue float64,
	colorByVersion map[string]color.Color,
) {
	if !bucket.HasActivity {
		if level == 1 && slot.Anchor < len(cells) {
			cells[slot.Anchor] = lipgloss.NewStyle().Foreground(ColorNormalDesc).Render("·")
		}
		return
	}
	rate := groupedDailyShareRate(bucket)
	totalHeight := ScaledFloatWidth(rate, maxValue, plotHeight)
	if totalHeight < level {
		return
	}
	values := make([]float64, 0, len(bucket.Versions))
	for _, version := range bucket.Versions {
		values = append(values, float64(version.Value))
	}
	segmentHeights := ResolveFloatSegmentHeights(totalHeight, values)
	remaining := level
	for i, version := range bucket.Versions {
		segmentHeight := segmentHeights[i]
		if remaining <= segmentHeight {
			fillGroupedDailySlot(cells, slot, colorByVersion[version.Version])
			return
		}
		remaining -= segmentHeight
	}
}

func fillGroupedDailySlot(cells []string, slot DailyRateBarSlot, fillColor color.Color) {
	for i := slot.Start; i < min(slot.End, len(cells)); i++ {
		cells[i] = lipgloss.NewStyle().Foreground(fillColor).Render("█")
	}
}

func renderGroupedDailyShareLabels(
	buckets []groupedDailyShareBucket,
	plotWidth int,
	slots []DailyRateBarSlot,
) string {
	if len(buckets) == 0 {
		return ""
	}
	dayBuckets := make([]DailyRateBucket, 0, len(buckets))
	for _, bucket := range buckets {
		dayBuckets = append(dayBuckets, DailyRateBucket{Start: bucket.Start, End: bucket.End})
	}
	return RenderDailyRateLabelLine(dayBuckets, plotWidth, slots)
}
