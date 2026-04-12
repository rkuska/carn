package app

import (
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	statspkg "github.com/rkuska/carn/internal/stats"
)

type histogramBucketRender struct {
	scaledHeight int
	label        string
	labelLevel   int
	labelInside  bool
}

func renderVerticalHistogram(title string, buckets []histBucket, width, maxHeight int) string {
	return renderVerticalHistogramWithColor(title, buckets, width, maxHeight, colorChartBar)
}

func renderVerticalHistogramWithColor(
	title string,
	buckets []histBucket,
	width, maxHeight int,
	barColor color.Color,
) string {
	body := renderVerticalHistogramBody(buckets, width, maxHeight, barColor)
	if body == "" {
		return ""
	}
	return renderStatsTitle(title) + "\n" + body
}

func renderVerticalHistogramBody(
	buckets []histBucket,
	width, maxHeight int,
	barColor color.Color,
) string {
	if width <= 0 {
		return ""
	}

	lines := make([]string, 0, maxHeight+3)
	if len(buckets) == 0 {
		lines = append(lines, "No data")
		return strings.Join(lines, "\n")
	}

	maxHeight = max(maxHeight, 1)
	maxCount := histogramMaxCount(buckets)
	renderBuckets := histogramBucketRenders(buckets, maxCount, maxHeight)
	axisLabelWidth := max(lipgloss.Width(statspkg.FormatNumber(maxCount)), 1)
	gapWidth := 1
	graphWidth := max(width-axisLabelWidth-3, 1)
	bucketWidth := max((graphWidth-gapWidth*(len(buckets)-1))/len(buckets), 3)
	graphWidth = bucketWidth*len(buckets) + gapWidth*(len(buckets)-1)
	barStyle := lipgloss.NewStyle().Foreground(barColor)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))

	for level := maxHeight; level >= 1; level-- {
		lines = append(lines, renderHistogramLevel(
			renderBuckets,
			level,
			maxCount,
			maxHeight,
			axisLabelWidth,
			bucketWidth,
			gapWidth,
			width,
			barStyle,
			labelStyle,
		))
	}

	lines = append(lines, renderHistogramAxis(axisLabelWidth, graphWidth, width))
	lines = append(lines, renderHistogramLabels(buckets, axisLabelWidth, bucketWidth, gapWidth, width))

	return strings.Join(lines, "\n")
}

func histogramMaxCount(buckets []histBucket) int {
	maxCount := 0
	for _, bucket := range buckets {
		maxCount = max(maxCount, bucket.Count)
	}
	if maxCount == 0 {
		return 1
	}
	return maxCount
}

func histogramBucketRenders(
	buckets []histBucket,
	maxCount, maxHeight int,
) []histogramBucketRender {
	renderBuckets := make([]histogramBucketRender, 0, len(buckets))
	for _, bucket := range buckets {
		scaledHeight := scaledWidth(bucket.Count, maxCount, maxHeight)
		labelLevel, labelInside := histogramValueLabelPlacement(scaledHeight, maxHeight)
		label := bucket.Display
		if label == "" {
			label = statspkg.FormatNumber(bucket.Count)
		}
		renderBuckets = append(renderBuckets, histogramBucketRender{
			scaledHeight: scaledHeight,
			label:        label,
			labelLevel:   labelLevel,
			labelInside:  labelInside,
		})
	}
	return renderBuckets
}

func histogramValueLabelPlacement(scaledHeight, maxHeight int) (int, bool) {
	switch {
	case scaledHeight <= 0:
		return 1, false
	case scaledHeight >= maxHeight:
		return maxHeight, true
	default:
		return scaledHeight + 1, false
	}
}

func renderHistogramLevel(
	buckets []histogramBucketRender,
	level, maxCount, maxHeight, axisLabelWidth, bucketWidth, gapWidth, width int,
	barStyle, labelStyle lipgloss.Style,
) string {
	parts := make([]string, 0, len(buckets))
	for _, bucket := range buckets {
		parts = append(parts, renderHistogramCell(bucket, level, bucketWidth, barStyle, labelStyle))
	}
	prefix := fitToWidth(histogramAxisLabel(histogramLevelLabel(level, maxHeight, maxCount)), axisLabelWidth) +
		" " + histogramAxisLine("│") + " "
	return ansi.Truncate(prefix+strings.Join(parts, strings.Repeat(" ", gapWidth)), width, "…")
}

func renderHistogramCell(
	bucket histogramBucketRender,
	level, bucketWidth int,
	barStyle, labelStyle lipgloss.Style,
) string {
	if bucket.labelLevel == level {
		return renderHistogramValueLabel(bucket.label, bucketWidth, labelStyle)
	}
	if bucket.scaledHeight < level {
		return strings.Repeat(" ", bucketWidth)
	}
	return barStyle.Render(strings.Repeat("█", bucketWidth))
}

func renderHistogramValueLabel(label string, bucketWidth int, labelStyle lipgloss.Style) string {
	text := fitToWidth(ansi.Truncate(label, bucketWidth, "…"), bucketWidth)
	return lipgloss.PlaceHorizontal(bucketWidth, lipgloss.Center, labelStyle.Render(text))
}

func histogramLevelLabel(level, maxHeight, maxCount int) string {
	switch level {
	case maxHeight:
		return statspkg.FormatNumber(maxCount)
	case max((maxHeight+1)/2, 1):
		return statspkg.FormatNumber(int(math.Round(float64(maxCount) / 2)))
	default:
		return ""
	}
}

func renderHistogramAxis(axisLabelWidth, graphWidth, width int) string {
	prefix := fitToWidth(histogramAxisLabel("0"), axisLabelWidth) + " " + histogramAxisLine("└")
	return ansi.Truncate(prefix+histogramAxisLine(strings.Repeat("─", graphWidth)), width, "…")
}

func renderHistogramLabels(
	buckets []histBucket,
	axisLabelWidth, bucketWidth, gapWidth, width int,
) string {
	labels := make([]string, 0, len(buckets))
	for _, bucket := range buckets {
		label := ansi.Truncate(bucket.Label, bucketWidth, "…")
		labels = append(labels, lipgloss.PlaceHorizontal(bucketWidth, lipgloss.Center, histogramAxisLabel(label)))
	}
	return ansi.Truncate(
		strings.Repeat(" ", axisLabelWidth+3)+strings.Join(labels, strings.Repeat(" ", gapWidth)),
		width,
		"…",
	)
}

func histogramAxisLabel(text string) string {
	return lipgloss.NewStyle().Foreground(colorNormalDesc).Render(text)
}

func histogramAxisLine(text string) string {
	return lipgloss.NewStyle().Foreground(colorSecondary).Render(text)
}
