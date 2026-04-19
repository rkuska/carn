package elements

import (
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func dailyRateChartScale(
	rates []statspkg.DailyRate,
	yFormatter linechart.LabelFormatter,
) (float64, int) {
	maxValue := 0.01
	for _, rate := range rates {
		if rate.HasActivity && rate.Rate > maxValue {
			maxValue = rate.Rate
		}
	}
	return maxValue, dailyRateAxisLabelWidth(maxValue, yFormatter)
}

func dailyRateAxisLabelWidth(
	maxValue float64,
	yFormatter linechart.LabelFormatter,
) int {
	topLabel := yFormatter(0, maxValue)
	midLabel := yFormatter(0, maxValue/2)
	bottomLabel := yFormatter(0, 0)
	return max(
		lipgloss.Width(topLabel),
		lipgloss.Width(midLabel),
		lipgloss.Width(bottomLabel),
		1,
	)
}
