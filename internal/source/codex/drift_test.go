package codex

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	src "github.com/rkuska/carn/internal/source"
)

func TestDetectLineDrift(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		line map[string]any
		want []src.DriftFinding
	}{
		{
			name: "known session meta record",
			line: map[string]any{
				"timestamp": "2026-03-21T10:00:00Z",
				"type":      recordTypeSessionMeta,
				"payload": map[string]any{
					"id":             "thread-1",
					"timestamp":      "2026-03-21T10:00:00Z",
					"cwd":            "/workspace/project",
					"originator":     "codex_cli_rs",
					"cli_version":    "0.114.0",
					"source":         "cli",
					"model_provider": "openai",
					"git": map[string]any{
						"branch":      "main",
						"commit_hash": "abc123",
					},
				},
			},
			want: []src.DriftFinding{},
		},
		{
			name: "unknown envelope field",
			line: map[string]any{
				"timestamp": "2026-03-21T10:00:00Z",
				"type":      recordTypeSessionMeta,
				"payload": map[string]any{
					"id": "thread-1",
				},
				"transport": "cli",
			},
			want: []src.DriftFinding{
				{Category: "envelope_field", Value: "transport"},
			},
		},
		{
			name: "unknown session meta field",
			line: map[string]any{
				"timestamp": "2026-03-21T10:00:00Z",
				"type":      recordTypeSessionMeta,
				"payload": map[string]any{
					"id":             "thread-1",
					"timestamp":      "2026-03-21T10:00:00Z",
					"cwd":            "/workspace/project",
					"schema":         "v2",
					"originator":     "codex_cli_rs",
					"cli_version":    "0.114.0",
					"source":         "cli",
					"model_provider": "openai",
				},
			},
			want: []src.DriftFinding{
				{Category: "session_meta_field", Value: "schema"},
			},
		},
		{
			name: "unknown git field",
			line: map[string]any{
				"timestamp": "2026-03-21T10:00:00Z",
				"type":      recordTypeSessionMeta,
				"payload": map[string]any{
					"id":             "thread-1",
					"timestamp":      "2026-03-21T10:00:00Z",
					"cwd":            "/workspace/project",
					"originator":     "codex_cli_rs",
					"cli_version":    "0.114.0",
					"source":         "cli",
					"model_provider": "openai",
					"git": map[string]any{
						"branch":      "main",
						"commit_hash": "abc123",
						"remote":      "origin",
					},
				},
			},
			want: []src.DriftFinding{
				{Category: "git_field", Value: "remote"},
			},
		},
		{
			name: "unknown record type",
			line: map[string]any{
				"timestamp": "2026-03-21T10:00:00Z",
				"type":      "session_meta_v2",
				"payload":   map[string]any{},
			},
			want: []src.DriftFinding{
				{Category: "record_type", Value: "session_meta_v2"},
			},
		},
		{
			name: "unknown response item type",
			line: map[string]any{
				"timestamp": "2026-03-21T10:00:00Z",
				"type":      recordTypeResponseItem,
				"payload": map[string]any{
					"type": "file_reference",
				},
			},
			want: []src.DriftFinding{
				{Category: "response_item_type", Value: "file_reference"},
			},
		},
		{
			name: "unknown response message role and content block",
			line: map[string]any{
				"timestamp": "2026-03-21T10:00:00Z",
				"type":      recordTypeResponseItem,
				"payload": map[string]any{
					"type": "message",
					"role": "system",
					"content": []map[string]any{
						{"type": "image", "text": "diagram"},
					},
				},
			},
			want: []src.DriftFinding{
				{Category: "content_block_type", Value: "image"},
				{Category: "role", Value: "system"},
			},
		},
		{
			name: "unknown reasoning field and summary block type",
			line: map[string]any{
				"timestamp": "2026-03-21T10:00:00Z",
				"type":      recordTypeResponseItem,
				"payload": map[string]any{
					"type":              "reasoning",
					"summary":           []map[string]any{{"type": "summary_markdown", "text": "thinking"}},
					"encrypted_content": "opaque",
					"schema":            "v2",
				},
			},
			want: []src.DriftFinding{
				{Category: "reasoning_field", Value: "schema"},
				{Category: "reasoning_summary_block_type", Value: "summary_markdown"},
			},
		},
		{
			name: "unknown custom tool call field",
			line: map[string]any{
				"timestamp": "2026-03-21T10:00:00Z",
				"type":      recordTypeResponseItem,
				"payload": map[string]any{
					"type":    responseTypeCustomToolCall,
					"status":  "completed",
					"call_id": "call-1",
					"name":    "apply_patch",
					"input":   "*** Begin Patch\n*** End Patch",
					"result":  "ok",
				},
			},
			want: []src.DriftFinding{
				{Category: "custom_tool_call_field", Value: "result"},
			},
		},
		{
			name: "unknown event type",
			line: map[string]any{
				"timestamp": "2026-03-21T10:00:00Z",
				"type":      recordTypeEventMsg,
				"payload": map[string]any{
					"type": "agent_status",
				},
			},
			want: []src.DriftFinding{
				{Category: "event_type", Value: "agent_status"},
			},
		},
		{
			name: "unknown agent message field",
			line: map[string]any{
				"timestamp": "2026-03-21T10:00:00Z",
				"type":      recordTypeEventMsg,
				"payload": map[string]any{
					"type":    eventTypeAgentMessage,
					"phase":   "commentary",
					"message": "done",
					"stream":  "stderr",
				},
			},
			want: []src.DriftFinding{
				{Category: "agent_message_field", Value: "stream"},
			},
		},
		{
			name: "unknown token usage field",
			line: map[string]any{
				"timestamp": "2026-03-21T10:00:00Z",
				"type":      recordTypeEventMsg,
				"payload": map[string]any{
					"type": eventTypeTokenCount,
					"info": map[string]any{
						"total_token_usage": map[string]any{
							"input_tokens":            1,
							"cached_input_tokens":     2,
							"output_tokens":           3,
							"reasoning_output_tokens": 4,
							"total_tokens":            10,
							"audio_output_tokens":     5,
						},
					},
				},
			},
			want: []src.DriftFinding{
				{Category: "token_usage_field", Value: "audio_output_tokens"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			raw, err := json.Marshal(testCase.line)
			require.NoError(t, err)

			report := src.NewDriftReport()
			detectLineDrift(raw, &report)

			assert.Equal(t, testCase.want, report.Findings())
		})
	}
}

