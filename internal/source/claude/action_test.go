package claude

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestClassifyClaudeToolAction(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		toolName   string
		input      map[string]any
		wantType   conv.NormalizedActionType
		wantTarget conv.ActionTarget
	}{
		{
			name:     "read file",
			toolName: "Read",
			input: map[string]any{
				"file_path": "/tmp/main.go",
			},
			wantType: conv.NormalizedActionRead,
			wantTarget: conv.ActionTarget{
				Type:  conv.ActionTargetFilePath,
				Value: "/tmp/main.go",
			},
		},
		{
			name:     "rewrite file",
			toolName: "Write",
			input: map[string]any{
				"file_path": "/tmp/main.go",
			},
			wantType: conv.NormalizedActionRewrite,
			wantTarget: conv.ActionTarget{
				Type:  conv.ActionTargetFilePath,
				Value: "/tmp/main.go",
			},
		},
		{
			name:     "bash test command",
			toolName: "Bash",
			input: map[string]any{
				"command": "go test ./...",
			},
			wantType: conv.NormalizedActionTest,
			wantTarget: conv.ActionTarget{
				Type:  conv.ActionTargetCommand,
				Value: "go test ./...",
			},
		},
		{
			name:     "web search query",
			toolName: "WebSearch",
			input: map[string]any{
				"query": "codex reasoning tokens",
			},
			wantType: conv.NormalizedActionWeb,
			wantTarget: conv.ActionTarget{
				Type:  conv.ActionTargetQuery,
				Value: "codex reasoning tokens",
			},
		},
		{
			name:     "delegate task output",
			toolName: "TaskOutput",
			input: map[string]any{
				"task_id": "task-1",
			},
			wantType: conv.NormalizedActionDelegate,
			wantTarget: conv.ActionTarget{
				Type:  conv.ActionTargetPlanPath,
				Value: "task-1",
			},
		},
		{
			name:     "plan mode toggle",
			toolName: "EnterPlanMode",
			input:    map[string]any{},
			wantType: conv.NormalizedActionPlan,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			raw, err := json.Marshal(testCase.input)
			require.NoError(t, err)

			got := classifyClaudeToolAction(testCase.toolName, raw)
			assert.Equal(t, testCase.wantType, got.Type)
			if testCase.wantTarget.Type != "" {
				assert.Contains(t, got.Targets, testCase.wantTarget)
			}
		})
	}
}
