package app

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	statspkg "github.com/rkuska/carn/internal/stats"
)

const statsRejectedSuggestionsTitle = "Rejected Suggestions"

func (m statsModel) renderToolsTab(width int) string {
	tools := m.snapshot.Tools
	chips := renderSummaryChips([]chip{
		{Label: "total calls", Value: statspkg.FormatNumber(tools.TotalCalls)},
		{Label: "avg/session", Value: formatFloat(tools.AverageCallsPerSession)},
		{Label: "error rate", Value: toolRateChipValue(tools.ErrorRate)},
		{Label: "rejected", Value: toolRateChipValue(tools.RejectionRate)},
		{Label: "read:write:bash", Value: formatToolRatio(tools.ReadWriteBashRatio)},
	}, width)

	topTools := make([]barItem, 0, len(tools.TopTools))
	for _, item := range tools.TopTools {
		topTools = append(topTools, barItem{Label: item.Name, Value: item.Count})
	}

	callBuckets := make([]histBucket, 0, len(tools.CallsPerSession))
	for _, bucket := range tools.CallsPerSession {
		callBuckets = append(callBuckets, histBucket{Label: bucket.Label, Count: bucket.Count})
	}

	topChart := centerBlock(
		renderHorizontalBars("Top Tools", topTools, min(width, 72), colorChartBar),
		width,
	)
	errorChartWidth := max((width-3)/2, 30)
	errorChart := renderToolRateChart("Tool Error Rate", tools.ToolErrorRates, errorChartWidth, colorChartError)
	rejectedChart := centerBlock(
		renderToolRateChart(statsRejectedSuggestionsTitle, tools.ToolRejectRates, min(width, 72), colorPrimary),
		width,
	)
	sideBySide := renderSideBySide(
		renderVerticalHistogram("Tool Calls/Session", callBuckets, max((width-3)/2, 30), 8),
		errorChart,
		width,
	)
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", chips, topChart, sideBySide, rejectedChart)
}

func renderToolRateChart(
	title string,
	rates []statspkg.ToolRateStat,
	width int,
	barColor color.Color,
) string {
	if width <= 0 {
		return ""
	}

	lines := []string{renderStatsTitle(title)}
	if len(rates) == 0 {
		lines = append(lines, "No data")
		return strings.Join(lines, "\n")
	}

	labelWidth := 16
	valueWidth := 1
	maxRate := 0
	values := make([]string, len(rates))
	for i, item := range rates {
		values[i] = fmt.Sprintf("%.1f%%", item.Rate)
		valueWidth = max(valueWidth, lipgloss.Width(values[i]))
		maxRate = max(maxRate, int(math.Round(item.Rate*10)))
	}
	barWidth := max(width-labelWidth-valueWidth-2, 1)
	barStyle := lipgloss.NewStyle().Foreground(barColor)

	for i, item := range rates {
		scaledRate := int(math.Round(item.Rate * 10))
		fillWidth := scaledWidth(scaledRate, maxRate, barWidth)
		bar := barStyle.Render(strings.Repeat("█", fillWidth)) +
			strings.Repeat(" ", max(barWidth-fillWidth, 0))
		label := fitToWidth(ansi.Truncate(item.Name, labelWidth, "…"), labelWidth)
		value := fitToWidth(values[i], valueWidth)
		lines = append(lines, ansi.Truncate(label+" "+bar+" "+value, width, "…"))
	}

	return strings.Join(lines, "\n")
}

func toolRateChipValue(rate float64) string {
	return fmt.Sprintf("%.1f%%", rate)
}
