package elements

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

type HistogramLayout struct {
	BucketWidths []int
	GapWidth     int
	GraphWidth   int
}

func (t *Theme) RenderVerticalHistogram(title string, buckets []HistBucket, width, maxHeight int) string {
	return t.RenderVerticalHistogramWithColor(title, buckets, width, maxHeight, t.ColorChartBar)
}

func (t *Theme) RenderVerticalHistogramWithColor(
	title string,
	buckets []HistBucket,
	width, maxHeight int,
	barColor color.Color,
) string {
	body := t.RenderVerticalHistogramBody(buckets, width, maxHeight, barColor)
	if body == "" {
		return ""
	}
	return t.RenderStatsTitle(title) + "\n" + body
}

func (t *Theme) RenderVerticalHistogramBody(
	buckets []HistBucket,
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
	graphWidth := max(width-axisLabelWidth-3, 1)
	layout := ResolveHistogramLayout(graphWidth, len(buckets))
	gap := strings.Repeat(" ", layout.GapWidth)
	barStyle := lipgloss.NewStyle().Foreground(barColor)
	labelStyle := t.StyleHistogramValueLabel

	for level := maxHeight; level >= 1; level-- {
		lines = append(lines, renderHistogramLevel(
			t,
			renderBuckets,
			level,
			maxCount,
			maxHeight,
			axisLabelWidth,
			layout.BucketWidths,
			gap,
			width,
			barStyle,
			labelStyle,
		))
	}

	lines = append(lines, t.RenderHistogramAxis(axisLabelWidth, layout.GraphWidth, width))
	lines = append(lines, t.RenderHistogramLabels(buckets, axisLabelWidth, layout.BucketWidths, layout.GapWidth, width))

	return strings.Join(lines, "\n")
}

func histogramMaxCount(buckets []HistBucket) int {
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
	buckets []HistBucket,
	maxCount, maxHeight int,
) []histogramBucketRender {
	renderBuckets := make([]histogramBucketRender, 0, len(buckets))
	for _, bucket := range buckets {
		scaledHeight := ScaledWidth(bucket.Count, maxCount, maxHeight)
		labelLevel, labelInside := HistogramValueLabelPlacement(scaledHeight, maxHeight)
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

func HistogramValueLabelPlacement(scaledHeight, maxHeight int) (int, bool) {
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
	t *Theme,
	buckets []histogramBucketRender,
	level, maxCount, maxHeight, axisLabelWidth int,
	bucketWidths []int,
	gap string,
	width int,
	barStyle, labelStyle lipgloss.Style,
) string {
	var row strings.Builder
	row.Grow(width)
	for i, bucket := range buckets {
		if i == 0 {
			label := histogramLevelLabel(level, maxHeight, maxCount)
			row.WriteString(FitToWidth(t.HistogramAxisLabel(label), axisLabelWidth))
			row.WriteByte(' ')
			row.WriteString(t.HistogramAxisLine("│"))
			row.WriteByte(' ')
		} else {
			row.WriteString(gap)
		}
		row.WriteString(renderHistogramCell(bucket, level, bucketWidths[i], barStyle, labelStyle))
	}
	return ansi.Truncate(row.String(), width, "…")
}

func renderHistogramCell(
	bucket histogramBucketRender,
	level, bucketWidth int,
	barStyle, labelStyle lipgloss.Style,
) string {
	if bucketWidth <= 0 {
		return ""
	}
	if bucket.labelLevel == level {
		return renderHistogramValueLabel(bucket.label, bucketWidth, labelStyle)
	}
	if bucket.scaledHeight < level {
		return strings.Repeat(" ", bucketWidth)
	}
	return barStyle.Render(strings.Repeat("█", bucketWidth))
}

func renderHistogramValueLabel(label string, bucketWidth int, labelStyle lipgloss.Style) string {
	if bucketWidth <= 0 {
		return ""
	}
	text := FitToWidth(ansi.Truncate(label, bucketWidth, "…"), bucketWidth)
	return centerStyledText(labelStyle, text, bucketWidth)
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

func (t *Theme) RenderHistogramAxis(axisLabelWidth, graphWidth, width int) string {
	prefix := FitToWidth(t.HistogramAxisLabel("0"), axisLabelWidth) + " " + t.HistogramAxisLine("└")
	return ansi.Truncate(prefix+t.HistogramAxisLine(strings.Repeat("─", graphWidth)), width, "…")
}

func (t *Theme) RenderHistogramLabels(
	buckets []HistBucket,
	axisLabelWidth int,
	bucketWidths []int,
	gapWidth, width int,
) string {
	var row strings.Builder
	row.Grow(width)
	row.WriteString(strings.Repeat(" ", axisLabelWidth+3))
	gap := strings.Repeat(" ", gapWidth)
	for i, bucket := range buckets {
		if i > 0 {
			row.WriteString(gap)
		}
		if bucketWidths[i] <= 0 {
			continue
		}
		label := ansi.Truncate(bucket.Label, bucketWidths[i], "…")
		row.WriteString(centerStyledText(t.StyleHistogramAxisLabel, label, bucketWidths[i]))
	}
	return ansi.Truncate(row.String(), width, "…")
}

func ResolveHistogramLayout(graphWidth, bucketCount int) HistogramLayout {
	if graphWidth <= 0 || bucketCount <= 0 {
		return HistogramLayout{}
	}

	gapWidth := 1
	if graphWidth < bucketCount*2-1 {
		gapWidth = 0
	}
	bucketSpace := max(graphWidth-gapWidth*(bucketCount-1), 0)
	baseWidth := 0
	remainder := 0
	if bucketCount > 0 {
		baseWidth = bucketSpace / bucketCount
		remainder = bucketSpace % bucketCount
	}

	widths := make([]int, bucketCount)
	for i := range bucketCount {
		widths[i] = baseWidth
		if i < remainder {
			widths[i]++
		}
	}

	return HistogramLayout{
		BucketWidths: widths,
		GapWidth:     gapWidth,
		GraphWidth:   bucketSpace + gapWidth*(bucketCount-1),
	}
}

func (t *Theme) HistogramAxisLabel(text string) string {
	return t.StyleHistogramAxisLabel.Render(text)
}

func (t *Theme) HistogramAxisLine(text string) string {
	return t.StyleHistogramAxisLine.Render(text)
}
