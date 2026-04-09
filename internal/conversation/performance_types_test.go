package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenUsageTotalTokensIncludesReasoningOutput(t *testing.T) {
	t.Parallel()

	usage := TokenUsage{
		InputTokens:              10,
		CacheCreationInputTokens: 2,
		CacheReadInputTokens:     3,
		OutputTokens:             4,
		ReasoningOutputTokens:    5,
	}

	assert.Equal(t, 24, usage.TotalTokens())
}

func TestDeriveActionOutcomeCounts(t *testing.T) {
	t.Parallel()

	messages := []Message{
		{
			Role: RoleAssistant,
			ToolCalls: []ToolCall{
				{
					Name: "Read",
					Action: NormalizedAction{
						Type: NormalizedActionRead,
					},
				},
				{
					Name: "Bash",
					Action: NormalizedAction{
						Type: NormalizedActionTest,
					},
				},
			},
		},
		{
			Role: RoleUser,
			ToolResults: []ToolResult{
				{
					ToolName: "Read",
					IsError:  true,
					Content:  "The tool use was rejected by the user.",
					Action: NormalizedAction{
						Type: NormalizedActionRead,
					},
				},
				{
					ToolName: "Bash",
					IsError:  true,
					Content:  "go test failed",
					Action: NormalizedAction{
						Type: NormalizedActionTest,
					},
				},
			},
		},
	}

	assert.Equal(t, ActionOutcomeCounts{
		Calls:      map[string]int{"read": 1, "test": 1},
		Errors:     map[string]int{"test": 1},
		Rejections: map[string]int{"read": 1},
	}, DeriveActionOutcomeCounts(messages))
}
