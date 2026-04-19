package elements

import (
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"

	statspkg "github.com/rkuska/carn/internal/stats"
)

type DailyRateBucket struct {
	Start       time.Time
	End         time.Time
	Rate        float64
	HasActivity bool
	HasInactive bool
}

type DailyRateBarSlot struct {
	Start  int
	End    int
	Anchor int
}

type dailyRateLabelPlacement struct {
	Align  dailyRateLabelAlign
	Anchor int
	Label  string
}

type dailyRateLabelAlign int

const (
	dailyRateLabelAlignLeft dailyRateLabelAlign = iota
	dailyRateLabelAlignCenter
	dailyRateLabelAlignRight
)

func (t *Theme) RenderDailyRateColumnChart(
	rates []statspkg.DailyRate,
	width, height int,
	barColor color.Color,
	yFormatter linechart.LabelFormatter,
) string {
	maxValue, _ := dailyRateChartScale(rates, yFormatter)
	return t.renderDailyRateColumnChartWithMaxValue(
		rates,
		width,
		height,
		barColor,
		yFormatter,
		maxValue,
	)
}

func (t *Theme) renderDailyRateColumnChartWithMaxValue(
	rates []statspkg.DailyRate,
	width, height int,
	barColor color.Color,
	yFormatter linechart.LabelFormatter,
	maxValue float64,
) string {
	if width <= 0 {
		return ""
	}
	if len(rates) == 0 {
		return NoDataLabel
	}

	if maxValue <= 0 {
		maxValue = 0.01
	}
	axisLabelWidth := dailyRateAxisLabelWidth(maxValue, yFormatter)
	plotWidth := max(width-axisLabelWidth-3, 1)
	buckets := BucketDailyRates(rates, plotWidth)
	if len(buckets) == 0 {
		return NoDataLabel
	}

	slots := DailyRateBarSlots(len(buckets), plotWidth)
	plotHeight, showLabels := DailyRateChartDimensions(height)
	barStyle := lipgloss.NewStyle().Foreground(barColor)
	inactiveStyle := lipgloss.NewStyle().Foreground(t.ColorNormalDesc)
	lines := t.renderDailyRateRows(
		buckets,
		slots,
		plotHeight,
		maxValue,
		axisLabelWidth,
		yFormatter,
		barStyle,
		inactiveStyle,
	)
	if showLabels {
		lines = append(lines, strings.Repeat(" ", axisLabelWidth+3)+RenderDailyRateLabelLine(buckets, plotWidth, slots))
	}

	return strings.Join(lines, "\n")
}

func DailyRateChartDimensions(height int) (int, bool) {
	if height <= 1 {
		return 1, false
	}
	return max(height-1, 1), true
}

func (t *Theme) renderDailyRateRows(
	buckets []DailyRateBucket,
	slots []DailyRateBarSlot,
	plotHeight int,
	maxValue float64,
	axisLabelWidth int,
	yFormatter linechart.LabelFormatter,
	barStyle, inactiveStyle lipgloss.Style,
) []string {
	lines := make([]string, 0, plotHeight)
	for level := plotHeight; level >= 1; level-- {
		lines = append(lines, t.renderDailyRateRow(
			buckets,
			slots,
			level,
			plotHeight,
			maxValue,
			axisLabelWidth,
			yFormatter,
			barStyle,
			inactiveStyle,
		))
	}
	return lines
}

func (t *Theme) renderDailyRateRow(
	buckets []DailyRateBucket,
	slots []DailyRateBarSlot,
	level, plotHeight int,
	maxValue float64,
	axisLabelWidth int,
	yFormatter linechart.LabelFormatter,
	barStyle, inactiveStyle lipgloss.Style,
) string {
	label := dailyRateAxisLabel(level, plotHeight, maxValue, yFormatter)
	prefix := FitToWidth(t.HistogramAxisLabel(label), axisLabelWidth) +
		" " + t.HistogramAxisLine("│") + " "
	cells := BlankDailyRateCells(DailyRatePlotWidth(slots))
	for i, bucket := range buckets {
		cell, fill := renderDailyRateBucketLevel(
			bucket,
			level,
			plotHeight,
			maxValue,
			barStyle,
			inactiveStyle,
		)
		writeDailyRateSlot(cells, slots[i], cell, fill)
	}
	return prefix + strings.Join(cells, "")
}

