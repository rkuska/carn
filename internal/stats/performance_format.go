package stats

import "fmt"

type performanceMetricDefinition struct {
	question         string
	formula          string
	scoreWeight      float64
	higherIsBetter   bool
	visibleByDefault bool
	providerSpecific bool
}

var performanceMetricDefinitions = map[string]performanceMetricDefinition{
	perfMetricVerificationPass: {
		question:         "Are mutated sessions getting verified after changes?",
		formula:          "verified sessions / mutated sessions",
		scoreWeight:      1,
		higherIsBetter:   true,
		visibleByDefault: true,
	},
	perfMetricFirstPassResolution: {
		question:         "Are mutated sessions getting resolved without follow-up repair work or post-mutation failures?",
		formula:          "resolved mutated sessions / mutated sessions",
		scoreWeight:      1,
		higherIsBetter:   true,
		visibleByDefault: true,
	},
	perfMetricCorrectionBurden: {
		question:         "How many user follow-up turns arrive after the first change attempt?",
		formula:          "follow-up user turns after first mutation or failure / mutated sessions",
		scoreWeight:      1,
		higherIsBetter:   false,
		visibleByDefault: true,
	},
	perfMetricPatchChurn: {
		question:         "How much rewrite churn is needed per mutated session?",
		formula:          "mutation attempts + targets + hunks / mutated sessions",
		scoreWeight:      1,
		higherIsBetter:   false,
		visibleByDefault: false,
	},
	perfMetricReadBeforeWrite: {
		question:         "Does the model inspect context before it edits?",
		formula:          "(read + search) / (mutate + rewrite)",
		scoreWeight:      1,
		higherIsBetter:   true,
		visibleByDefault: true,
	},
	perfMetricResearchRatio: {
		question:         "How much investigation happens before mutation or execution?",
		formula:          "(read + search + web) / (mutate + rewrite + execute)",
		scoreWeight:      1,
		higherIsBetter:   true,
		visibleByDefault: false,
	},
	perfMetricRewriteRate: {
		question:         "How often does the model choose full rewrites instead of precise edits?",
		formula:          "rewrite / (rewrite + mutate)",
		scoreWeight:      1,
		higherIsBetter:   false,
		visibleByDefault: false,
	},
	perfMetricBlindEditRate: {
		question:         "How often does the model edit a target without reading it first?",
		formula:          "blind targeted mutations / targeted mutations",
		scoreWeight:      1,
		higherIsBetter:   false,
		visibleByDefault: true,
	},
	perfMetricReasoningLoopRate: {
		question:         "Is the model getting stuck in repeated same-target retries?",
		formula:          "same-action same-target loops / actions",
		scoreWeight:      1,
		higherIsBetter:   false,
		visibleByDefault: true,
	},
	perfMetricTokensPerTurn: {
		question:         "How much token spend is needed per user turn?",
		formula:          "provider total tokens / user turns",
		scoreWeight:      1,
		higherIsBetter:   false,
		visibleByDefault: true,
	},
	perfMetricActionsPerTurn: {
		question:         "How many actions are needed per user turn?",
		formula:          "normalized actions / user turns",
		scoreWeight:      1,
		higherIsBetter:   false,
		visibleByDefault: true,
	},
	perfMetricTimeToMutation: {
		question:         "How much setup work happens before the first mutation?",
		formula:          "actions before first mutation / mutated sessions",
		scoreWeight:      0,
		higherIsBetter:   false,
		visibleByDefault: true,
	},
	perfMetricVisibleThinking: {
		question:         "How much visible reasoning appears per assistant turn?",
		formula:          "visible reasoning characters / assistant turns",
		scoreWeight:      0.25,
		higherIsBetter:   true,
		visibleByDefault: false,
		providerSpecific: true,
	},
	perfMetricReasoningShare: {
		question:         "How much total spend goes to reasoning tokens?",
		formula:          "reasoning output tokens / provider total tokens",
		scoreWeight:      0.5,
		higherIsBetter:   false,
		visibleByDefault: false,
		providerSpecific: true,
	},
	perfMetricErrorRate: {
		question:         "Are tool calls failing less often than before?",
		formula:          "errored action results / action calls",
		scoreWeight:      1,
		higherIsBetter:   false,
		visibleByDefault: true,
	},
	perfMetricRejectionRate: {
		question:         "Are users rejecting suggested tools less often?",
		formula:          "rejected action results / action calls",
		scoreWeight:      1,
		higherIsBetter:   false,
		visibleByDefault: false,
	},
	perfMetricAbortRate: {
		question:         "Are started turns reaching completion more reliably?",
		formula:          "aborted turns / started turns",
		scoreWeight:      1,
		higherIsBetter:   false,
		visibleByDefault: false,
		providerSpecific: true,
	},
	perfMetricContextPressure: {
		question:         "Is context pressure affecting fewer sessions?",
		formula:          "sessions with compaction or context pressure / sessions",
		scoreWeight:      1,
		higherIsBetter:   false,
		visibleByDefault: true,
	},
	perfMetricRetryBurden: {
		question:         "Are retries and API errors staying under control?",
		formula:          "(retry attempts + API errors) / sessions",
		scoreWeight:      1,
		higherIsBetter:   false,
		visibleByDefault: true,
		providerSpecific: true,
	},
}

func enrichPerformanceMetrics(metrics []PerformanceMetric) []PerformanceMetric {
	if len(metrics) == 0 {
		return nil
	}

	enriched := make([]PerformanceMetric, len(metrics))
	for i, metric := range metrics {
		enriched[i] = enrichPerformanceMetric(metric)
	}
	return enriched
}

func enrichPerformanceMetric(metric PerformanceMetric) PerformanceMetric {
	def, ok := performanceMetricDefinitions[metric.ID]
	if ok {
		metric.Question = def.question
		metric.Formula = def.formula
		metric.ScoreWeight = def.scoreWeight
		metric.HigherIsBetter = def.higherIsBetter
		metric.VisibleByDefault = def.visibleByDefault
		metric.ProviderSpecific = def.providerSpecific
	}
	if metric.HasBaseline {
		metric.DeltaText = formatPerformanceDelta(metric.ID, metric.Current-metric.Baseline)
	}
	metric.Status = classifyPerformanceMetric(metric)
	return metric
}

func classifyPerformanceMetric(metric PerformanceMetric) PerformanceMetricStatus {
	if !metric.HasScore {
		return PerformanceMetricStatusLowSample
	}
	switch metric.Trend {
	case TrendDirectionNone:
		return PerformanceMetricStatusNone
	case TrendDirectionUp:
		return PerformanceMetricStatusBetter
	case TrendDirectionDown:
		return PerformanceMetricStatusWorse
	case TrendDirectionFlat:
		return PerformanceMetricStatusFlat
	default:
		return PerformanceMetricStatusNone
	}
}

func formatPerformanceDelta(id string, delta float64) string {
	switch id {
	case perfMetricVerificationPass,
		perfMetricFirstPassResolution,
		perfMetricRewriteRate,
		perfMetricBlindEditRate,
		perfMetricReasoningShare,
		perfMetricErrorRate,
		perfMetricRejectionRate,
		perfMetricAbortRate,
		perfMetricContextPressure,
		perfMetricRetryBurden:
		return fmt.Sprintf("%+.1f pts", delta*100)
	default:
		return fmt.Sprintf("%+.1f", delta)
	}
}
