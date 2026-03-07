package app

import (
	"context"
	"strings"
	"testing"
	"time"
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

func testConversation(id, slug string) conversation {
	return conversation{
		name:    slug,
		project: project{dirName: "test", displayName: "test"},
		sessions: []sessionMeta{
			{id: id, slug: slug, filePath: "/nonexistent/" + id + ".jsonl", timestamp: time.Now()},
		},
	}
}

func TestDeepSearchCmd_UsesSessionCache(t *testing.T) {
	t.Parallel()

	mainConvs := []conversation{
		testConversation("s1", "slug-1"),
		testConversation("s2", "slug-2"),
	}
	sessionCache := map[string]sessionFull{
		"s1": {messages: []message{{role: roleUser, text: "nothing here"}}},
		"s2": {messages: []message{{role: roleAssistant, text: "contains needle"}}},
	}

	msg := deepSearchCmd(context.Background(), "needle", mainConvs, nil, sessionCache)()
	result, ok := msg.(deepSearchResultMsg)
	if !ok {
		t.Fatalf("unexpected msg type: %T", msg)
	}
	if len(result.conversations) != 1 {
		t.Fatalf("matches = %d, want 1", len(result.conversations))
	}
	if result.conversations[0].id() != "s2" {
		t.Fatalf("matched id = %q, want s2", result.conversations[0].id())
	}
	if len(result.indexed) != 2 {
		t.Fatalf("indexed len = %d, want 2", len(result.indexed))
	}
}

func TestDeepSearchCmd_UsesExistingIndexCache(t *testing.T) {
	t.Parallel()

	mainConvs := []conversation{
		testConversation("s1", "slug-1"),
	}
	indexCache := map[string]string{"s1": "cached needle content"}

	msg := deepSearchCmd(context.Background(), "needle", mainConvs, indexCache, nil)()
	result, ok := msg.(deepSearchResultMsg)
	if !ok {
		t.Fatalf("unexpected msg type: %T", msg)
	}
	if len(result.conversations) != 1 {
		t.Fatalf("matches = %d, want 1", len(result.conversations))
	}
	if len(result.indexed) != 0 {
		t.Fatalf("indexed len = %d, want 0", len(result.indexed))
	}
}

func TestDeepSearchCmd_EmptyQueryReturnsMainConversations(t *testing.T) {
	t.Parallel()

	mainConvs := []conversation{
		testConversation("s1", "slug-1"),
		testConversation("s2", "slug-2"),
	}
	msg := deepSearchCmd(context.Background(), "", mainConvs, nil, nil)()
	result, ok := msg.(deepSearchResultMsg)
	if !ok {
		t.Fatalf("unexpected msg type: %T", msg)
	}
	if len(result.conversations) != 2 {
		t.Fatalf("conversations len = %d, want 2", len(result.conversations))
	}
}
