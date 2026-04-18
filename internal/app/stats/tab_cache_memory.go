package stats

import (
	"fmt"
	"math"

	statspkg "github.com/rkuska/carn/internal/stats"
)

const cacheFirstTurnLaneTitle = "First-Turn Cold Cache by Version (Claude)"

func cacheFirstTurnBuckets(stats []statspkg.CacheFirstTurnVersionStat) []histBucket {
	if len(stats) == 0 {
		return nil
	}
	buckets := make([]histBucket, 0, len(stats))
	for _, stat := range stats {
		buckets = append(buckets, histBucket{
			Label:   stat.Version,
			Count:   int(math.Round(stat.ZeroReadRate * 100)),
			Display: fmt.Sprintf("%.0f%%", stat.ZeroReadRate*100),
		})
	}
	return buckets
}

func (m statsModel) renderCacheFirstTurnLane(cache statspkg.Cache, width int, selected bool) string {
	body := renderVerticalHistogramBody(
		m.theme,
		cacheFirstTurnBuckets(cache.FirstTurnByVersion),
		statsLaneBodyWidth(width),
		8,
		m.theme.ColorChartToken,
	)
	return renderStatsLaneBox(m.theme, cacheFirstTurnLaneTitle, selected, width, body)
}

func (m statsModel) renderCacheFirstTurnMetricDetail(cache statspkg.Cache, width int) string {
	stats := cache.FirstTurnByVersion
	worstVersion := noDataLabel
	worstRate := noDataLabel
	overallRate := noDataLabel
	compared := noDataLabel

	if len(stats) > 0 {
		worst := stats[0]
		totalSessions := 0
		totalZeros := 0
		for _, stat := range stats {
			if stat.ZeroReadRate > worst.ZeroReadRate {
				worst = stat
			}
			totalSessions += stat.SessionCount
			totalZeros += stat.ZeroCount
		}
		worstVersion = worst.Version
		worstRate = fmt.Sprintf("%.0f%%", worst.ZeroReadRate*100)
		if totalSessions > 0 {
			overallRate = fmt.Sprintf("%.0f%%", float64(totalZeros)/float64(totalSessions)*100)
		}
		compared = statspkg.FormatNumber(len(stats))
	}

	return m.renderStatsMetricDetailBody(cacheFirstTurnLaneTitle, width, []chip{
		{Label: "worst version", Value: worstVersion},
		{Label: "worst cold rate", Value: worstRate},
		{Label: "overall cold rate", Value: overallRate},
		{Label: "versions compared", Value: compared},
	},
		m.metricDetailLine("Question", "Does a Claude release regress cold-start priming?"),
		m.metricDetailLine(
			"Reading",
			"Bars show the share of first turns that came in with zero cache reads. "+
				"Higher bars mean the release failed to warm the cache for more sessions.",
		),
	)
}
