package stats

import (
	el "github.com/rkuska/carn/internal/app/elements"
	statspkg "github.com/rkuska/carn/internal/stats"
)

func initPaletteForTest(hasDarkBG bool) {
	el.InitPaletteForTest(hasDarkBG)
	syncPaletteFromElements()
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

func helpItemKeys(items []helpItem) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	return keys
}
