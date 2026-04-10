package stats

import conv "github.com/rkuska/carn/internal/conversation"

func buildOutcomeLane(
	current, baseline performanceAggregate,
	context performanceMetricContext,
) PerformanceLane {
	metric := performanceMetricFromCounts(
		perfMetricVerificationPass,
		"verification pass",
		current.verifiedSessions,
		current.mutatedSessions,
		baseline.verifiedSessions,
		baseline.mutatedSessions,
		true,
		performanceMinSessionSamples,
		0.05,
		"Mutated sessions followed by successful test or build steps.",
		context,
		func(agg performanceAggregate) (int, int) { return agg.verifiedSessions, agg.mutatedSessions },
	)
	return buildPerformanceLane(
		"Outcome",
		"Tracks whether changes stick and get verified.",
		[]PerformanceMetric{metric},
	)
}

func buildDisciplineLane(
	current, baseline performanceAggregate,
	context performanceMetricContext,
) PerformanceLane {
	metrics := []PerformanceMetric{
		performanceMetricFromRatio(
			perfMetricReadBeforeWrite,
			"read before write",
			float64(current.readCount+current.searchCount),
			float64(current.mutateCount+current.rewriteCount),
			float64(baseline.readCount+baseline.searchCount),
			float64(baseline.mutateCount+baseline.rewriteCount),
			true,
			performanceMinSessionSamples,
			0.25,
			"(read + search) / (mutate + rewrite)",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.readCount + agg.searchCount), float64(agg.mutateCount + agg.rewriteCount)
			},
		),
		performanceMetricFromRatio(
			perfMetricResearchRatio,
			"research ratio",
			float64(current.readCount+current.searchCount+current.webCount),
			float64(current.mutateCount+current.rewriteCount+current.executeCount),
			float64(baseline.readCount+baseline.searchCount+baseline.webCount),
			float64(baseline.mutateCount+baseline.rewriteCount+baseline.executeCount),
			true,
			performanceMinSessionSamples,
			0.25,
			"(read + search + web) / (mutate + rewrite + execute)",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.readCount + agg.searchCount + agg.webCount),
					float64(agg.mutateCount + agg.rewriteCount + agg.executeCount)
			},
		),
		performanceMetricFromRatio(
			perfMetricRewriteRate,
			"full rewrite rate",
			float64(current.rewriteCount),
			float64(current.rewriteCount+current.mutateCount),
			float64(baseline.rewriteCount),
			float64(baseline.rewriteCount+baseline.mutateCount),
			false,
			performanceMinSessionSamples,
			0.05,
			"rewrite / (rewrite + mutate)",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.rewriteCount), float64(agg.rewriteCount + agg.mutateCount)
			},
		),
	}
	return buildPerformanceLane(
		"Discipline",
		"Tracks inspection before mutation and rewrite churn.",
		metrics,
	)
}

func buildEfficiencyLane(
	current, baseline performanceAggregate,
	context performanceMetricContext,
	provider conv.Provider,
) PerformanceLane {
	metrics := []PerformanceMetric{
		performanceMetricFromRatio(
			perfMetricTokensPerTurn,
			"tokens / user turn",
			float64(current.totalTokens),
			float64(current.userTurns),
			float64(baseline.totalTokens),
			float64(baseline.userTurns),
			false,
			performanceMinSessionSamples,
			50,
			"Total tokens per user message.",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.totalTokens), float64(agg.userTurns)
			},
		),
		performanceMetricFromRatio(
			perfMetricActionsPerTurn,
			"actions / user turn",
			float64(current.totalActions),
			float64(current.userTurns),
			float64(baseline.totalActions),
			float64(baseline.userTurns),
			false,
			performanceMinSessionSamples,
			0.25,
			"Normalized actions per user message.",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.totalActions), float64(agg.userTurns)
			},
		),
	}
	if provider == conv.ProviderCodex {
		metrics = append(metrics, performanceMetricFromRatio(
			perfMetricReasoningShare,
			"reasoning token share",
			float64(current.reasoningOutputTokens),
			float64(current.totalTokens),
			float64(baseline.reasoningOutputTokens),
			float64(baseline.totalTokens),
			false,
			performanceMinSessionSamples,
			0.05,
			"Reasoning output tokens / total tokens.",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.reasoningOutputTokens), float64(agg.totalTokens)
			},
		))
	}
	return buildPerformanceLane(
		"Efficiency",
		"Tracks token and action cost per unit of user direction.",
		metrics,
	)
}

