package stats

type performanceSequenceAggregate struct {
	sessionCount          int
	mutatedSessions       int
	resolvedSessions      int
	correctionFollowups   int
	patchChurn            int
	blindMutations        int
	targetedMutations     int
	loopCount             int
	actionCount           int
	actionsBeforeMutation int
	visibleReasoning      int
	assistantTurns        int
	hiddenThinking        int
}

type performanceSequenceBucketAggregate struct {
	bucket    performanceBucket
	aggregate performanceSequenceAggregate
}

type performanceSequenceMetricContext struct {
	currentSampleCount  int
	baselineSampleCount int
	bucketAggregates    []performanceSequenceBucketAggregate
}

func newPerformanceSequenceMetricContext(
	timeRange TimeRange,
	currentSampleCount, baselineSampleCount int,
	sessions []PerformanceSequenceSession,
) performanceSequenceMetricContext {
	return performanceSequenceMetricContext{
		currentSampleCount:  currentSampleCount,
		baselineSampleCount: baselineSampleCount,
		bucketAggregates:    buildPerformanceSequenceBucketAggregates(sessions, timeRange),
	}
}

func aggregatePerformanceSequenceInRange(
	sessions []PerformanceSequenceSession,
	timeRange TimeRange,
) performanceSequenceAggregate {
	var agg performanceSequenceAggregate
	for _, session := range sessions {
		if !timeRangeContains(timeRange, session.Timestamp) {
			continue
		}
		addPerformanceSequenceSession(&agg, session)
	}
	return agg
}

func addPerformanceSequenceSession(
	agg *performanceSequenceAggregate,
	session PerformanceSequenceSession,
) {
	agg.sessionCount++
	if session.Mutated {
		agg.mutatedSessions++
		if session.FirstPassResolved {
			agg.resolvedSessions++
		}
		agg.correctionFollowups += session.CorrectionFollowups
		agg.patchChurn += session.MutationCount +
			session.DistinctMutationTargets +
			session.PatchHunkCount
		agg.actionsBeforeMutation += session.ActionsBeforeFirstMutation
	}
	agg.blindMutations += session.BlindMutationCount
	agg.targetedMutations += session.TargetedMutationCount
	agg.loopCount += session.ReasoningLoopCount
	agg.actionCount += session.ActionCount
	agg.visibleReasoning += session.VisibleReasoningChars
	agg.assistantTurns += session.AssistantTurns
	agg.hiddenThinking += session.HiddenThinkingTurns
}

func buildPerformanceSequenceBucketAggregates(
	sessions []PerformanceSequenceSession,
	timeRange TimeRange,
) []performanceSequenceBucketAggregate {
	buckets := performanceBuckets(timeRange)
	if len(buckets) == 0 {
		return nil
	}

	aggregates := make([]performanceSequenceBucketAggregate, len(buckets))
	for i, bucket := range buckets {
		aggregates[i].bucket = bucket
	}
	step := performanceBucketStep(buckets)
	for _, session := range sessions {
		if !timeRangeContains(timeRange, session.Timestamp) {
			continue
		}
		index := performanceBucketIndex(session.Timestamp, buckets[0].start, step, len(buckets))
		addPerformanceSequenceSession(&aggregates[index].aggregate, session)
	}
	return aggregates
}
