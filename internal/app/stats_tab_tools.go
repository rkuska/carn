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
	if m.toolsGrouped {
		return m.renderGroupedToolsTab(width, tools)
	}
	chips := renderSummaryChips([]chip{
		{Label: "total calls", Value: statspkg.FormatNumber(tools.TotalCalls)},
		{Label: "avg/session", Value: formatFloat(tools.AverageCallsPerSession)},
		{Label: "error rate", Value: toolRateChipValue(tools.ErrorRate)},
		{Label: "rejected", Value: toolRateChipValue(tools.RejectionRate)},
		{Label: "read", Value: formatToolShare(tools.ReadWriteBashShare.Read)},
		{Label: "write", Value: formatToolShare(tools.ReadWriteBashShare.Write)},
		{Label: "bash", Value: formatToolShare(tools.ReadWriteBashShare.Bash)},
	}, width)

	topTools := make([]barItem, 0, len(tools.TopTools))
	for _, item := range tools.TopTools {
		topTools = append(topTools, barItem{Label: item.Name, Value: item.Count})
	}

	callBuckets := make([]histBucket, 0, len(tools.CallsPerSession))
	for _, bucket := range tools.CallsPerSession {
		callBuckets = append(callBuckets, histBucket{Label: bucket.Label, Count: bucket.Count})
	}

	usageCharts := renderStatsLanePair(
		width,
		30,
		"Tool Calls/Session",
		m.toolsLaneCursor == 0,
		func(bodyWidth int) string {
			return renderVerticalHistogramBody(
				callBuckets,
				bodyWidth,
				toolCallsChartHeight(len(tools.ToolErrorRates)),
				colorChartBar,
			)
		},
		"Top Tools",
		m.toolsLaneCursor == 1,
		func(bodyWidth int) string {
			return renderHorizontalBarsBody(topTools, bodyWidth, colorChartBar)
		},
	)

	qualityCharts := renderStatsLanePair(
		width,
		30,
		"Tool Error Rate",
		m.toolsLaneCursor == 2,
		func(bodyWidth int) string {
			return renderToolRateChartBody(tools.ToolErrorRates, bodyWidth, colorChartError, true)
		},
		statsRejectedSuggestionsTitle,
		m.toolsLaneCursor == 3,
		func(bodyWidth int) string {
			return renderToolRateChartBody(tools.ToolRejectRates, bodyWidth, colorPrimary, false)
		},
	)
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", chips, usageCharts, qualityCharts, m.renderActiveMetricDetail(width))
}

func toolCallsChartHeight(errorRateCount int) int {
	height := 1
	if errorRateCount <= 2 {
		return max(height, 3)
	}
	height = errorRateCount - 2
	return max(height, 3)
}

func renderToolRateChart(
	title string,
	rates []statspkg.ToolRateStat,
	width int,
	barColor color.Color,
	showCount bool,
) string {
	body := renderToolRateChartBody(rates, width, barColor, showCount)
	if body == "" {
		return ""
	}
	return renderStatsTitle(title) + "\n" + body
}

func renderToolRateChartBody(
	rates []statspkg.ToolRateStat,
	width int,
	barColor color.Color,
	showCount bool,
) string {
	if width <= 0 {
		return ""
	}

	lines := make([]string, 0, len(rates)+1)
	if len(rates) == 0 {
		lines = append(lines, "No data")
		return strings.Join(lines, "\n")
	}

	labelWidth := 16
	valueWidth := 1
	maxRate := 0
	values := make([]string, len(rates))
	for i, item := range rates {
		values[i] = renderToolRateValue(item, showCount)
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
	return formatToolRatePercent(rate)
}

func renderToolRateValue(item statspkg.ToolRateStat, showCount bool) string {
	percentage := formatToolRatePercent(item.Rate)
	if !showCount {
		return percentage
	}
	return percentage + " " + lipgloss.NewStyle().
		Foreground(colorNormalDesc).
		Render(fmt.Sprintf("(%s)", statspkg.FormatNumber(item.Count)))
}

func formatToolRatePercent(rate float64) string {
	if rate > 0 && rate < 0.1 {
		return "<0.1%"
	}
	return fmt.Sprintf("%.1f%%", rate)
}

func formatToolShare(value float64) string {
	return fmt.Sprintf("%.0f%%", value)
}
