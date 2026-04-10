package stats

func applyPerformanceSequence(
	performance Performance,
	timeRange TimeRange,
	sequence []PerformanceSequenceSession,
) Performance {
	window := performanceTimeWindow(timeRange, sequence)
	current := aggregatePerformanceSequenceInRange(sequence, window.current)
	baseline := aggregatePerformanceSequenceInRange(sequence, window.baseline)
	context := newPerformanceMetricContext(
		current.sessionCount,
		baseline.sessionCount,
		buildPerformanceSequenceBucketAggregates(sequence, window.current),
	)

	outcomeMetrics := append([]PerformanceMetric(nil), performance.Outcome.Metrics...)
	outcomeMetrics = append(outcomeMetrics,
		performanceMetricFromCounts(
			perfMetricFirstPassResolution,
			"first-pass resolution",
			current.resolvedSessions,
			current.mutatedSessions,
			baseline.resolvedSessions,
			baseline.mutatedSessions,
			true,
			performanceMinSequenceSamples,
			0.05,
			"Mutated sessions without follow-up correction signals or post-mutation tool failures.",
			context,
			func(agg performanceSequenceAggregate) (int, int) {
				return agg.resolvedSessions, agg.mutatedSessions
			},
		),
		performanceMetricFromRatio(
			perfMetricCorrectionBurden,
			"correction burden",
			float64(current.correctionFollowups),
			float64(current.mutatedSessions),
			float64(baseline.correctionFollowups),
			float64(baseline.mutatedSessions),
			false,
			performanceMinSequenceSamples,
			0.25,
			"User follow-up turns after the first mutation or failure, until successful verification.",
			context,
			func(agg performanceSequenceAggregate) (float64, float64) {
				return float64(agg.correctionFollowups), float64(agg.mutatedSessions)
			},
		),
		performanceMetricFromRatio(
			perfMetricPatchChurn,
			"patch churn",
			float64(current.patchChurn),
			float64(current.mutatedSessions),
			float64(baseline.patchChurn),
			float64(baseline.mutatedSessions),
			false,
			performanceMinSequenceSamples,
			0.5,
			"Mutation attempts, targets, and hunks per mutated session.",
			context,
			func(agg performanceSequenceAggregate) (float64, float64) {
				return float64(agg.patchChurn), float64(agg.mutatedSessions)
			},
		),
	)
	performance.Outcome = buildPerformanceLane(performance.Outcome.Label, performance.Outcome.Detail, outcomeMetrics)

	disciplineMetrics := append([]PerformanceMetric(nil), performance.Discipline.Metrics...)
	disciplineMetrics = append(disciplineMetrics,
		performanceMetricFromCounts(
			perfMetricBlindEditRate,
			"blind edit rate",
			current.blindMutations,
			current.targetedMutations,
			baseline.blindMutations,
			baseline.targetedMutations,
			false,
			performanceMinSequenceSamples,
			0.05,
			"Targeted mutations without a prior read of the same file or a recent search.",
			context,
			func(agg performanceSequenceAggregate) (int, int) {
				return agg.blindMutations, agg.targetedMutations
			},
		),
		performanceMetricFromRatio(
			perfMetricReasoningLoopRate,
			"reasoning loop rate",
			float64(current.loopCount),
			float64(current.actionCount),
			float64(baseline.loopCount),
			float64(baseline.actionCount),
			false,
			performanceMinSequenceSamples,
			0.1,
			"Repeated same-action same-target patterns per action.",
			context,
			func(agg performanceSequenceAggregate) (float64, float64) {
				return float64(agg.loopCount), float64(agg.actionCount)
			},
		),
	)
	performance.Discipline = buildPerformanceLane(
		performance.Discipline.Label,
		performance.Discipline.Detail,
		disciplineMetrics,
	)

	efficiencyMetrics := append([]PerformanceMetric(nil), performance.Efficiency.Metrics...)
	efficiencyMetrics = append(efficiencyMetrics,
		performanceMetricFromRatio(
			perfMetricTimeToMutation,
			"actions to mutation",
			float64(current.actionsBeforeMutation),
			float64(current.mutatedSessions),
			float64(baseline.actionsBeforeMutation),
			float64(baseline.mutatedSessions),
			false,
			performanceMinSequenceSamples,
			0.5,
			"Average normalized actions before the first mutation.",
			context,
			func(agg performanceSequenceAggregate) (float64, float64) {
				return float64(agg.actionsBeforeMutation), float64(agg.mutatedSessions)
			},
		),
		performanceMetricFromRatio(
			perfMetricVisibleThinking,
			"visible thinking",
			float64(current.visibleReasoning),
			float64(current.assistantTurns),
			float64(baseline.visibleReasoning),
			float64(baseline.assistantTurns),
			true,
			performanceMinSequenceSamples,
			25,
			"Visible reasoning characters per assistant turn.",
			context,
			func(agg performanceSequenceAggregate) (float64, float64) {
				return float64(agg.visibleReasoning), float64(agg.assistantTurns)
			},
		),
	)
	performance.Efficiency = buildPerformanceLane(
		performance.Efficiency.Label,
		performance.Efficiency.Detail,
		efficiencyMetrics,
	)

	performance.Diagnostics = append(performance.Diagnostics, performanceDiagnosticFromRatio(
		"hidden thinking turns",
		float64(current.hiddenThinking),
		float64(current.assistantTurns),
		float64(baseline.hiddenThinking),
		float64(baseline.assistantTurns),
		false,
		performanceMinSequenceSamples,
		"Assistant turns with hidden reasoning and no visible reasoning text.",
		context,
		func(agg performanceSequenceAggregate) (float64, float64) {
			return float64(agg.hiddenThinking), float64(agg.assistantTurns)
		},
	))
	performance.Scope.SequenceLoaded = true
	performance.Scope.SequenceSampleCount = current.sessionCount
	return performance
}
