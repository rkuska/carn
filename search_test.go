package main

import (
	"context"
	"strings"
	"testing"
)

func TestBuildSessionSearchBlob(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{
				role:     roleUser,
				text:     "Hello User",
				thinking: "Internal Thought",
				toolCalls: []toolCall{
					{name: "Read", summary: "README.md"},
				},
				toolResults: []toolResult{
					{toolUseID: "t1", content: "Tool Output"},
				},
			},
		},
	}

	got := buildSessionSearchBlob(session)
	for _, want := range []string{"hello user", "internal thought", "readme.md", "tool output"} {
		if !strings.Contains(got, want) {
			t.Errorf("buildSessionSearchBlob() missing %q in %q", want, got)
		}
	}
}

func TestDeepSearchCmd_UsesSessionCache(t *testing.T) {
	t.Parallel()

	mainSessions := []sessionMeta{
		{id: "s1", filePath: "/nonexistent/s1.jsonl"},
		{id: "s2", filePath: "/nonexistent/s2.jsonl"},
	}
	sessionCache := map[string]sessionFull{
		"s1": {messages: []message{{role: roleUser, text: "nothing here"}}},
		"s2": {messages: []message{{role: roleAssistant, text: "contains needle"}}},
	}

	msg := deepSearchCmd(context.Background(), "needle", mainSessions, nil, sessionCache)()
	result, ok := msg.(deepSearchResultMsg)
	if !ok {
		t.Fatalf("unexpected msg type: %T", msg)
	}
	if len(result.sessions) != 1 {
		t.Fatalf("matches = %d, want 1", len(result.sessions))
	}
	if result.sessions[0].id != "s2" {
		t.Fatalf("matched id = %q, want s2", result.sessions[0].id)
	}
	if len(result.indexed) != 2 {
		t.Fatalf("indexed len = %d, want 2", len(result.indexed))
	}
}

func TestDeepSearchCmd_UsesExistingIndexCache(t *testing.T) {
	t.Parallel()

	mainSessions := []sessionMeta{{id: "s1", filePath: "/nonexistent/s1.jsonl"}}
	indexCache := map[string]string{"s1": "cached needle content"}

	msg := deepSearchCmd(context.Background(), "needle", mainSessions, indexCache, nil)()
	result, ok := msg.(deepSearchResultMsg)
	if !ok {
		t.Fatalf("unexpected msg type: %T", msg)
	}
	if len(result.sessions) != 1 {
		t.Fatalf("matches = %d, want 1", len(result.sessions))
	}
	if len(result.indexed) != 0 {
		t.Fatalf("indexed len = %d, want 0", len(result.indexed))
	}
}

func TestDeepSearchCmd_EmptyQueryReturnsMainSessions(t *testing.T) {
	t.Parallel()

	mainSessions := []sessionMeta{{id: "s1"}, {id: "s2"}}
	msg := deepSearchCmd(context.Background(), "", mainSessions, nil, nil)()
	result, ok := msg.(deepSearchResultMsg)
	if !ok {
		t.Fatalf("unexpected msg type: %T", msg)
	}
	if len(result.sessions) != 2 {
		t.Fatalf("sessions len = %d, want 2", len(result.sessions))
	}
}
