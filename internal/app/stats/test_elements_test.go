package stats

import (
	el "github.com/rkuska/carn/internal/app/elements"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func initPaletteForTest(hasDarkBG bool) {
	_ = hasDarkBG
}

func heatmapIntervalCells(cells [7][24]int) [7][6]int {
	return el.HeatmapIntervalCells(cells)
}

func heatmapCellWidth(width int) int {
	return el.HeatmapCellWidth(width)
}

func histogramValueLabelPlacement(scaledHeight, maxHeight int) (int, bool) {
	return el.HistogramValueLabelPlacement(scaledHeight, maxHeight)
}

func bucketDailyRates(rates []statspkg.DailyRate, columnCount int) []dailyRateBucket {
	return el.BucketDailyRates(rates, columnCount)
}
