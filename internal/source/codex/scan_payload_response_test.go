package codex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestBuildScannedToolCallPreservesSummaryAndAction(t *testing.T) {
	t.Parallel()

	item := collectResponseItemPayload([]byte(
		`{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"go test ./...\"}","call_id":"call-1"}`,
	))

	call, ok := buildScannedToolCall(item, nil)
	require.True(t, ok)

	assert.Equal(t, toolNameExecCommand, call.Name)
	assert.Equal(t, "go test ./...", call.Summary)
	assert.Equal(t, conv.NormalizedActionTest, call.Action.Type)
	assert.Contains(t, call.Action.Targets, conv.ActionTarget{
		Type:  conv.ActionTargetCommand,
		Value: "go test ./...",
	})
}