func TestScanRolloutKeepsFixtureCorpusDriftFree(t *testing.T) {
	t.Parallel()

	rawDir := filepath.Join("..", "..", "..", "testdata", "codex_raw", "2026", "03", "13")
	paths := []string{
		filepath.Join(rawDir, "rollout-2026-03-13T10-00-00-019cexample-main.jsonl"),
		filepath.Join(rawDir, "rollout-2026-03-13T10-05-00-019cexample-legacy.jsonl"),
		filepath.Join(rawDir, "rollout-2026-03-13T10-15-00-019cexample-hidden.jsonl"),
	}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()

			rollout, _, err := scanRollout(path)
			require.NoError(t, err)
			assert.True(t, rollout.drift.Empty())
		})
	}
}

func TestKnownCodexSchemaSets(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, []string{"timestamp", "type", "payload"}, setKeys(knownEnvelopeFields))
	assert.ElementsMatch(
		t,
		[]string{recordTypeSessionMeta, recordTypeTurnContext, recordTypeResponseItem, recordTypeEventMsg},
		setKeys(knownRecordTypes),
	)
	assert.ElementsMatch(
		t,
		[]string{"id", "timestamp", "cwd", "originator", "cli_version", "source", "model_provider", "git"},
		setKeys(knownSessionMetaFields),
	)
	assert.ElementsMatch(t, []string{"branch", "commit_hash"}, setKeys(knownGitFields))
	assert.ElementsMatch(
		t,
		[]string{"cwd", "model", "effort", "approval_policy", "sandbox_policy"},
		setKeys(knownTurnContextFields),
	)
	assert.ElementsMatch(
		t,
		[]string{
			responseTypeMessage,
			responseTypeReasoning,
			responseTypeFunctionCall,
			responseTypeCustomToolCall,
			responseTypeWebSearchCall,
			responseTypeFunctionCallOutput,
			responseTypeCustomToolCallOutput,
		},
		setKeys(knownResponseItemTypes),
	)
	assert.ElementsMatch(
		t,
		[]string{responseRoleUser, responseRoleAssistant, responseRoleDeveloper},
		setKeys(knownRoles),
	)
	assert.ElementsMatch(t, []string{"type", "role", "phase", "content"}, setKeys(knownResponseMessageFields))
	assert.ElementsMatch(t, []string{"input_text", "output_text"}, setKeys(knownContentBlockTypes))
	assert.ElementsMatch(
		t,
		[]string{"type", "summary", "content", "encrypted_content"},
		setKeys(knownReasoningFields),
	)
	assert.ElementsMatch(t, []string{"summary_text"}, setKeys(knownReasoningSummaryBlockTypes))
	assert.ElementsMatch(t, []string{"type", "name", "arguments", "call_id"}, setKeys(knownFunctionCallFields))
	assert.ElementsMatch(t, []string{"type", "status", "call_id", "name", "input"}, setKeys(knownCustomToolCallFields))
	assert.ElementsMatch(t, []string{"type", "call_id", "output", "status"}, setKeys(knownToolCallOutputFields))
	assert.ElementsMatch(t, []string{"type", "action", "call_id", "status"}, setKeys(knownWebSearchCallFields))
	assert.ElementsMatch(
		t,
		[]string{
			eventTypeTokenCount,
			eventTypeUserMessage,
			eventTypeAgentMessage,
			eventTypeAgentReasoning,
			eventTypeItemCompleted,
			eventTypeTaskStarted,
			eventTypeTaskComplete,
			eventTypeTurnAborted,
			eventTypeContextCompacted,
		},
		setKeys(knownEventTypes),
	)
	assert.ElementsMatch(t, []string{"type", "message"}, setKeys(knownUserMessageFields))
	assert.ElementsMatch(t, []string{"type", "phase", "message"}, setKeys(knownAgentMessageFields))
	assert.ElementsMatch(t, []string{"type", "text"}, setKeys(knownAgentReasoningFields))
	assert.ElementsMatch(t, []string{"type", "item"}, setKeys(knownItemCompletedFields))
	assert.ElementsMatch(t, []string{"type", "id", "text"}, setKeys(knownCompletedItemFields))
	assert.ElementsMatch(t, []string{eventItemTypePlan}, setKeys(knownCompletedItemTypes))
	assert.ElementsMatch(t, []string{"type", "turn_id", "model_context_window"}, setKeys(knownTaskStartedFields))
	assert.ElementsMatch(t, []string{"type", "last_agent_message"}, setKeys(knownTaskCompleteFields))
	assert.ElementsMatch(t, []string{"type", "turn_id"}, setKeys(knownTurnAbortedFields))
	assert.ElementsMatch(t, []string{"type"}, setKeys(knownContextCompactedFields))
	assert.ElementsMatch(t, []string{"type", "rate_limits", "info"}, setKeys(knownTokenCountFields))
	assert.ElementsMatch(
		t,
		[]string{"total_token_usage", "last_token_usage", "model_context_window"},
		setKeys(knownTokenCountInfoFields),
	)
	assert.ElementsMatch(
		t,
		[]string{
			"input_tokens",
			"cached_input_tokens",
			"output_tokens",
			"reasoning_output_tokens",
			"total_tokens",
		},
		setKeys(knownTokenUsageFields),
	)
}

func setKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
