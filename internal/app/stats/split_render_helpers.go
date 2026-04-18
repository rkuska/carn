package stats

import (
	"image/color"
	"slices"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func splitHistSegments(
	values []statspkg.SplitValue,
	colorByKey map[string]color.Color,
) []stackedHistSegment {
	segments := make([]stackedHistSegment, 0, len(values))
	for _, value := range values {
		segments = append(segments, stackedHistSegment{
			Value: value.Value,
			Color: colorByKey[value.Key],
		})
	}
	return segments
}

func splitRowSegmentsForValues(
	values []statspkg.SplitValue,
	colorByKey map[string]color.Color,
) []stackedRowSegment {
	segments := make([]stackedRowSegment, 0, len(values))
	for _, value := range values {
		segments = append(segments, stackedRowSegment{
			Value: value.Value,
			Color: colorByKey[value.Key],
		})
	}
	return segments
}

// presentSplitKeys returns the unique, sorted set of split keys that have
// non-zero values across the supplied per-item Splits slices. The legend
// next to a chart should only list series that contributed data.
func presentSplitKeys[T any](items []T, splits func(T) []statspkg.SplitValue) []string {
	seen := make(map[string]bool)
	keys := make([]string, 0)
	for _, item := range items {
		for _, value := range splits(item) {
			if value.Value <= 0 || seen[value.Key] {
				continue
			}
			seen[value.Key] = true
			keys = append(keys, value.Key)
		}
	}
	slices.Sort(keys)
	return keys
}

func histBucketSplits(b statspkg.SplitHistogramBucket) []statspkg.SplitValue { return b.Splits }
func namedStatSplits(s statspkg.SplitNamedStat) []statspkg.SplitValue        { return s.Splits }
func rateStatSplits(s statspkg.SplitRateStat) []statspkg.SplitValue          { return s.Splits }
func dailyShareSplits(s statspkg.SplitDailyShare) []statspkg.SplitValue      { return s.Splits }
