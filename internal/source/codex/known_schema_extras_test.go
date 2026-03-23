package codex

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	src "github.com/rkuska/carn/internal/source"
)

func TestKnownSchemaExtrasEntriesAreDocumented(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, codexKnownSchemaExtras.Categories())

	for category, values := range codexKnownSchemaExtras.Categories() {
		require.NotEmptyf(t, values, "category %s", category)

		for value, entry := range values {
			assert.NotEmptyf(t, entry.Status, "%s/%s", category, value)
			assert.NotEmptyf(t, entry.Path, "%s/%s", category, value)
			assert.NotEmptyf(t, entry.RecordTypes, "%s/%s", category, value)
			assert.NotEmptyf(t, entry.Description, "%s/%s", category, value)
			assert.NotEmptyf(t, entry.FutureUse, "%s/%s", category, value)
			assert.NotEmptyf(t, entry.FirstSeen, "%s/%s", category, value)
			assert.NotEmptyf(t, entry.Example, "%s/%s", category, value)

			var decoded any
			require.NoErrorf(t, json.Unmarshal(entry.Example, &decoded), "%s/%s", category, value)
		}
	}
}

func TestKnownSchemaExtrasDoNotDuplicateCoreSchema(t *testing.T) {
	t.Parallel()

	core := map[string]map[string]struct{}{
		"record_type":                  knownRecordTypes,
		"session_meta_field":           knownSessionMetaFields,
		"git_field":                    knownGitFields,
		"turn_context_field":           knownTurnContextFields,
		"response_item_type":           knownResponseItemTypes,
		"response_message_field":       knownResponseMessageFields,
		"content_block_type":           knownContentBlockTypes,
		"event_type":                   knownEventTypes,
		"user_message_field":           knownUserMessageFields,
		"agent_message_field":          knownAgentMessageFields,
		"item_completed_field":         knownItemCompletedFields,
		"task_complete_field":          knownTaskCompleteFields,
		"token_count_field":            knownTokenCountFields,
		"token_count_info_field":       knownTokenCountInfoFields,
		"reasoning_summary_block_type": knownReasoningSummaryBlockTypes,
	}

	for category, values := range codexKnownSchemaExtras.Categories() {
		coreValues, ok := core[category]
		if !ok {
			continue
		}
		for value := range values {
			assert.NotContainsf(t, coreValues, value, "%s/%s", category, value)
		}
	}
}

