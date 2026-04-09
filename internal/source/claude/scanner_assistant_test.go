package claude

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestExtractAssistantContentVisibleThinking(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`[
		{"type":"thinking","thinking":"reasoning about the problem"},
		{"type":"text","text":"the answer"}
	]`)

	text, thinking, hiddenThinking, toolCalls, _, ok := extractAssistantContent(raw)
	require.True(t, ok)
	assert.Equal(t, "the answer", text)
	assert.Equal(t, "reasoning about the problem", thinking)
	assert.False(t, hiddenThinking)
	assert.Empty(t, toolCalls)
}

func TestExtractAssistantContentSignedEmptyThinking(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`[
		{"type":"thinking","thinking":"","signature":"Ev8DCkYICxgCFakeSignature"},
		{"type":"text","text":"the answer"}
	]`)

	text, thinking, hiddenThinking, toolCalls, _, ok := extractAssistantContent(raw)
	require.True(t, ok)
	assert.Equal(t, "the answer", text)
	assert.Empty(t, thinking)
	assert.True(t, hiddenThinking)
	assert.Empty(t, toolCalls)
}

func TestExtractAssistantContentSignedWithVisibleThinking(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`[
		{"type":"thinking","thinking":"visible reasoning","signature":"Ev8DCkYICxgCFakeSignature"},
		{"type":"text","text":"the answer"}
	]`)

	text, thinking, hiddenThinking, _, _, ok := extractAssistantContent(raw)
	require.True(t, ok)
	assert.Equal(t, "the answer", text)
	assert.Equal(t, "visible reasoning", thinking)
	assert.False(t, hiddenThinking)
}

func TestExtractAssistantContentNoSignatureNoThinking(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`[
		{"type":"thinking","thinking":""},
		{"type":"text","text":"the answer"}
	]`)

	_, thinking, hiddenThinking, _, _, ok := extractAssistantContent(raw)
	require.True(t, ok)
	assert.Empty(t, thinking)
	assert.False(t, hiddenThinking)
}

func TestExtractAssistantContentCapturesPerformanceAndActions(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`[
		{"type":"thinking","thinking":"inspect"},
		{"type":"thinking","thinking":"","signature":"Ev8DCkYICxgCFakeSignature"},
		{"type":"tool_use","id":"toolu_1","name":"Write","input":{"file_path":"/tmp/main.go"}},
		{"type":"tool_use","id":"toolu_2","name":"Bash","input":{"command":"go test ./..."}},
		{"type":"text","text":"done"}
	]`)

	extracted, ok := extractAssistantContentDetails(raw)
	require.True(t, ok)
	require.Len(t, extracted.toolCalls, 2)
	assert.Equal(t, "done", extracted.text)
	assert.Equal(t, "inspect", extracted.thinking)
	assert.Equal(t, 2, extracted.performance.ReasoningBlockCount)
	assert.Equal(t, 1, extracted.performance.ReasoningRedactionCount)
	assert.Equal(t, conv.NormalizedActionRewrite, extracted.toolCalls[0].Action.Type)
	assert.Contains(t, extracted.toolCalls[0].Action.Targets, conv.ActionTarget{
		Type:  conv.ActionTargetFilePath,
		Value: "/tmp/main.go",
	})
	assert.Equal(t, conv.NormalizedActionTest, extracted.toolCalls[1].Action.Type)
	assert.Contains(t, extracted.toolCalls[1].Action.Targets, conv.ActionTarget{
		Type:  conv.ActionTargetCommand,
		Value: "go test ./...",
	})
}

func TestSummarizeToolCallFast(t *testing.T) {
	t.Parallel()

	longCommand := "go test ./... && go test -race ./... && golangci-lint run ./... && echo done"

	testCases := []struct {
		name  string
		tool  string
		input map[string]any
		want  string
	}{
		{
			name:  "read file path",
			tool:  "Read",
			input: map[string]any{"file_path": "/tmp/main.go"},
			want:  "/tmp/main.go",
		},
		{
			name:  "bash command is truncated",
			tool:  "Bash",
			input: map[string]any{"command": longCommand},
			want:  conv.Truncate(longCommand, 80),
		},
		{
			name:  "plan mode constant",
			tool:  "EnterPlanMode",
			input: map[string]any{},
			want:  "enter plan mode",
		},
		{
			name:  "mcp query",
			tool:  "mcp__context7__resolve-library-id",
			input: map[string]any{"query": "zod"},
			want:  "zod",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			raw, err := json.Marshal(testCase.input)
			require.NoError(t, err)
			assert.Equal(t, testCase.want, summarizeToolCallFast(testCase.tool, raw))
		})
	}
}
