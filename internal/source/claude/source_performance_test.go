package claude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestSourceScanAndLoadCapturePerformanceMetadataAndActions(t *testing.T) {
	t.Parallel()

	rawDir := t.TempDir()
	projectDir := filepath.Join(rawDir, "project-a")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	path := filepath.Join(projectDir, "session-performance.jsonl")
	content := strings.Join([]string{
		marshalTestJSONLRecord(t, map[string]any{
			"type":             "user",
			"sessionId":        "s1",
			"slug":             "performance",
			"timestamp":        "2026-03-21T10:00:00Z",
			"cwd":              "/workspace/project",
			"thinkingMetadata": map[string]any{"maxThinkingTokens": 32000},
			"message": map[string]any{
				"role":    "user",
				"content": "fix the parser",
			},
		}),
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "assistant",
			"sessionId": "s1",
			"timestamp": "2026-03-21T10:00:01Z",
			"message": map[string]any{
				"role":        "assistant",
				"model":       "claude-sonnet-4",
				"stop_reason": "tool_use",
				"content": []map[string]any{
					{"type": "thinking", "thinking": "inspect the parser first"},
					{"type": "thinking", "thinking": "", "signature": "Ev8DCkYFakeSignature"},
					{"type": "text", "text": "Running a test and rewriting the file."},
					{
						"type": "tool_use",
						"id":   "toolu_write",
						"name": "Write",
						"input": map[string]any{
							"file_path": "/workspace/project/main.go",
						},
					},
					{
						"type": "tool_use",
						"id":   "toolu_bash",
						"name": "Bash",
						"input": map[string]any{
							"command": "go test ./...",
						},
					},
				},
				"usage": map[string]any{
					"input_tokens":  120,
					"output_tokens": 30,
					"server_tool_use": map[string]any{
						"web_search_requests": 2,
						"web_fetch_requests":  1,
					},
					"service_tier": "priority",
					"speed":        "fast",
				},
			},
		}),
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "user",
			"sessionId": "s1",
			"slug":      "performance",
			"timestamp": "2026-03-21T10:00:02Z",
			"message": map[string]any{
				"role": "user",
				"content": []map[string]any{
					{
						"type":        "tool_result",
						"tool_use_id": "toolu_write",
						"is_error":    true,
						"content":     "The tool use was rejected by the user.",
					},
					{
						"type":        "tool_result",
						"tool_use_id": "toolu_bash",
						"is_error":    true,
						"content":     "tests failed",
					},
					{
						"type": "text",
						"text": "try again",
					},
				},
			},
		}),
		marshalTestJSONLRecord(t, map[string]any{
			"type":       "system",
			"sessionId":  "s1",
			"timestamp":  "2026-03-21T10:00:03Z",
			"subtype":    "turn_duration",
			"durationMs": 1500,
			"content":    "turn finished",
		}),
		marshalTestJSONLRecord(t, map[string]any{
			"type":         "system",
			"sessionId":    "s1",
			"timestamp":    "2026-03-21T10:00:04Z",
			"subtype":      "api_error",
			"retryAttempt": 1,
			"retryInMs":    620.4,
			"maxRetries":   5,
			"error": map[string]any{
				"type": "overloaded_error",
			},
		}),
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "system",
			"sessionId": "s1",
			"timestamp": "2026-03-21T10:00:05Z",
			"subtype":   "turn_info",
			"compactMetadata": map[string]any{
				"trigger": "manual",
			},
		}),
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "system",
			"sessionId": "s1",
			"timestamp": "2026-03-21T10:00:06Z",
			"subtype":   "microcompact_boundary",
			"microcompactMetadata": map[string]any{
				"trigger": "auto",
			},
		}),
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	source := New()
	scanResult, err := source.Scan(context.Background(), rawDir)
	require.NoError(t, err)
	require.Len(t, scanResult.Conversations, 1)

	meta := scanResult.Conversations[0].Sessions[0]
	assert.Equal(t, map[string]int{"rewrite": 1, "test": 1}, meta.ActionCounts)
	assert.Equal(t, map[string]int{"test": 1}, meta.ActionErrorCounts)
	assert.Equal(t, map[string]int{"rewrite": 1}, meta.ActionRejectCounts)
	assert.Equal(t, 32000, meta.Performance.MaxThinkingTokens)
	assert.Equal(t, map[string]int{"tool_use": 1}, meta.Performance.StopReasonCounts)
	assert.Equal(
		t,
		map[string]int{"web_fetch_requests": 1, "web_search_requests": 2},
		meta.Performance.ServerToolUseCounts,
	)
	assert.Equal(t, map[string]int{"priority": 1}, meta.Performance.ServiceTierCounts)
	assert.Equal(t, map[string]int{"fast": 1}, meta.Performance.SpeedCounts)
	assert.Equal(t, 2, meta.Performance.ReasoningBlockCount)
	assert.Equal(t, 1, meta.Performance.ReasoningRedactionCount)
	assert.Equal(t, 1500, meta.Performance.DurationMS)
	assert.Equal(t, 1, meta.Performance.RetryAttemptCount)
	assert.Equal(t, 620, meta.Performance.RetryDelayMS)
	assert.Equal(t, 5, meta.Performance.MaxRetries)
	assert.Equal(t, map[string]int{"overloaded_error": 1}, meta.Performance.APIErrorCounts)
	assert.Equal(t, 1, meta.Performance.CompactionCount)
	assert.Equal(t, 1, meta.Performance.MicroCompactionCount)

	session, err := source.Load(context.Background(), scanResult.Conversations[0])
	require.NoError(t, err)
	require.Len(t, session.Messages, 3)
	require.Len(t, session.Messages[1].ToolCalls, 2)
	require.Len(t, session.Messages[2].ToolResults, 2)

	assert.Equal(t, conv.NormalizedActionRewrite, session.Messages[1].ToolCalls[0].Action.Type)
	assert.Contains(t, session.Messages[1].ToolCalls[0].Action.Targets, conv.ActionTarget{
		Type:  conv.ActionTargetFilePath,
		Value: "/workspace/project/main.go",
	})
	assert.Equal(t, conv.NormalizedActionTest, session.Messages[1].ToolCalls[1].Action.Type)
	assert.Contains(t, session.Messages[1].ToolCalls[1].Action.Targets, conv.ActionTarget{
		Type:  conv.ActionTargetCommand,
		Value: "go test ./...",
	})
	assert.Equal(t, "tool_use", session.Messages[1].Performance.StopReason)
	assert.Equal(t, 2, session.Messages[1].Performance.ReasoningBlockCount)
	assert.Equal(t, 1, session.Messages[1].Performance.ReasoningRedactionCount)
	assert.Equal(t, conv.NormalizedActionRewrite, session.Messages[2].ToolResults[0].Action.Type)
	assert.Equal(t, conv.NormalizedActionTest, session.Messages[2].ToolResults[1].Action.Type)
	assert.Equal(t, meta.ActionCounts, session.Meta.ActionCounts)
	assert.Equal(t, meta.Performance, session.Meta.Performance)
}
