package stats

import conv "github.com/rkuska/carn/internal/conversation"

const (
	perfMetricVerificationPass = "verification_pass_rate"
	perfMetricReadBeforeWrite  = "read_before_write_ratio"
	perfMetricResearchRatio    = "research_to_mutation_ratio"
	perfMetricRewriteRate      = "full_rewrite_rate"
	perfMetricTokensPerTurn    = "tokens_per_user_turn"
	perfMetricActionsPerTurn   = "actions_per_user_turn"
	perfMetricReasoningShare   = "reasoning_token_share"
	perfMetricErrorRate        = "tool_error_rate"
	perfMetricRejectionRate    = "tool_rejection_rate"
	perfMetricAbortRate        = "abort_rate"
	perfMetricContextPressure  = "context_pressure_rate"
	perfMetricRetryBurden      = "retry_burden"
)

func ComputePerformance(
	conversations []conv.Conversation,
	timeRange TimeRange,
	sequence []PerformanceSequenceSession,
) Performance {
	sessions := flattenPerformanceSessions(conversations)
	window := performanceTimeWindow(timeRange, sessions)
	current := aggregatePerformanceSessionsInRange(sessions, window.current)
	baseline := aggregatePerformanceSessionsInRange(sessions, window.baseline)
	context := newPerformanceMetricContext(window.current, current.sessionCount, baseline.sessionCount, sessions)
	scope := buildPerformanceScope(sessions, window.current, window.baseline, baseline.sessionCount, len(sequence))
	provider := singleProviderInRange(sessions, window.current)

	performance := Performance{
		Scope:      scope,
		Outcome:    buildOutcomeLane(current, baseline, context),
		Discipline: buildDisciplineLane(current, baseline, context),
		Efficiency: buildEfficiencyLane(current, baseline, context, provider),
		Robustness: buildRobustnessLane(current, baseline, context, provider),
		Diagnostics: buildPerformanceDiagnostics(
			current,
			baseline,
			context,
			provider,
		),
	}
	if len(sequence) > 0 {
		performance = applyPerformanceSequence(performance, timeRange, sequence)
	}
	performance.Overall = combinePerformanceScores(allPerformanceMetrics(performance))
	return performance
}
