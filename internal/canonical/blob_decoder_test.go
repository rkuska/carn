package canonical

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestDecodeSessionBlobFastPreservesDistinctActionTargets(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		Meta: sessionMeta{ID: "s1"},
		Messages: []message{
			{
				Role:      conv.RoleAssistant,
				Timestamp: time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC),
				ToolCalls: []toolCall{
					{
						Name: "Read",
						Action: conv.NormalizedAction{
							Type: conv.NormalizedActionRead,
							Targets: []conv.ActionTarget{{
								Type:  conv.ActionTargetFilePath,
								Value: "/tmp/one.go",
							}},
						},
					},
					{
						Name: "Bash",
						Action: conv.NormalizedAction{
							Type: conv.NormalizedActionTest,
							Targets: []conv.ActionTarget{{
								Type:  conv.ActionTargetCommand,
								Value: "go test ./...",
							}},
						},
					},
				},
				ToolResults: []toolResult{
					{
						ToolName: "Bash",
						Action: conv.NormalizedAction{
							Type: conv.NormalizedActionBuild,
							Targets: []conv.ActionTarget{{
								Type:  conv.ActionTargetCommand,
								Value: "go build ./...",
							}},
						},
					},
				},
			},
		},
	}

	var blob []byte
	require.NoError(t, withEncodedSessionBlob(session, func(encoded []byte) error {
		blob = append(blob[:0], encoded...)
		return nil
	}))

	decoded, err := decodeSessionBlobFast(blob)
	require.NoError(t, err)
	require.Len(t, decoded.Messages, 1)
	require.Len(t, decoded.Messages[0].ToolCalls, 2)
	require.Len(t, decoded.Messages[0].ToolResults, 1)

	assert.True(t, decoded.Messages[0].Timestamp.Equal(session.Messages[0].Timestamp))
	assert.Equal(t, "/tmp/one.go", decoded.Messages[0].ToolCalls[0].Action.Targets[0].Value)
	assert.Equal(t, "go test ./...", decoded.Messages[0].ToolCalls[1].Action.Targets[0].Value)
	assert.Equal(t, "go build ./...", decoded.Messages[0].ToolResults[0].Action.Targets[0].Value)

	decoded.Messages[0].ToolCalls[0].Action.Targets[0].Value = "/tmp/changed.go"
	assert.Equal(t, "go test ./...", decoded.Messages[0].ToolCalls[1].Action.Targets[0].Value)
	assert.Equal(t, "go build ./...", decoded.Messages[0].ToolResults[0].Action.Targets[0].Value)
}
