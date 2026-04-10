package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestCollectPerformanceSequenceSessionsSummarizesMutationBehavior(t *testing.T) {
	t.Parallel()

	sessions := []conv.Session{
		{
			Meta: conv.SessionMeta{Timestamp: time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)},
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "fix a.go"},
				{
					Role:     conv.RoleAssistant,
					Thinking: "inspect first",
					ToolCalls: []conv.ToolCall{{
						Name: "Read",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionRead,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: "/tmp/a.go"}},
						},
					}},
				},
				{
					Role: conv.RoleAssistant,
					ToolCalls: []conv.ToolCall{{
						Name: "Edit",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionMutate,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: "/tmp/a.go"}},
						},
					}},
				},
				{
					Role: conv.RoleUser,
					ToolResults: []conv.ToolResult{{
						ToolName: "Edit",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionMutate,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: "/tmp/a.go"}},
						},
					}},
				},
				{
					Role: conv.RoleAssistant,
					ToolCalls: []conv.ToolCall{{
						Name: "Bash",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionTest,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: "go test ./..."}},
						},
					}},
				},
				{
					Role: conv.RoleUser,
					ToolResults: []conv.ToolResult{{
						ToolName: "Bash",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionTest,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: "go test ./..."}},
						},
					}},
				},
			},
		},
		{
			Meta: conv.SessionMeta{Timestamp: time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC)},
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "fix b.go"},
				{
					Role:              conv.RoleAssistant,
					HasHiddenThinking: true,
					ToolCalls: []conv.ToolCall{{
						Name: "Edit",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionMutate,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: "/tmp/b.go"}},
						},
					}},
				},
				{
					Role: conv.RoleUser,
					Text: "inspect first",
					ToolResults: []conv.ToolResult{{
						ToolName: "Edit",
						IsError:  true,
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionMutate,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: "/tmp/b.go"}},
						},
					}},
				},
				{
					Role: conv.RoleAssistant,
					ToolCalls: []conv.ToolCall{{
						Name: "Edit",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionMutate,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: "/tmp/b.go"}},
						},
					}},
				},
			},
		},
	}

	got := CollectPerformanceSequenceSessions(sessions)

	require.Len(t, got, 2)
	assert.True(t, got[0].Mutated)
	assert.True(t, got[0].FirstPassResolved)
	assert.True(t, got[0].VerificationPassed)
	assert.Zero(t, got[0].BlindMutationCount)
	assert.Equal(t, 1, got[0].ActionsBeforeFirstMutation)

	assert.True(t, got[1].Mutated)
	assert.False(t, got[1].FirstPassResolved)
	assert.Equal(t, 2, got[1].BlindMutationCount)
	assert.Equal(t, 1, got[1].CorrectionFollowups)
	assert.Equal(t, 1, got[1].HiddenThinkingTurns)
}

func TestCollectPerformanceSequenceSessionsTreatsPostMutationFailureAsUnresolved(t *testing.T) {
	t.Parallel()

	sessions := []conv.Session{
		{
			Meta: conv.SessionMeta{Timestamp: time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)},
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "fix a.go"},
				{
					Role: conv.RoleAssistant,
					ToolCalls: []conv.ToolCall{{
						Name: "Read",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionRead,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: "/tmp/a.go"}},
						},
					}},
				},
				{
					Role: conv.RoleAssistant,
					ToolCalls: []conv.ToolCall{{
						Name: "Edit",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionMutate,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: "/tmp/a.go"}},
						},
					}},
				},
				{
					Role: conv.RoleUser,
					ToolResults: []conv.ToolResult{{
						ToolName: "Edit",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionMutate,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: "/tmp/a.go"}},
						},
					}},
				},
				{
					Role: conv.RoleAssistant,
					ToolCalls: []conv.ToolCall{{
						Name: "Bash",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionTest,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: "go test ./..."}},
						},
					}},
				},
				{
					Role: conv.RoleUser,
					ToolResults: []conv.ToolResult{{
						ToolName: "Bash",
						IsError:  true,
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionTest,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: "go test ./..."}},
						},
					}},
				},
			},
		},
	}

	got := CollectPerformanceSequenceSessions(sessions)

	require.Len(t, got, 1)
	assert.True(t, got[0].Mutated)
	assert.False(t, got[0].VerificationPassed)
	assert.False(t, got[0].FirstPassResolved)
}

func TestCollectPerformanceSequenceSessionsStopsFollowupCountingAfterVerification(t *testing.T) {
	t.Parallel()

	sessions := []conv.Session{
		{
			Meta: conv.SessionMeta{Timestamp: time.Date(2026, 2, 4, 12, 0, 0, 0, time.UTC)},
			Messages: []conv.Message{
				{Role: conv.RoleUser, Text: "fix a.go"},
				{
					Role: conv.RoleAssistant,
					ToolCalls: []conv.ToolCall{{
						Name: "Read",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionRead,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: "/tmp/a.go"}},
						},
					}},
				},
				{
					Role: conv.RoleAssistant,
					ToolCalls: []conv.ToolCall{{
						Name: "Edit",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionMutate,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: "/tmp/a.go"}},
						},
					}},
				},
				{
					Role: conv.RoleUser,
					ToolResults: []conv.ToolResult{{
						ToolName: "Edit",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionMutate,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: "/tmp/a.go"}},
						},
					}},
				},
				{
					Role: conv.RoleAssistant,
					ToolCalls: []conv.ToolCall{{
						Name: "Bash",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionTest,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: "go test ./..."}},
						},
					}},
				},
				{
					Role: conv.RoleUser,
					ToolResults: []conv.ToolResult{{
						ToolName: "Bash",
						Action: conv.NormalizedAction{
							Type:    conv.NormalizedActionTest,
							Targets: []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: "go test ./..."}},
						},
					}},
				},
				{Role: conv.RoleUser, Text: "next task"},
			},
		},
	}

	got := CollectPerformanceSequenceSessions(sessions)

	require.Len(t, got, 1)
	assert.True(t, got[0].VerificationPassed)
	assert.Zero(t, got[0].CorrectionFollowups)
}

func TestCollectPerformanceSequenceSessionsCountsVisibleReasoningRunes(t *testing.T) {
	t.Parallel()

	sessions := []conv.Session{
		{
			Meta: conv.SessionMeta{Timestamp: time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)},
			Messages: []conv.Message{
				{
					Role:     conv.RoleAssistant,
					Thinking: " áβ ",
				},
			},
		},
	}

	got := CollectPerformanceSequenceSessions(sessions)

	require.Len(t, got, 1)
	assert.Equal(t, 2, got[0].VisibleReasoningChars)
}

func TestAddPerformanceSequenceSessionUsesFullPatchHunkCount(t *testing.T) {
	t.Parallel()

	var aggregate performanceSequenceAggregate
	addPerformanceSequenceSession(&aggregate, PerformanceSequenceSession{
		Mutated:                 true,
		MutationCount:           1,
		DistinctMutationTargets: 2,
		PatchHunkCount:          4,
	})

	assert.Equal(t, 7, aggregate.patchChurn)
}