func BlankDailyRateCells(plotWidth int) []string {
	cells := make([]string, plotWidth)
	for i := range cells {
		cells[i] = " "
	}
	return cells
}

func BucketDailyRates(rates []statspkg.DailyRate, columnCount int) []DailyRateBucket {
	if len(rates) == 0 || columnCount <= 0 {
		return nil
	}

	bucketCount := min(len(rates), columnCount)
	buckets := make([]DailyRateBucket, 0, bucketCount)
	for i := range bucketCount {
		start := i * len(rates) / bucketCount
		end := (i + 1) * len(rates) / bucketCount
		if end <= start {
			end = start + 1
		}

		buckets = append(buckets, buildDailyRateBucket(rates[start:end]))
	}

	return buckets
}

func buildDailyRateBucket(chunk []statspkg.DailyRate) DailyRateBucket {
	bucket := DailyRateBucket{
		Start: chunk[0].Date,
		End:   chunk[len(chunk)-1].Date,
	}

	activeCount := 0
	for _, rate := range chunk {
		if rate.HasActivity {
			bucket.HasActivity = true
			bucket.Rate += rate.Rate
			activeCount++
			continue
		}
		bucket.HasInactive = true
	}
	if activeCount > 0 {
		bucket.Rate /= float64(activeCount)
	}
	return bucket
}

func dailyRateAxisLabel(
	level, plotHeight int,
	maxValue float64,
	yFormatter linechart.LabelFormatter,
) string {
	switch level {
	case plotHeight:
		return yFormatter(0, maxValue)
	case max((plotHeight+1)/2, 1):
		return yFormatter(0, maxValue/2)
	case 1:
		return yFormatter(0, 0)
	default:
		return ""
	}
}

func renderDailyRateBucketLevel(
	bucket DailyRateBucket,
	level, plotHeight int,
	maxValue float64,
	barStyle, inactiveStyle lipgloss.Style,
) (string, bool) {
	if !bucket.HasActivity {
		if level == 1 {
			return inactiveStyle.Render("·"), false
		}
		return "", false
	}
	if bucket.Rate <= 0 {
		if level == 1 {
			return barStyle.Render("▁"), true
		}
		return "", false
	}

	barHeight := ScaledWidth(int(bucket.Rate*1000), int(maxValue*1000), plotHeight)
	if barHeight >= level {
		return barStyle.Render("█"), true
	}
	return "", false
}

func writeDailyRateSlot(cells []string, slot DailyRateBarSlot, cell string, fill bool) {
	if cell == "" || slot.Start < 0 || slot.End <= slot.Start || slot.Start >= len(cells) {
		return
	}
	end := min(slot.End, len(cells))
	if !fill {
		anchor := min(max(slot.Anchor, slot.Start), end-1)
		cells[anchor] = cell
		return
	}
	for i := slot.Start; i < end; i++ {
		cells[i] = cell
	}
}

func DailyRatePlotWidth(slots []DailyRateBarSlot) int {
	if len(slots) == 0 {
		return 0
	}
	return slots[len(slots)-1].End
}

func DailyRateBarSlots(bucketCount, plotWidth int) []DailyRateBarSlot {
	if bucketCount <= 0 || plotWidth <= 0 {
		return nil
	}

	barWidth := dailyRateBarWidth(bucketCount, plotWidth)
	gaps := dailyRateBarGaps(bucketCount, plotWidth, barWidth)
	slots := make([]DailyRateBarSlot, 0, bucketCount)
	start := 0
	for i := range bucketCount {
		end := start + barWidth
		slots = append(slots, buildDailyRateBarSlot(start, end))
		if i < len(gaps) {
			start = end + gaps[i]
		}
	}
	return slots
}

