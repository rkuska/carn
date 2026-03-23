package claude

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	src "github.com/rkuska/carn/internal/source"
)

func TestKnownSchemaExtrasEntriesAreDocumented(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, claudeKnownSchemaExtras.Categories())

	for category, values := range claudeKnownSchemaExtras.Categories() {
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
		"envelope_field":     knownEnvelopeFields,
		"message_field":      knownMessageFields,
		"usage_field":        knownUsageFields,
		"record_type":        knownRecordTypes,
		"content_block_type": knownContentBlockTypes,
	}

	for category, values := range claudeKnownSchemaExtras.Categories() {
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
			name: "assistant message and usage extras",
			line: map[string]any{
				"type":       "assistant",
				"sessionId":  "session-1",
				"slug":       "demo",
				"timestamp":  "2026-03-21T10:00:01Z",
				"requestId":  "req_1",
				"userType":   "external",
				"cwd":        "/workspace/project",
				"gitBranch":  "main",
				"version":    "2.1.81",
				"entrypoint": "cli",
				"uuid":       "uuid-1",
				"message": map[string]any{
					"role":               "assistant",
					"id":                 "msg-1",
					"type":               "message",
					"model":              "claude-opus-4-6",
					"container":          nil,
					"stop_reason":        "end_turn",
					"stop_sequence":      nil,
					"context_management": nil,
					"usage": map[string]any{
						"input_tokens":                1,
						"output_tokens":               2,
						"cache_creation_input_tokens": 3,
						"cache_read_input_tokens":     4,
						"cache_creation": map[string]any{
							"ephemeral_5m_input_tokens": 0,
							"ephemeral_1h_input_tokens": 3,
						},
						"server_tool_use": map[string]any{
							"web_search_requests": 0,
							"web_fetch_requests":  0,
						},
						"service_tier":  "standard",
						"inference_geo": "not_available",
						"iterations":    []any{},
						"speed":         "standard",
					},
					"content": []map[string]any{
						{"type": "text", "text": "hello"},
					},
				},
			},
		},
		{
			name: "user image content and planning extras",
			line: map[string]any{
				"type":           "user",
				"sessionId":      "session-1",
				"slug":           "demo",
				"timestamp":      "2026-03-21T10:00:00Z",
				"promptId":       "prompt-1",
				"userType":       "external",
				"cwd":            "/workspace/project",
				"version":        "2.1.81",
				"gitBranch":      "main",
				"uuid":           "uuid-2",
				"permissionMode": "plan",
				"thinkingMetadata": map[string]any{
					"maxThinkingTokens": 32000,
				},
				"todos":           []any{},
				"planContent":     "# plan",
				"sourceToolUseID": "toolu_1",
				"message": map[string]any{
					"role": "user",
					"content": []map[string]any{
						{"type": "text", "text": "show image"},
						{
							"type": "image",
							"source": map[string]any{
								"type":       "base64",
								"media_type": "image/png",
								"data":       "iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB",
							},
						},
					},
				},
			},
		},
		{
			name: "system envelope extras",
			line: map[string]any{
				"type":              "system",
				"sessionId":         "session-1",
				"slug":              "demo",
				"timestamp":         "2026-03-21T10:00:02Z",
				"userType":          "external",
				"cwd":               "/workspace/project",
				"gitBranch":         "main",
				"version":           "2.1.81",
				"entrypoint":        "cli",
				"parentUuid":        "uuid-parent",
				"logicalParentUuid": "uuid-logical",
				"subtype":           "turn_duration",
				"durationMs":        1234,
				"content":           "Conversation compacted",
				"level":             "info",
				"uuid":              "uuid-3",
				"compactMetadata": map[string]any{
					"trigger":   "manual",
					"preTokens": 1000,
				},
			},
		},
		{
			name: "microcompact system extras",
			line: map[string]any{
				"type":      "system",
				"sessionId": "session-1",
				"slug":      "demo",
				"timestamp": "2026-01-30T09:44:21.056Z",
				"userType":  "external",
				"cwd":       "/workspace/project",
				"gitBranch": "main",
				"version":   "2.1.20",
				"subtype":   "microcompact_boundary",
				"content":   "Context microcompacted",
				"level":     "info",
				"uuid":      "uuid-4",
				"microcompactMetadata": map[string]any{
					"trigger":   "auto",
					"preTokens": 54410,
				},
			},
		},
		{
			name: "queue operation extras",
			line: map[string]any{
				"type":      "queue-operation",
				"operation": "enqueue",
				"timestamp": "2026-03-21T10:00:03Z",
				"sessionId": "session-1",
				"content":   "commit this",
			},
		},
		{
			name: "file history snapshot extras",
			line: map[string]any{
				"type":      "file-history-snapshot",
				"messageId": "msg-1",
				"snapshot": map[string]any{
					"messageId": "msg-1",
					"trackedFileBackups": map[string]any{
						"main.go": map[string]any{
							"backupFileName": "backup@v1",
							"version":        1,
						},
					},
					"timestamp": "2026-03-21T10:00:04Z",
				},
				"isSnapshotUpdate": true,
			},
		},
		{
			name: "progress extras",
			line: map[string]any{
				"type":            "progress",
				"sessionId":       "session-1",
				"timestamp":       "2026-03-21T10:00:05Z",
				"agentId":         "agent-1",
				"parentToolUseID": "toolu-parent",
				"toolUseID":       "toolu-child",
				"data": map[string]any{
					"type":      "hook_progress",
					"hookEvent": "PostToolUse",
				},
			},
		},
		{
			name: "agent name extras",
			line: map[string]any{
				"type":      "agent-name",
				"agentName": "format-drift-detection",
				"sessionId": "session-1",
			},
		},
		{
			name: "custom title extras",
			line: map[string]any{
				"type":        "custom-title",
				"customTitle": "format-drift-detection",
				"sessionId":   "session-1",
			},
		},
		{
			name: "last prompt extras",
			line: map[string]any{
				"type":       "last-prompt",
				"lastPrompt": "commit this",
				"sessionId":  "session-1",
			},
		},
		{
			name: "summary extras",
			line: map[string]any{
				"type":     "summary",
				"summary":  "Fix drift detection",
				"leafUuid": "leaf-1",
			},
		},
		{
			name: "pr link extras",
			line: map[string]any{
				"type":         "pr-link",
				"sessionId":    "session-1",
				"prNumber":     705,
				"prUrl":        "https://github.com/example/repo/pull/705",
				"prRepository": "example/repo",
				"timestamp":    "2026-03-12T12:07:59.289Z",
			},
		},
		{
			name: "assistant api error extras",
			line: map[string]any{
				"type":              "assistant",
				"sessionId":         "session-1",
				"slug":              "demo",
				"timestamp":         "2026-03-21T10:00:06Z",
				"userType":          "external",
				"cwd":               "/workspace/project",
				"gitBranch":         "main",
				"version":           "2.1.50",
				"uuid":              "uuid-5",
				"apiError":          "max_output_tokens",
				"error":             "max_output_tokens",
				"isApiErrorMessage": true,
				"message": map[string]any{
					"role":               "assistant",
					"id":                 "msg-2",
					"type":               "message",
					"model":              "<synthetic>",
					"container":          nil,
					"stop_reason":        "stop_sequence",
					"stop_sequence":      "",
					"context_management": nil,
					"usage": map[string]any{
						"input_tokens":                0,
						"output_tokens":               0,
						"cache_creation_input_tokens": 0,
						"cache_read_input_tokens":     0,
						"server_tool_use": map[string]any{
							"web_search_requests": 0,
						},
						"service_tier": nil,
						"cache_creation": map[string]any{
							"ephemeral_5m_input_tokens": 0,
						},
						"inference_geo": nil,
						"iterations":    nil,
						"speed":         nil,
					},
					"content": []map[string]any{
						{"type": "text", "text": "API Error"},
					},
				},
			},
		},
		{
			name: "system retry extras",
			line: map[string]any{
				"type":         "system",
				"sessionId":    "session-1",
				"slug":         "demo",
				"timestamp":    "2026-02-23T08:24:44.878Z",
				"userType":     "external",
				"cwd":          "/workspace/project",
				"gitBranch":    "main",
				"version":      "2.1.47",
				"subtype":      "api_error",
				"level":        "error",
				"uuid":         "uuid-6",
				"error":        map[string]any{"status": 529},
				"retryInMs":    617.8,
				"retryAttempt": 1,
				"maxRetries":   10,
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
