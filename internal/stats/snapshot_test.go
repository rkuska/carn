package stats

import (
	"testing"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestComputeSnapshotFiltersByTimeRangeAndLeavesClaudeTurnMetricsNil(t *testing.T) {
	t.Parallel()

	conversations := []conversation{
		testConversation(
			conv.ProviderClaude,
			"old",
			testMeta("old", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), withMainMessages(3), withUsage(100, 0, 0, 20)),
		),
		testConversation(
			conv.ProviderClaude,
			"in-range",
			testMeta("in-range", time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC), withMainMessages(5), withUsage(200, 0, 0, 50)),
		),
	}

	got := ComputeSnapshot(conversations, TimeRange{
		Start: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 15, 23, 59, 59, 0, time.UTC),
	}, nil)

	if got.Overview.SessionCount != 1 || got.Overview.MessageCount != 5 {
		t.Fatalf("Overview = %#v", got.Overview)
	}
	if got.Sessions.ClaudeTurnMetrics != nil {
		t.Fatalf("ClaudeTurnMetrics = %#v, want nil", got.Sessions.ClaudeTurnMetrics)
	}
}

func TestComputeSnapshotAppliesLoadedPerformanceSequence(t *testing.T) {
	t.Parallel()

	currentTime := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
	conversations := []conversation{
		testConversation(
			conv.ProviderClaude,
			"current",
			testMeta(
				"current",
				currentTime,
				withModel("claude-sonnet-4"),
				func(meta *conv.SessionMeta) {
					meta.UserMessageCount = 1
					meta.ActionCounts = map[string]int{
						string(conv.NormalizedActionMutate): 1,
						string(conv.NormalizedActionTest):   1,
					}
				},
			),
		),
	}
	sequence := []PerformanceSequenceSession{
		{
			Timestamp:             currentTime,
			Mutated:               true,
			FirstPassResolved:     true,
			AssistantTurns:        1,
			MutationCount:         1,
			TargetedMutationCount: 1,
			ActionCount:           1,
		},
	}

	got := ComputeSnapshot(conversations, TimeRange{
		Start: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	}, sequence)

	if !got.Performance.Scope.SequenceLoaded {
		t.Fatalf("SequenceLoaded = false, want true")
	}
	if got.Performance.Scope.SequenceSampleCount != 1 {
		t.Fatalf("SequenceSampleCount = %d, want 1", got.Performance.Scope.SequenceSampleCount)
	}
}
