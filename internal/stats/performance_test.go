package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestComputePerformanceBuildsSessionLevelLaneScores(t *testing.T) {
	t.Parallel()

	timeRange := TimeRange{
		Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	}
	conversations := append(
		makePerformanceConversations(
			conv.ProviderClaude,
			"baseline",
			time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 2
				meta.TotalUsage = conv.TokenUsage{InputTokens: 400, OutputTokens: 120}
				meta.ActionCounts = map[string]int{
					string(conv.NormalizedActionRead):    1,
					string(conv.NormalizedActionMutate):  2,
					string(conv.NormalizedActionRewrite): 1,
					string(conv.NormalizedActionExecute): 2,
				}
				meta.ActionErrorCounts = map[string]int{
					string(conv.NormalizedActionMutate):  1,
					string(conv.NormalizedActionExecute): 1,
				}
				meta.Performance.AbortCount = 1
				meta.Performance.TaskStartedCount = 2
				meta.Performance.CompactionCount = 1
				meta.Performance.RetryAttemptCount = 2
				meta.Performance.APIErrorCounts = map[string]int{"api_error": 1}
			},
		),
		makePerformanceConversations(
			conv.ProviderClaude,
			"current",
			time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 3
				meta.TotalUsage = conv.TokenUsage{InputTokens: 220, OutputTokens: 80}
				meta.ActionCounts = map[string]int{
					string(conv.NormalizedActionRead):   3,
					string(conv.NormalizedActionSearch): 1,
					string(conv.NormalizedActionMutate): 1,
					string(conv.NormalizedActionTest):   1,
				}
				meta.Performance.StopReasonCounts = map[string]int{"end_turn": 1}
			},
		)...,
	)

	got := ComputePerformance(conversations, timeRange, nil)

	assert.Equal(t, 12, got.Scope.SessionCount)
	assert.Equal(t, []string{"Claude"}, got.Scope.Providers)
	assert.Equal(t, []string{"claude-sonnet-4"}, got.Scope.Models)
	assert.True(t, got.Scope.SingleFamily)
	assert.Equal(t, "Claude", got.Scope.PrimaryProvider)
	assert.Equal(t, "claude-sonnet-4", got.Scope.PrimaryModel)
	assert.True(t, got.Overall.HasScore)

	verification := findPerformanceMetric(t, got.Outcome.Metrics, perfMetricVerificationPass)
	assert.InDelta(t, 1.0, verification.Current, 0.0001)
	assert.Equal(t, TrendDirectionUp, verification.Trend)
	assert.NotEmpty(t, verification.Question)
	assert.NotEmpty(t, verification.Formula)
	assert.Equal(t, PerformanceMetricStatusBetter, verification.Status)
	assert.NotEmpty(t, verification.DeltaText)

	readBeforeWrite := findPerformanceMetric(t, got.Discipline.Metrics, perfMetricReadBeforeWrite)
	assert.InDelta(t, 4.0, readBeforeWrite.Current, 0.0001)
	assert.Equal(t, TrendDirectionUp, readBeforeWrite.Trend)

	tokensPerTurn := findPerformanceMetric(t, got.Efficiency.Metrics, perfMetricTokensPerTurn)
	assert.InDelta(t, 100.0, tokensPerTurn.Current, 0.0001)
	assert.Equal(t, TrendDirectionUp, tokensPerTurn.Trend)

	errorRate := findPerformanceMetric(t, got.Robustness.Metrics, perfMetricErrorRate)
	assert.InDelta(t, 0.0, errorRate.Current, 0.0001)
	assert.Equal(t, TrendDirectionUp, errorRate.Trend)

	require.NotEmpty(t, got.Diagnostics)
	assert.Equal(t, "stop reason", findPerformanceDiagnostic(t, got.Diagnostics, "stop reason").Label)
}

func TestComputePerformanceMarksMixedProviderAndModelScopeAsNonComparable(t *testing.T) {
	t.Parallel()

	timeRange := TimeRange{
		Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	}
	conversations := []conv.Conversation{
		testPerformanceConversation(
			conv.ProviderClaude,
			"claude-current",
			testPerformanceSessionMeta(
				"claude-current",
				"alpha",
				time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
				func(meta *conv.SessionMeta) {
					meta.Model = "claude-opus-4-1"
				},
			),
		),
		testPerformanceConversation(
			conv.ProviderCodex,
			"codex-current",
			testPerformanceSessionMeta(
				"codex-current",
				"beta",
				time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
				func(meta *conv.SessionMeta) {
					meta.Model = "gpt-5.4"
				},
			),
		),
	}

	got := ComputePerformance(conversations, timeRange, nil)

	assert.False(t, got.Scope.SingleProvider)
	assert.False(t, got.Scope.SingleModel)
	assert.False(t, got.Scope.SingleFamily)
	assert.Empty(t, got.Scope.PrimaryProvider)
	assert.Empty(t, got.Scope.PrimaryModel)
	assert.Equal(t, []string{"Claude", "Codex"}, got.Scope.Providers)
	assert.Equal(t, []string{"claude-opus-4-1", "gpt-5.4"}, got.Scope.Models)
}

