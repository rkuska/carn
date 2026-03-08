package app

import (
	"testing"
	"time"
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
	if len(got.messages) != 5 {
		t.Fatalf("len(messages) = %d, want 5", len(got.messages))
	}
	if got.messages[1].isAgentDivider != true {
		t.Fatalf("messages[1].isAgentDivider = %v, want true", got.messages[1].isAgentDivider)
	}
	if got.messages[1].text != "sub task" {
		t.Fatalf("messages[1].text = %q, want %q", got.messages[1].text, "sub task")
	}
	if got.messages[2].text != "subagent prompt" {
		t.Fatalf("messages[2].text = %q, want %q", got.messages[2].text, "subagent prompt")
	}
	if got.messages[4].text != "parent two" {
		t.Fatalf("messages[4].text = %q, want %q", got.messages[4].text, "parent two")
	}
}
