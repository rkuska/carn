package stats

import (
	"fmt"
	"math"
	"strings"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"

	el "github.com/rkuska/carn/internal/app/elements"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) splitActivitySeries() (string, []statspkg.SplitDailyValueSeries) {
	switch m.activityMetric {
	case metricSessions:
		return activityDailySessionsTitle, m.splitActivityResult.DailySessions
	case metricMessages:
		return activityDailyMessagesTitle, m.splitActivityResult.DailyMessages
	case metricTokens:
		return activityDailyTokensTitle, m.splitActivityResult.DailyTokens
	default:
		return activityDailySessionsTitle, m.splitActivityResult.DailySessions
	}
}

func (m statsModel) renderSplitActivityDailyChart(width, height int) string {
	_, series := m.splitActivitySeries()
	return renderChartWithSplitLegend(
		m.theme,
		width,
		splitDailyValueSeriesKeys(series),
		m.splitColors,
		splitChartMinWidth,
		func(chartWidth int) string {
			return renderSplitDailyValueChartBody(
				m.theme,
				series,
				chartWidth,
				height,
				m.splitColors,
				activitySplitYLabel(),
				1,
			)
		},
	)
}

func activitySplitYLabel() linechart.LabelFormatter {
	return func(_ int, v float64) string {
		rounded := math.Round(v)
		if math.Abs(v-rounded) < 0.05 {
			return statspkg.FormatNumber(int(rounded))
		}
		return strings.TrimRight(
			strings.TrimRight(fmt.Sprintf("%.1f", v), "0"),
			".",
		)
	}
}

var renderSplitDailyValueChartBody = (*el.Theme).RenderSplitDailyValueChartBody
