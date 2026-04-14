package stats

import el "github.com/rkuska/carn/internal/app/elements"

type histogramLayout struct {
	bucketWidths []int
	gapWidth     int
	graphWidth   int
}

func resolveHistogramLayout(graphWidth, bucketCount int) histogramLayout {
	layout := el.ResolveHistogramLayout(graphWidth, bucketCount)
	return histogramLayout{
		bucketWidths: layout.BucketWidths,
		gapWidth:     layout.GapWidth,
		graphWidth:   layout.GraphWidth,
	}
}

type uniformTurnBarLayout struct {
	barWidth int
	gapWidth int
	leftPad  int
}

func resolveUniformTurnBarLayout(graphWidth, barCount int) (uniformTurnBarLayout, bool) {
	layout, ok := el.ResolveUniformTurnBarLayout(graphWidth, barCount)
	if !ok {
		return uniformTurnBarLayout{}, false
	}
	return uniformTurnBarLayout{
		barWidth: layout.BarWidth,
		gapWidth: layout.GapWidth,
		leftPad:  layout.LeftPad,
	}, true
}

var resolveFloatSegmentHeights = el.ResolveFloatSegmentHeights
