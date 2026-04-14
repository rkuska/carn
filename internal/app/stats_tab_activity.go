package app

import (
	"fmt"
	"image/color"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	tslc "github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderActivityTab(width, height int) string {
	activity := m.snapshot.Activity
	chips := renderSummaryChips([]chip{
		{Label: "active days", Value: fmt.Sprintf("%d/%d", activity.ActiveDays, activity.TotalDays)},
		{Label: "current streak", Value: statspkg.FormatNumber(activity.CurrentStreak)},
		{Label: "longest streak", Value: statspkg.FormatNumber(activity.LongestStreak)},
	}, width)

	chartTitle, counts := m.activitySeries()
	chartHeight := 12
	if height < 18 {
		chartHeight = max(height-6, 6)
	}

	lineChart := renderStatsLaneBox(
		chartTitle,
		m.activityLaneCursor == 0,
		width,
		renderDailyActivityChartBody(
			counts,
			max(statsLaneBodyWidth(width), 10),
			chartHeight,
			colorChartTime,
		),
	)
	heatmap := renderStatsLaneBox(
		"Activity Heatmap",
		m.activityLaneCursor == 1,
		width,
		renderActivityHeatmapBody(activity.Heatmap, statsLaneBodyWidth(width)),
	)
	return joinSections(chips, lineChart, heatmap, m.renderActiveMetricDetail(width))
}

func (m statsModel) activitySeries() (string, []statspkg.DailyCount) {
	switch m.activityMetric {
	case metricSessions:
		return "Daily Sessions", m.snapshot.Activity.DailySessions
	case metricMessages:
		return "Daily Messages", m.snapshot.Activity.DailyMessages
	case metricTokens:
		return "Daily Tokens", m.snapshot.Activity.DailyTokens
	default:
		return "Daily Sessions", m.snapshot.Activity.DailySessions
	}
}

func renderDailyActivityChartBody(
	counts []statspkg.DailyCount,
	width, height int,
	lineColor color.Color,
) string {
	lines := make([]string, 0, 2)
	if len(counts) == 0 {
		lines = append(lines, "No data")
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	maxValue := 1
	start, end := activityChartRange(counts)
	for _, count := range counts {
		maxValue = max(maxValue, count.Count)
	}

	chart := tslc.New(
		width,
		height,
		tslc.WithTimeRange(start, end),
		tslc.WithYRange(0, float64(maxValue)),
		tslc.WithLineStyle(runes.ArcLineStyle),
		tslc.WithStyle(lipgloss.NewStyle().Foreground(lineColor)),
		tslc.WithAxesStyles(
			lipgloss.NewStyle().Foreground(colorSecondary),
			lipgloss.NewStyle().Foreground(colorNormalDesc),
		),
	)
	for _, count := range counts {
		chart.Push(tslc.TimePoint{Time: count.Date, Value: float64(count.Count)})
	}
	chart.Draw()
	lines = append(lines, chart.View())
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func renderDailyRateChartBody(
	rates []statspkg.DailyRate,
	width, height int,
	lineColor color.Color,
	yFormatter linechart.LabelFormatter,
) string {
	return renderDailyRateColumnChart(rates, width, height, lineColor, yFormatter)
}

func activityChartRange(counts []statspkg.DailyCount) (time.Time, time.Time) {
	if len(counts) == 0 {
		return time.Time{}, time.Time{}
	}

	start := counts[0].Date
	end := counts[len(counts)-1].Date
	if !end.After(start) {
		end = start.Add(24 * time.Hour)
	}
	return start, end
}
