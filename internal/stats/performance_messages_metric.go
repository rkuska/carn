package stats

func applyPerformanceSequence(
	performance Performance,
	timeRange TimeRange,
	sequence []PerformanceSequenceSession,
) Performance {
	window := performanceTimeWindow(timeRange, sequence)
	current := aggregatePerformanceSequenceInRange(sequence, window.current)
	baseline := aggregatePerformanceSequenceInRange(sequence, window.baseline)
	context := newPerformanceSequenceMetricContext(
		window.current,
		current.sessionCount,
		baseline.sessionCount,
		sequence,
	)

	outcomeMetrics := append([]PerformanceMetric(nil), performance.Outcome.Metrics...)
	outcomeMetrics = append(outcomeMetrics,
		performanceSequenceRateMetric(
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
			func(agg performanceSequenceAggregate) (float64, float64) {
				return float64(agg.resolvedSessions), float64(agg.mutatedSessions)
			},
		),
		performanceSequenceAverageMetric(
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
		performanceSequenceAverageMetric(
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
		performanceSequenceRateMetric(
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
			func(agg performanceSequenceAggregate) (float64, float64) {
				return float64(agg.blindMutations), float64(agg.targetedMutations)
			},
		),
		performanceSequenceAverageMetric(
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
		performanceSequenceAverageMetric(
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
		performanceSequenceAverageMetric(
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

	performance.Diagnostics = append(performance.Diagnostics, performanceSequenceDiagnostic(
		"hidden thinking turns",
		float64(current.hiddenThinking),
		float64(current.assistantTurns),
		float64(baseline.hiddenThinking),
		float64(baseline.assistantTurns),
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

func performanceSequenceRateMetric(
	id, label string,
	currentNumerator, currentDenominator int,
	baselineNumerator, baselineDenominator int,
	higherIsBetter bool,
	minSamples int,
	floor float64,
	detail string,
	context performanceSequenceMetricContext,
	extract func(performanceSequenceAggregate) (float64, float64),
) PerformanceMetric {
	return performanceSequenceAverageMetric(
		id,
		label,
		float64(currentNumerator),
		float64(currentDenominator),
		float64(baselineNumerator),
		float64(baselineDenominator),
		higherIsBetter,
		minSamples,
		floor,
		detail,
		context,
		extract,
	)
}

func performanceSequenceAverageMetric(
	id, label string,
	currentNumerator, currentDenominator float64,
	baselineNumerator, baselineDenominator float64,
	higherIsBetter bool,
	minSamples int,
	floor float64,
	detail string,
	context performanceSequenceMetricContext,
	extract func(performanceSequenceAggregate) (float64, float64),
) PerformanceMetric {
	currentValue := safeRatio(currentNumerator, currentDenominator)
	baselineValue := safeRatio(baselineNumerator, baselineDenominator)
	currentSamples := performanceRelevantSampleCount(currentDenominator, context.currentSampleCount)
	baselineSamples := performanceRelevantSampleCount(baselineDenominator, context.baselineSampleCount)
	score := scorePerformanceMetric(
		currentValue,
		baselineValue,
		higherIsBetter,
		currentSamples,
		baselineSamples,
		minSamples,
		floor,
	)
	return PerformanceMetric{
		ID:          id,
		Label:       label,
		Value:       formatPerformanceValue(id, currentValue),
		Detail:      detail,
		Current:     currentValue,
		Baseline:    baselineValue,
		HasBaseline: score.hasBaseline,
		Score:       score.score,
		ScoreWeight: 1,
		HasScore:    score.hasScore,
		Trend:       score.trend,
		SampleCount: currentSamples,
		Series:      performanceSequenceSeries(context, extract),
	}
}

func performanceSequenceDiagnostic(
	label string,
	currentNumerator, currentDenominator float64,
	baselineNumerator, baselineDenominator float64,
	detail string,
	context performanceSequenceMetricContext,
	extract func(performanceSequenceAggregate) (float64, float64),
) PerformanceDiagnostic {
	currentValue := safeRatio(currentNumerator, currentDenominator)
	baselineValue := safeRatio(baselineNumerator, baselineDenominator)
	score := scorePerformanceMetric(
		currentValue,
		baselineValue,
		false,
		performanceRelevantSampleCount(currentDenominator, context.currentSampleCount),
		performanceRelevantSampleCount(baselineDenominator, context.baselineSampleCount),
		performanceMinSequenceSamples,
		0.05,
	)
	return PerformanceDiagnostic{
		Group:       "provider_signals",
		Label:       label,
		Value:       formatPerformanceRatio(currentValue),
		Detail:      detail,
		Current:     currentValue,
		Baseline:    baselineValue,
		HasBaseline: score.hasBaseline,
		Trend:       score.trend,
		Series:      performanceSequenceSeries(context, extract),
	}
}

func performanceSequenceSeries(
	context performanceSequenceMetricContext,
	extract func(performanceSequenceAggregate) (float64, float64),
) []PerformancePoint {
	if len(context.bucketAggregates) == 0 {
		return nil
	}

	points := make([]PerformancePoint, 0, len(context.bucketAggregates))
	for _, bucket := range context.bucketAggregates {
		numerator, denominator := extract(bucket.aggregate)
		points = append(points, PerformancePoint{
			Timestamp:   bucket.bucket.start,
			Value:       safeRatio(numerator, denominator),
			SampleCount: performanceRelevantSampleCount(denominator, bucket.aggregate.sessionCount),
		})
	}
	return points
}
