package app

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type stackedHistSegment struct {
	Value int
	Color color.Color
}

type stackedHistBucket struct {
	Label    string
	Total    int
	Segments []stackedHistSegment
}

func renderVerticalStackedHistogramBody(
	buckets []stackedHistBucket,
	width, maxHeight int,
	yLabel func(int) string,
) string {
	if width <= 0 {
		return ""
	}
	if len(buckets) == 0 {
		return "No data"
	}

	maxHeight = max(maxHeight, 1)
	maxTotal, renderBuckets := buildStackedHistogramBucketRenders(buckets, maxHeight)
	topLabel := yLabel(maxTotal)
	midLabel := yLabel(maxTotal / 2)
	axisLabelWidth := max(lipgloss.Width(topLabel), lipgloss.Width(midLabel), lipgloss.Width(yLabel(0)), 1)
	graphWidth := max(width-axisLabelWidth-3, 1)
	layout := resolveHistogramLayout(graphWidth, len(buckets))

	lines := make([]string, 0, maxHeight+2)
	for level := maxHeight; level >= 1; level-- {
		label := stackedHistogramAxisLabel(level, maxHeight, maxTotal, yLabel)
		prefix := fitToWidth(histogramAxisLabel(label), axisLabelWidth) +
			" " + histogramAxisLine("│") + " "
		lines = append(lines, ansi.Truncate(prefix+renderStackedHistogramLevel(renderBuckets, level, layout), width, "…"))
	}
	lines = append(lines, renderHistogramAxis(axisLabelWidth, layout.graphWidth, width))
	lines = append(lines, renderHistogramLabels(
		histBucketsFromStacked(buckets),
		axisLabelWidth,
		layout.bucketWidths,
		layout.gapWidth,
		width,
	))
	return strings.Join(lines, "\n")
}

func buildStackedHistogramBucketRenders(
	buckets []stackedHistBucket,
	maxHeight int,
) (int, []stackedHistogramBucketRender) {
	maxTotal := 0
	renderBuckets := make([]stackedHistogramBucketRender, 0, len(buckets))
	for _, bucket := range buckets {
		maxTotal = max(maxTotal, bucket.Total)
		renderBuckets = append(renderBuckets, stackedHistogramBucketRender{
			Label: bucket.Label,
			Total: bucket.Total,
		})
	}
	if maxTotal == 0 {
		maxTotal = 1
	}
	for i, bucket := range buckets {
		renderBuckets[i].Height = scaledWidth(bucket.Total, maxTotal, maxHeight)
		renderBuckets[i].Segments = buildStackedHistogramSegments(bucket.Segments, renderBuckets[i].Height)
	}
	return maxTotal, renderBuckets
}

func buildStackedHistogramSegments(
	segments []stackedHistSegment,
	totalHeight int,
) []stackedHistogramSegmentRender {
	values := make([]float64, 0, len(segments))
	for _, segment := range segments {
		values = append(values, float64(max(segment.Value, 0)))
	}
	heights := resolveFloatSegmentHeights(totalHeight, values)
	rendered := make([]stackedHistogramSegmentRender, 0, len(segments))
	for i, segment := range segments {
		rendered = append(rendered, stackedHistogramSegmentRender{
			Height: heights[i],
			Color:  segment.Color,
		})
	}
	return rendered
}

func stackedHistogramAxisLabel(
	level, maxHeight, maxTotal int,
	yLabel func(int) string,
) string {
	switch level {
	case maxHeight:
		return yLabel(maxTotal)
	case max((maxHeight+1)/2, 1):
		return yLabel(maxTotal / 2)
	default:
		return ""
	}
}

type stackedHistogramBucketRender struct {
	Label    string
	Total    int
	Height   int
	Segments []stackedHistogramSegmentRender
}

type stackedHistogramSegmentRender struct {
	Height int
	Color  color.Color
}

func renderStackedHistogramLevel(
	buckets []stackedHistogramBucketRender,
	level int,
	layout histogramLayout,
) string {
	parts := make([]string, 0, len(buckets))
	for i, bucket := range buckets {
		width := layout.bucketWidths[i]
		if width <= 0 {
			parts = append(parts, "")
			continue
		}
		parts = append(parts, renderStackedHistogramCell(bucket, level, width))
	}
	return strings.Join(parts, strings.Repeat(" ", layout.gapWidth))
}

func renderStackedHistogramCell(
	bucket stackedHistogramBucketRender,
	level int,
	width int,
) string {
	if bucket.Height < level || width <= 0 {
		return strings.Repeat(" ", width)
	}
	remaining := level
	for _, segment := range bucket.Segments {
		if segment.Height <= 0 {
			continue
		}
		if remaining <= segment.Height {
			return lipgloss.NewStyle().
				Foreground(segment.Color).
				Render(strings.Repeat("█", width))
		}
		remaining -= segment.Height
	}
	return strings.Repeat(" ", width)
}

func histBucketsFromStacked(buckets []stackedHistBucket) []histBucket {
	items := make([]histBucket, 0, len(buckets))
	for _, bucket := range buckets {
		items = append(items, histBucket{Label: bucket.Label, Count: bucket.Total})
	}
	return items
}
