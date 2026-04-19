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

var resolveFloatSegmentHeights = el.ResolveFloatSegmentHeights
