package stats

type performanceBucketAggregate[T any] struct {
	bucket    performanceBucket
	aggregate T
}

type performanceMetricContext[T any] struct {
	currentSampleCount  int
	baselineSampleCount int
	bucketAggregates    []performanceBucketAggregate[T]
}

func newPerformanceMetricContext[T any](
	currentSampleCount, baselineSampleCount int,
	bucketAggregates []performanceBucketAggregate[T],
) performanceMetricContext[T] {
	return performanceMetricContext[T]{
		currentSampleCount:  currentSampleCount,
		baselineSampleCount: baselineSampleCount,
		bucketAggregates:    bucketAggregates,
	}
}