func TestComputePerformanceAddsCodexReasoningShareOnlyForSingleProviderScope(t *testing.T) {
	t.Parallel()

	timeRange := TimeRange{
		Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	}

	codexOnly := append(
		makePerformanceConversations(
			conv.ProviderCodex,
			"baseline-codex",
			time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 2
				meta.TotalUsage = conv.TokenUsage{InputTokens: 300, OutputTokens: 120, ReasoningOutputTokens: 90}
			},
		),
		makePerformanceConversations(
			conv.ProviderCodex,
			"current-codex",
			time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 2
				meta.TotalUsage = conv.TokenUsage{InputTokens: 300, OutputTokens: 120, ReasoningOutputTokens: 30}
			},
		)...,
	)
	got := ComputePerformance(codexOnly, timeRange, nil)
	assert.NotNil(t, tryFindPerformanceMetric(got.Efficiency.Metrics, perfMetricReasoningShare))

	mixed := append([]conv.Conversation{}, codexOnly...)
	mixed = append(
		mixed,
		makePerformanceConversations(
			conv.ProviderClaude,
			"current-claude",
			time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
			2,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 2
			},
		)...,
	)
	got = ComputePerformance(mixed, timeRange, nil)
	assert.Nil(t, tryFindPerformanceMetric(got.Efficiency.Metrics, perfMetricReasoningShare))
}

func TestComputePerformanceMergesSequenceMetrics(t *testing.T) {
	t.Parallel()

	timeRange := TimeRange{
		Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	}
	conversations := append(
		makePerformanceConversations(
			conv.ProviderClaude,
			"baseline",
			time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 2
				meta.ActionCounts = map[string]int{
					string(conv.NormalizedActionRead):   1,
					string(conv.NormalizedActionMutate): 1,
				}
			},
		),
		makePerformanceConversations(
			conv.ProviderClaude,
			"current",
			time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
			12,
			func(meta *conv.SessionMeta) {
				meta.UserMessageCount = 2
				meta.ActionCounts = map[string]int{
					string(conv.NormalizedActionRead):   1,
					string(conv.NormalizedActionMutate): 1,
				}
			},
		)...,
	)
	sequence := append(
		makeSequenceSessions(time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC), 12, true),
		makeSequenceSessions(time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC), 6, true)...,
	)
	sequence = append(sequence, makeSequenceSessions(time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC), 6, false)...)

	got := ComputePerformance(conversations, timeRange, sequence)

	firstPass := findPerformanceMetric(t, got.Outcome.Metrics, perfMetricFirstPassResolution)
	assert.InDelta(t, 0.5, firstPass.Current, 0.0001)
	assert.Equal(t, "50.0%", firstPass.Value)
	assert.Equal(t, TrendDirectionDown, firstPass.Trend)

	blindEdit := findPerformanceMetric(t, got.Discipline.Metrics, perfMetricBlindEditRate)
	assert.Greater(t, blindEdit.Current, 0.0)
	assert.Equal(t, "50.0%", blindEdit.Value)

	assert.True(t, got.Scope.SequenceLoaded)
	assert.Equal(t, 12, got.Scope.SequenceSampleCount)

	labels := make(map[string]bool, len(got.Diagnostics))
	for _, diagnostic := range got.Diagnostics {
		assert.False(t, labels[diagnostic.Label], "duplicate diagnostic label %q", diagnostic.Label)
		labels[diagnostic.Label] = true
	}
}

func TestComputePerformanceBuildsMetricSeriesByDay(t *testing.T) {
	t.Parallel()

	timeRange := TimeRange{
		Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 3, 23, 59, 59, 0, time.UTC),
	}
	conversations := makePerformanceConversations(
		conv.ProviderClaude,
		"series",
		time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
		3,
		func(meta *conv.SessionMeta) {
			meta.UserMessageCount = 1
			meta.ActionCounts = map[string]int{
				string(conv.NormalizedActionMutate): 1,
			}
			if meta.Timestamp.Day()%2 == 1 {
				meta.ActionCounts[string(conv.NormalizedActionTest)] = 1
			}
		},
	)

	got := ComputePerformance(conversations, timeRange, nil)
	metric := findPerformanceMetric(t, got.Outcome.Metrics, perfMetricVerificationPass)

	require.Len(t, metric.Series, 3)
	assert.Equal(t, 1, metric.Series[0].SampleCount)
	assert.InDelta(t, 1.0, metric.Series[0].Value, 0.0001)
	assert.Equal(t, 1, metric.Series[1].SampleCount)
	assert.InDelta(t, 0.0, metric.Series[1].Value, 0.0001)
	assert.Equal(t, 1, metric.Series[2].SampleCount)
	assert.InDelta(t, 1.0, metric.Series[2].Value, 0.0001)
}

