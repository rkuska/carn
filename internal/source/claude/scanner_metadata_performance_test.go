package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccumulateAssistantPerformanceStatsTracksReasoningAndToolCalls(t *testing.T) {
	t.Parallel()

	line := marshalTestJSONLRecord(t, map[string]any{
		"type":      "assistant",
		"sessionId": "s1",
		"timestamp": "2026-03-21T10:00:01Z",
		"message": map[string]any{
			"role": "assistant",
			"content": []map[string]any{
				{"type": "text", "text": "working"},
				{"type": "thinking", "thinking": "step one"},
				{"type": "thinking", "thinking": "", "signature": "sig"},
				{
					"type":  "tool_use",
					"id":    "toolu_read",
					"name":  "Read",
					"input": map[string]any{"file_path": "/workspace/project/main.go"},
				},
				{
					"type":  "tool_use",
					"id":    "toolu_bash",
					"name":  "Bash",
					"input": map[string]any{"command": "go test ./..."},
				},
			},
		},
	})

	var stats scanStats
	accumulateAssistantPerformanceStats([]byte(line), &stats)

	assert.Equal(t, map[string]int{"Read": 1, "Bash": 1}, stats.toolCounts)
	assert.Equal(t, map[string]int{"read": 1, "test": 1}, stats.actionCounts)
	assert.Equal(t, 2, stats.performance.ReasoningBlockCount)
	assert.Equal(t, 1, stats.performance.ReasoningRedactionCount)
	require.Contains(t, stats.toolCallByID, "toolu_read")
	require.Contains(t, stats.toolCallByID, "toolu_bash")
	assert.Equal(t, "Read", stats.toolCallByID["toolu_read"].Name)
	assert.Equal(t, "Bash", stats.toolCallByID["toolu_bash"].Name)
	assert.Equal(t, normalizedActionType("read"), stats.toolCallByID["toolu_read"].Action.Type)
	assert.Equal(t, normalizedActionType("test"), stats.toolCallByID["toolu_bash"].Action.Type)
}