func TestDetectLineDriftSuppressesKnownSchemaExtras(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		line map[string]any
	}{
		{
			name: "session meta extras",
			line: map[string]any{
				"timestamp": "2026-03-23T08:44:47.855Z",
				"type":      recordTypeSessionMeta,
				"payload": map[string]any{
					"id":                "thread-1",
					"timestamp":         "2026-03-23T08:44:25.966Z",
					"cwd":               "/workspace/project",
					"originator":        "codex_cli_rs",
					"cli_version":       "0.116.0",
					"source":            "cli",
					"model_provider":    "openai",
					"instructions":      "custom",
					"base_instructions": map[string]any{"text": "base"},
					"agent_nickname":    "Feynman",
					"agent_role":        "explorer",
					"forked_from_id":    "thread-0",
					"git": map[string]any{
						"branch":         "main",
						"commit_hash":    "abc123",
						"repository_url": "git@github.com:org/repo.git",
					},
				},
			},
		},
		{
			name: "turn context extras",
			line: map[string]any{
				"timestamp": "2026-03-23T08:44:47.856Z",
				"type":      recordTypeTurnContext,
				"payload": map[string]any{
					"turn_id":                "turn-1",
					"cwd":                    "/workspace/project",
					"current_date":           "2026-03-23",
					"timezone":               "Europe/Prague",
					"approval_policy":        "on-request",
					"sandbox_policy":         map[string]any{"type": "workspace-write"},
					"model":                  "gpt-5.4",
					"personality":            "pragmatic",
					"collaboration_mode":     map[string]any{"mode": "default"},
					"realtime_active":        false,
					"effort":                 "xhigh",
					"summary":                "none",
					"user_instructions":      "project rules",
					"developer_instructions": "top-level developer instructions",
					"truncation_policy":      map[string]any{"mode": "tokens", "limit": 10000},
				},
			},
		},
		{
			name: "response item extras",
			line: map[string]any{
				"timestamp": "2026-03-23T08:44:47.965Z",
				"type":      recordTypeResponseItem,
				"payload": map[string]any{
					"type":  responseTypeMessage,
					"role":  responseRoleAssistant,
					"phase": "commentary",
					"content": []map[string]any{
						{"type": "output_text", "text": "hello"},
						{"type": "input_image", "image_url": "file:///tmp/example.png"},
					},
				},
			},
		},
		{
			name: "ghost snapshot record type extras",
			line: map[string]any{
				"timestamp": "2026-03-23T08:45:34.760Z",
				"type":      recordTypeResponseItem,
				"payload": map[string]any{
					"type":         "ghost_snapshot",
					"ghost_commit": map[string]any{"id": "deadbeef"},
				},
			},
		},
		{
			name: "event extras",
			line: map[string]any{
				"timestamp": "2026-03-23T08:50:00.000Z",
				"type":      recordTypeEventMsg,
				"payload": map[string]any{
					"type":                 "task_started",
					"turn_id":              "turn-1",
					"model_context_window": 258400,
				},
			},
		},
		{
			name: "user and token metadata extras",
			line: map[string]any{
				"timestamp": "2026-03-23T08:50:00.000Z",
				"type":      recordTypeEventMsg,
				"payload": map[string]any{
					"type":         eventTypeUserMessage,
					"message":      "Inspect the parser.",
					"images":       []any{},
					"local_images": []any{},
					"text_elements": []map[string]any{
						{"type": "input_text", "text": "Inspect the parser."},
					},
				},
			},
		},
		{
			name: "agent completion and token extras",
			line: map[string]any{
				"timestamp": "2026-03-23T08:50:00.000Z",
				"type":      recordTypeEventMsg,
				"payload": map[string]any{
					"type":            eventTypeAgentMessage,
					"phase":           "commentary",
					"message":         "done",
					"memory_citation": nil,
				},
			},
		},
		{
			name: "item completed and task complete extras",
			line: map[string]any{
				"timestamp": "2026-03-23T08:50:00.000Z",
				"type":      recordTypeEventMsg,
				"payload": map[string]any{
					"type":      eventTypeItemCompleted,
					"thread_id": "thread-1",
					"turn_id":   "turn-1",
					"item": map[string]any{
						"type": eventItemTypePlan,
						"id":   "plan-1",
						"text": "plan",
					},
				},
			},
		},
		{
			name: "task complete extras",
			line: map[string]any{
				"timestamp": "2026-03-23T08:50:00.000Z",
				"type":      recordTypeEventMsg,
				"payload": map[string]any{
					"type":               eventTypeTaskComplete,
					"turn_id":            "turn-1",
					"last_agent_message": "done",
				},
			},
		},
		{
			name: "token count extras",
			line: map[string]any{
				"timestamp": "2026-03-23T08:50:00.000Z",
				"type":      recordTypeEventMsg,
				"payload": map[string]any{
					"type":        eventTypeTokenCount,
					"rate_limits": nil,
					"info": map[string]any{
						"total_token_usage": map[string]any{
							"input_tokens":            1,
							"cached_input_tokens":     2,
							"output_tokens":           3,
							"reasoning_output_tokens": 4,
							"total_tokens":            10,
						},
						"last_token_usage": map[string]any{
							"input_tokens":            1,
							"cached_input_tokens":     2,
							"output_tokens":           3,
							"reasoning_output_tokens": 4,
							"total_tokens":            10,
						},
						"model_context_window": 258400,
					},
				},
			},
		},
		{
			name: "compacted record extra",
			line: map[string]any{
				"timestamp": "2026-03-23T08:48:50.458Z",
				"type":      "compacted",
				"payload": map[string]any{
					"message":             "",
					"replacement_history": []map[string]any{{"type": "message"}},
				},
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

			assert.Empty(t, report.Findings())
		})
	}
}
