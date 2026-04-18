package stats

import (
	"fmt"
	"image/color"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	tslc "github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"

	el "github.com/rkuska/carn/internal/app/elements"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) renderActivityTab(width, height int) string {
	activity := m.snapshot.Activity
	chips := renderSummaryChips(m.theme, []chip{
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
		m.theme,
		chartTitle,
		m.activityLaneCursor == 0,
		width,
		renderDailyActivityChartBody(
			m.theme,
			counts,
			max(statsLaneBodyWidth(width), 10),
			chartHeight,
			m.theme.ColorChartTime,
		),
	)
	heatmap := renderStatsLaneBox(
		m.theme,
		"Activity Heatmap",
		m.activityLaneCursor == 1,
		width,
		renderActivityHeatmapBody(m.theme, activity.Heatmap, statsLaneBodyWidth(width)),
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
	theme *el.Theme,
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
			lipgloss.NewStyle().Foreground(theme.ColorSecondary),
			lipgloss.NewStyle().Foreground(theme.ColorNormalDesc),
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
	theme *el.Theme,
	rates []statspkg.DailyRate,
	width, height int,
	lineColor color.Color,
	yFormatter linechart.LabelFormatter,
) string {
	return renderDailyRateColumnChart(theme, rates, width, height, lineColor, yFormatter)
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
