package app

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
	assert.True(t, got[1].isAgentDivider)
	assert.Equal(t, "sub task", got[1].text)
	assert.Equal(t, "subagent prompt", got[2].text)
	assert.Equal(t, "parent two", got[4].text)
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
			usage:          tokenUsage{inputTokens: 10, outputTokens: 5},
			isSidechain:    true,
			isAgentDivider: true,
		},
	})

	require.Len(t, got, 1)
	assert.Equal(t, message{
		role:      roleAssistant,
		text:      "answer",
		thinking:  "reasoning",
		toolCalls: []toolCall{{name: "Read", summary: "/tmp/file.go"}},
		toolResults: []toolResult{{
			toolName:    "Read",
			toolSummary: "/tmp/file.go",
			content:     "package main",
			isError:     true,
		}},
		isSidechain:    true,
		isAgentDivider: true,
	}, got[0])
}
