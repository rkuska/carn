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
	session := sessionFull{
		meta: sessionMeta{id: "parent"},
		messages: []message{
			{role: roleUser, text: "parent one", timestamp: baseTS},
			{role: roleAssistant, text: "parent two", timestamp: baseTS.Add(2 * time.Minute)},
		},
		linked: []linkedTranscript{
			{
				kind:   linkedTranscriptKindSubagent,
				title:  "sub task",
				anchor: baseTS.Add(time.Minute),
				messages: []message{
					{role: roleUser, text: "subagent prompt", timestamp: baseTS.Add(time.Minute)},
					{role: roleAssistant, text: "subagent answer", timestamp: baseTS.Add(90 * time.Second)},
				},
			},
		},
	}

	got := projectConversationTranscript(session)
	require.Len(t, got.messages, 5)
	assert.True(t, got.messages[1].isAgentDivider)
	assert.Equal(t, "sub task", got.messages[1].text)
	assert.Equal(t, "subagent prompt", got.messages[2].text)
	assert.Equal(t, "parent two", got.messages[4].text)
}