func TestComputePerformanceBuildsSequenceMetricSeriesByDay(t *testing.T) {
	t.Parallel()

	timeRange := TimeRange{
		Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 3, 23, 59, 59, 0, time.UTC),
	}
	conversations := makePerformanceConversations(
		conv.ProviderClaude,
		"sequence-series",
		time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
		3,
		func(meta *conv.SessionMeta) {
			meta.UserMessageCount = 1
			meta.ActionCounts = map[string]int{
				string(conv.NormalizedActionRead):   1,
				string(conv.NormalizedActionMutate): 1,
			}
		},
	)
	sequence := []PerformanceSequenceSession{
		{
			Timestamp:             time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC),
			Mutated:               true,
			FirstPassResolved:     true,
			VerificationPassed:    true,
			TargetedMutationCount: 1,
			ActionCount:           2,
			AssistantTurns:        1,
		},
		{
			Timestamp:             time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC),
			Mutated:               true,
			TargetedMutationCount: 1,
			CorrectionFollowups:   1,
			ActionCount:           2,
			AssistantTurns:        1,
		},
		{
			Timestamp:             time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC),
			Mutated:               true,
			FirstPassResolved:     true,
			VerificationPassed:    true,
			TargetedMutationCount: 1,
			ActionCount:           2,
			AssistantTurns:        1,
		},
	}

	got := ComputePerformance(conversations, timeRange, sequence)
	metric := findPerformanceMetric(t, got.Outcome.Metrics, perfMetricFirstPassResolution)

	require.Len(t, metric.Series, 3)
	assert.Equal(t, 1, metric.Series[0].SampleCount)
	assert.InDelta(t, 1.0, metric.Series[0].Value, 0.0001)
	assert.Equal(t, 1, metric.Series[1].SampleCount)
	assert.InDelta(t, 0.0, metric.Series[1].Value, 0.0001)
	assert.Equal(t, 1, metric.Series[2].SampleCount)
	assert.InDelta(t, 1.0, metric.Series[2].Value, 0.0001)
}

func makePerformanceConversations(
	provider conv.Provider,
	prefix string,
	start time.Time,
	count int,
	update func(*conv.SessionMeta),
) []conv.Conversation {
	conversations := make([]conv.Conversation, 0, count)
	for i := range count {
		meta := testMeta(
			prefix+FormatNumber(i),
			start.AddDate(0, 0, i),
			withProject(prefix),
		)
		update(&meta)
		conversations = append(conversations, testConversation(provider, meta.ID, meta))
	}
	return conversations
}

func makeSequenceSessions(start time.Time, count int, resolved bool) []PerformanceSequenceSession {
	sessions := make([]PerformanceSequenceSession, 0, count)
	for i := range count {
		session := PerformanceSequenceSession{
			Timestamp:             start.AddDate(0, 0, i),
			Mutated:               true,
			MutationCount:         1,
			TargetedMutationCount: 1,
			ActionCount:           3,
			AssistantTurns:        2,
			VisibleReasoningChars: 40,
		}
		if resolved {
			session.VerificationPassed = true
			session.FirstPassResolved = true
		} else {
			session.BlindMutationCount = 1
			session.CorrectionFollowups = 1
		}
		sessions = append(sessions, session)
	}
	return sessions
}

func testPerformanceConversation(
	provider conv.Provider,
	id string,
	session conv.SessionMeta,
) conv.Conversation {
	return testConversation(provider, id, session)
}

func testPerformanceSessionMeta(
	id, project string,
	timestamp time.Time,
	update func(*conv.SessionMeta),
) conv.SessionMeta {
	meta := testMeta(id, timestamp, withProject(project))
	meta.Model = "claude-opus-4-1"
	meta.UserMessageCount = 2
	if update != nil {
		update(&meta)
	}
	return meta
}

func findPerformanceMetric(t *testing.T, metrics []PerformanceMetric, id string) PerformanceMetric {
	t.Helper()

	metric := tryFindPerformanceMetric(metrics, id)
	require.NotNil(t, metric)
	return *metric
}

func tryFindPerformanceMetric(metrics []PerformanceMetric, id string) *PerformanceMetric {
	for i := range metrics {
		if metrics[i].ID == id {
			return &metrics[i]
		}
	}
	return nil
}

func findPerformanceDiagnostic(
	t *testing.T,
	diagnostics []PerformanceDiagnostic,
	label string,
) PerformanceDiagnostic {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if diagnostic.Label == label {
			return diagnostic
		}
	}
	require.FailNowf(t, "missing diagnostic", "label %q not found", label)
	return PerformanceDiagnostic{}
}
