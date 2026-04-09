package codex

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestClassifyCommand(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		command    string
		wantType   conv.NormalizedActionType
		wantTarget conv.ActionTarget
	}{
		{
			name:     "search command",
			command:  `rg parser internal/source/codex`,
			wantType: conv.NormalizedActionSearch,
			wantTarget: conv.ActionTarget{
				Type:  conv.ActionTargetPattern,
				Value: "parser",
			},
		},
		{
			name:     "read command",
			command:  `cat internal/source/codex/load.go`,
			wantType: conv.NormalizedActionRead,
			wantTarget: conv.ActionTarget{
				Type:  conv.ActionTargetFilePath,
				Value: "internal/source/codex/load.go",
			},
		},
		{
			name:     "test command",
			command:  `go test ./...`,
			wantType: conv.NormalizedActionTest,
			wantTarget: conv.ActionTarget{
				Type:  conv.ActionTargetCommand,
				Value: "go test ./...",
			},
		},
		{
			name:     "build command inside shell wrapper",
			command:  `/bin/zsh -lc "go build ./..."`,
			wantType: conv.NormalizedActionBuild,
			wantTarget: conv.ActionTarget{
				Type:  conv.ActionTargetCommand,
				Value: "go build ./...",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := classifyCommand(testCase.command)
			assert.Equal(t, testCase.wantType, got.actionType)
			assert.Contains(t, got.targets, testCase.wantTarget)
		})
	}
}

func TestParsePatchMetadataClassifiesRewriteWithoutReadEvidence(t *testing.T) {
	t.Parallel()

	patch := `*** Begin Patch
*** Update File: internal/source/codex/load.go
@@
-old line
-another old line
+new line
+another new line
*** End Patch`

	metadata := parsePatchMetadata(patch)
	require.Equal(t, []string{"internal/source/codex/load.go"}, metadata.Files)
	assert.Equal(t, conv.NormalizedActionRewrite, classifyPatchAction(metadata, nil))
	assert.Equal(
		t,
		conv.NormalizedActionMutate,
		classifyPatchAction(metadata, map[string]struct{}{"internal/source/codex/load.go": {}}),
	)
}

func TestScanAndLoadCapturePerformanceMetadataAndActions(t *testing.T) {
	t.Parallel()

	rawDir := t.TempDir()
	writeCodexRolloutFixture(t, rawDir, "2026/03/16/rollout-2026-03-16T10-00-00-action-metadata.jsonl", []map[string]any{
		codexSessionMetaLine("thread-action-metadata"),
		{
			"timestamp": "2026-03-16T10:00:00.100Z",
			"type":      recordTypeTurnContext,
			"payload": map[string]any{
				"cwd":    "/workspace/project",
				"model":  "gpt-5.4",
				"effort": "xhigh",
			},
		},
		codexUserMessageLine("2026-03-16T10:00:01Z", "Inspect the parser."),
		{
			"timestamp": "2026-03-16T10:00:02Z",
			"type":      recordTypeEventMsg,
			"payload": map[string]any{
				"type":                 eventTypeTaskStarted,
				"turn_id":              "turn-1",
				"model_context_window": 258400,
			},
		},
		{
			"timestamp": "2026-03-16T10:00:03Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":    responseTypeReasoning,
				"summary": []map[string]any{{"type": "summary_text", "text": "Checking parser files."}},
			},
		},
		{
			"timestamp": "2026-03-16T10:00:04Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":      responseTypeFunctionCall,
				"name":      "exec_command",
				"arguments": `{"cmd":"rg parser internal/source/codex"}`,
				"call_id":   "call-1",
			},
		},
		{
			"timestamp": "2026-03-16T10:00:05Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":    responseTypeFunctionCallOutput,
				"call_id": "call-1",
				"output":  "load.go: parser\n",
				"status":  "completed",
			},
		},
		{
			"timestamp": "2026-03-16T10:00:06Z",
			"type":      recordTypeEventMsg,
			"payload": map[string]any{
				"type":    eventTypeAgentMessage,
				"phase":   "commentary",
				"message": "Parser inspected.",
			},
		},
		{
			"timestamp": "2026-03-16T10:00:07Z",
			"type":      recordTypeEventMsg,
			"payload": map[string]any{
				"type":        eventTypeTokenCount,
				"rate_limits": nil,
				"info": map[string]any{
					"model_context_window": 258400,
					"total_token_usage": map[string]any{
						"input_tokens":            50,
						"cached_input_tokens":     5,
						"output_tokens":           10,
						"reasoning_output_tokens": 3,
					},
					"last_token_usage": map[string]any{
						"input_tokens":            20,
						"cached_input_tokens":     2,
						"output_tokens":           4,
						"reasoning_output_tokens": 1,
					},
				},
			},
		},
		{
			"timestamp": "2026-03-16T10:00:08Z",
			"type":      recordTypeEventMsg,
			"payload": map[string]any{
				"type":               eventTypeTaskComplete,
				"last_agent_message": "Parser inspected.",
			},
		},
	})

	source := New()
	scanResult, err := source.Scan(context.Background(), rawDir)
	require.NoError(t, err)
	require.Len(t, scanResult.Conversations, 1)

	meta := scanResult.Conversations[0].Sessions[0]
	assert.Equal(t, map[string]int{"search": 1}, meta.ActionCounts)
	assert.Equal(t, map[string]int{"xhigh": 1}, meta.Performance.EffortCounts)
	assert.Equal(t, map[string]int{"commentary": 1}, meta.Performance.PhaseCounts)
	assert.Equal(t, 1, meta.Performance.TaskStartedCount)
	assert.Equal(t, 1, meta.Performance.TaskCompleteCount)
	assert.Equal(t, 1, meta.Performance.RateLimitSnapshotCount)
	assert.Equal(t, 258400, meta.Performance.ModelContextWindow)
	assert.Equal(t, 1, meta.Performance.ReasoningBlockCount)

	session, err := source.Load(context.Background(), scanResult.Conversations[0])
	require.NoError(t, err)
	require.Len(t, session.Messages, 2)
	require.Len(t, session.Messages[1].ToolCalls, 1)
	assert.Equal(t, conv.NormalizedActionSearch, session.Messages[1].ToolCalls[0].Action.Type)
	assert.Contains(t, session.Messages[1].ToolCalls[0].Action.Targets, conv.ActionTarget{
		Type:  conv.ActionTargetPattern,
		Value: "parser",
	})
	assert.Equal(t, "commentary", session.Messages[1].Performance.Phase)
	assert.Equal(t, "xhigh", session.Messages[1].Performance.Effort)
	assert.Equal(t, 1, session.Messages[1].Performance.ReasoningBlockCount)
	assert.Equal(t, 1, session.Messages[1].Usage.ReasoningOutputTokens)
}
