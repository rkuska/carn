package claude

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectConversationTranscriptMergesLinkedTranscripts(t *testing.T) {
	t.Parallel()

	baseTS := time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC)
	got := projectConversationTranscript(
		[]parsedMessage{
			{role: roleUser, text: "parent one", timestamp: baseTS},
			{role: roleAssistant, text: "parent two", timestamp: baseTS.Add(2 * time.Minute)},
		},
		[]parsedLinkedTranscript{
			{
				kind:   linkedTranscriptKindSubagent,
				title:  "sub task",
				anchor: baseTS.Add(time.Minute),
				messages: []parsedMessage{
					{role: roleUser, text: "subagent prompt", timestamp: baseTS.Add(time.Minute)},
					{role: roleAssistant, text: "subagent answer", timestamp: baseTS.Add(90 * time.Second)},
				},
			},
		},
	)

	require.Len(t, got, 5)
	assert.True(t, got[1].IsAgentDivider)
	assert.Equal(t, "sub task", got[1].Text)
	assert.Equal(t, "subagent prompt", got[2].Text)
	assert.Equal(t, "parent two", got[4].Text)
}

func TestProjectParsedMessagesKeepsViewerFieldsOnly(t *testing.T) {
	t.Parallel()

	got := projectParsedMessages([]parsedMessage{
		{
			role:      roleAssistant,
			timestamp: time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC),
			text:      "answer",
			thinking:  "reasoning",
			toolCalls: []parsedToolCall{
				{id: "toolu_1", name: "Read", summary: "/tmp/file.go"},
			},
			toolResults: []parsedToolResult{
				{
					toolUseID:   "toolu_1",
					toolName:    "Read",
					toolSummary: "/tmp/file.go",
					content:     "package main",
					isError:     true,
				},
			},
			usage:          tokenUsage{InputTokens: 10, OutputTokens: 5},
			isSidechain:    true,
			isAgentDivider: true,
		},
	})

	require.Len(t, got, 1)
	assert.Equal(t, message{
		Role:      roleAssistant,
		Text:      "answer",
		Thinking:  "reasoning",
		ToolCalls: []toolCall{{Name: "Read", Summary: "/tmp/file.go"}},
		ToolResults: []toolResult{{
			ToolName:    "Read",
			ToolSummary: "/tmp/file.go",
			Content:     "package main",
			IsError:     true,
		}},
		IsSidechain:    true,
		IsAgentDivider: true,
	}, got[0])
}