func dailyRateBarGapWidth(bucketCount, plotWidth int) int {
	if bucketCount <= 1 {
		return 0
	}
	if plotWidth >= bucketCount*2-1 {
		return 1
	}
	return 0
}

func dailyRateBarWidth(bucketCount, plotWidth int) int {
	gapCount := max(bucketCount-1, 0)
	return max((plotWidth-gapCount*dailyRateBarGapWidth(bucketCount, plotWidth))/bucketCount, 1)
}

func dailyRateBarGaps(bucketCount, plotWidth, barWidth int) []int {
	gapCount := max(bucketCount-1, 0)
	if gapCount == 0 {
		return nil
	}

	baseGapWidth := dailyRateBarGapWidth(bucketCount, plotWidth)
	gaps := make([]int, gapCount)
	for i := range gaps {
		gaps[i] = baseGapWidth
	}

	extra := plotWidth - bucketCount*barWidth - gapCount*baseGapWidth
	for i := range extra {
		gaps[(i+1)*gapCount/(extra+1)]++
	}
	return gaps
}

func buildDailyRateBarSlot(start, end int) DailyRateBarSlot {
	width := end - start
	return DailyRateBarSlot{
		Start:  start,
		End:    end,
		Anchor: start + max(width-1, 0)/2,
	}
}

func RenderDailyRateLabelLine(buckets []DailyRateBucket, plotWidth int, slots []DailyRateBarSlot) string {
	if len(buckets) == 0 || plotWidth <= 0 || len(slots) != len(buckets) {
		return ""
	}

	runes := []rune(strings.Repeat(" ", plotWidth))
	used := make([]bool, plotWidth)
	for _, placement := range dailyRateLabelPlacements(buckets, slots) {
		if placement.Label == "" {
			continue
		}
		placeDailyRateLabel(runes, used, plotWidth, placement)
	}
	return string(runes)
}

func dailyRateLabelPlacements(
	buckets []DailyRateBucket,
	slots []DailyRateBarSlot,
) []dailyRateLabelPlacement {
	placements := []dailyRateLabelPlacement{{
		Anchor: slots[0].Anchor,
		Align:  dailyRateLabelAlignLeft,
		Label:  buckets[0].Start.Format("01/02"),
	}}
	if len(buckets) == 1 {
		return placements
	}

	placements = append(placements, dailyRateLabelPlacement{
		Anchor: slots[len(slots)-1].Anchor,
		Align:  dailyRateLabelAlignRight,
		Label:  buckets[len(buckets)-1].End.Format("01/02"),
	})
	if len(buckets) > 2 {
		mid := len(buckets) / 2
		placements = append(placements, dailyRateLabelPlacement{
			Anchor: slots[mid].Anchor,
			Align:  dailyRateLabelAlignCenter,
			Label:  buckets[mid].Start.Format("01/02"),
		})
	}
	return placements
}

func placeDailyRateLabel(
	runes []rune,
	used []bool,
	plotWidth int,
	placement dailyRateLabelPlacement,
) bool {
	start, ok := dailyRateLabelStart(plotWidth, placement)
	if !ok {
		return false
	}

	labelWidth := len([]rune(placement.Label))
	for i := range labelWidth {
		if used[start+i] {
			return false
		}
	}
	for i, r := range placement.Label {
		runes[start+i] = r
		used[start+i] = true
	}
	return true
}

func dailyRateLabelStart(plotWidth int, placement dailyRateLabelPlacement) (int, bool) {
	labelWidth := len([]rune(placement.Label))
	if labelWidth == 0 || labelWidth > plotWidth {
		return 0, false
	}

	switch placement.Align {
	case dailyRateLabelAlignLeft:
		return 0, true
	case dailyRateLabelAlignCenter:
		return max(min(placement.Anchor-labelWidth/2, plotWidth-labelWidth), 0), true
	case dailyRateLabelAlignRight:
		return max(plotWidth-labelWidth, 0), true
	default:
		return 0, false
	}
}