func buildRobustnessLane(
	current, baseline performanceAggregate,
	context performanceMetricContext,
	provider conv.Provider,
) PerformanceLane {
	metrics := []PerformanceMetric{
		performanceMetricFromRatio(
			perfMetricErrorRate,
			"tool error rate",
			float64(current.actionErrorCount),
			float64(current.totalActions),
			float64(baseline.actionErrorCount),
			float64(baseline.totalActions),
			false,
			performanceMinSessionSamples,
			0.05,
			"Errored action results / action calls.",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.actionErrorCount), float64(agg.totalActions)
			},
		),
		performanceMetricFromRatio(
			perfMetricRejectionRate,
			"tool rejection rate",
			float64(current.actionRejectCount),
			float64(current.totalActions),
			float64(baseline.actionRejectCount),
			float64(baseline.totalActions),
			false,
			performanceMinSessionSamples,
			0.05,
			"Rejected action results / action calls.",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.actionRejectCount), float64(agg.totalActions)
			},
		),
		performanceMetricFromCounts(
			perfMetricContextPressure,
			"context pressure",
			current.contextPressureCount,
			current.sessionCount,
			baseline.contextPressureCount,
			baseline.sessionCount,
			false,
			performanceMinSessionSamples,
			0.05,
			"Sessions with compaction or context pressure signals.",
			context,
			func(agg performanceAggregate) (int, int) { return agg.contextPressureCount, agg.sessionCount },
		),
	}
	if provider == conv.ProviderCodex {
		metrics = append(metrics, performanceMetricFromRatio(
			perfMetricAbortRate,
			"abort rate",
			float64(current.abortCount),
			float64(current.startedTurnCount),
			float64(baseline.abortCount),
			float64(baseline.startedTurnCount),
			false,
			performanceMinSessionSamples,
			0.05,
			"Aborted turns / started turns.",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.abortCount), float64(agg.startedTurnCount)
			},
		))
	}
	if provider == conv.ProviderClaude {
		metrics = append(metrics, performanceMetricFromRatio(
			perfMetricRetryBurden,
			"retry burden",
			float64(current.retryAttemptCount+current.apiErrorCount),
			float64(current.sessionCount),
			float64(baseline.retryAttemptCount+baseline.apiErrorCount),
			float64(baseline.sessionCount),
			false,
			performanceMinSessionSamples,
			0.25,
			"Retries and API errors per session.",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.retryAttemptCount + agg.apiErrorCount), float64(agg.sessionCount)
			},
		))
	}
	return buildPerformanceLane(
		"Robustness",
		"Tracks failures, rejections, aborts, retries, and context pressure.",
		metrics,
	)
}

func buildPerformanceDiagnostics(
	current, baseline performanceAggregate,
	context performanceMetricContext,
	provider conv.Provider,
) []PerformanceDiagnostic {
	diagnostics := []PerformanceDiagnostic{
		performanceDiagnosticFromRatio(
			"hidden thinking",
			float64(current.hiddenThinkingCount),
			float64(current.reasoningBlockCount),
			float64(baseline.hiddenThinkingCount),
			float64(baseline.reasoningBlockCount),
			"Redacted reasoning blocks / reasoning blocks.",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.hiddenThinkingCount), float64(agg.reasoningBlockCount)
			},
		),
		performanceDiagnosticFromRatio(
			"cache efficiency",
			float64(current.cacheReadTokens),
			float64(current.cachePromptTokens),
			float64(baseline.cacheReadTokens),
			float64(baseline.cachePromptTokens),
			"Cache-read tokens / prompt-side tokens.",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.cacheReadTokens), float64(agg.cachePromptTokens)
			},
		),
		performanceDiagnosticFromRatio(
			"output / input",
			float64(current.outputTokens),
			float64(current.inputTokens),
			float64(baseline.outputTokens),
			float64(baseline.inputTokens),
			"Output tokens / input tokens.",
			context,
			func(agg performanceAggregate) (float64, float64) {
				return float64(agg.outputTokens), float64(agg.inputTokens)
			},
		),
	}
	if provider == conv.ProviderClaude {
		diagnostics = append(diagnostics,
			performanceAverageDiagnostic(
				"thinking budget",
				current.maxThinkingTokens,
				current.maxThinkingSamples,
				baseline.maxThinkingTokens,
				baseline.maxThinkingSamples,
				"Average requested maxThinkingTokens.",
			),
			performanceTopCountDiagnostic(
				"stop reason",
				current.stopReasonCounts,
				baseline.stopReasonCounts,
				"Most common Claude stop_reason in the slice.",
			),
			performanceTopCountDiagnostic(
				"server tool use",
				current.serverToolUseCounts,
				baseline.serverToolUseCounts,
				"Top Claude server-side tool-use counter.",
			),
		)
	}
	if provider == conv.ProviderCodex {
		diagnostics = append(diagnostics, performanceTopCountDiagnostic(
			"effort mode",
			current.effortCounts,
			baseline.effortCounts,
			"Most common Codex effort mode in the slice.",
		))
	}
	return diagnostics
}

func buildPerformanceLane(label, detail string, metrics []PerformanceMetric) PerformanceLane {
	metrics = enrichPerformanceMetrics(metrics)
	score := combinePerformanceScores(metrics)
	return PerformanceLane{
		Label:    label,
		Detail:   detail,
		Score:    score.Score,
		HasScore: score.HasScore,
		Trend:    score.Trend,
		Metrics:  metrics,
	}
}
