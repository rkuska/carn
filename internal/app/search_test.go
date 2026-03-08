package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assertContainsAll(t, got, "hello user", "internal thought", "readme.md", "tool output")
}

func TestFindSessionSearchPreview(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{role: roleUser, text: "first prompt"},
			{role: roleAssistant, text: archiveMatchesSourceSubtitle},
		},
	}

	got := findSessionSearchPreview(session, "archive")
	assert.Equal(t, archiveMatchesSourceSubtitle, got)
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

	msg := deepSearchCmd(context.Background(), "needle", 1, mainConvs, nil, nil, sessionCache)()
	result := requireMsgType[deepSearchResultMsg](t, msg)
	require.Len(t, result.conversations, 1)
	assert.Equal(t, "s2", result.conversations[0].id())
	assert.Len(t, result.indexed, 2)
	assert.Equal(t, "contains needle", result.conversations[0].searchPreview)
}

func TestDeepSearchCmd_UsesExistingIndexCache(t *testing.T) {
	t.Parallel()

	mainConvs := []conversation{
		testConversation("s1", "slug-1"),
	}
	indexCache := map[string]string{"s1": "cached needle content"}

	msg := deepSearchCmd(context.Background(), "needle", 1, mainConvs, indexCache, nil, nil)()
	result := requireMsgType[deepSearchResultMsg](t, msg)
	assert.Len(t, result.conversations, 1)
	assert.Empty(t, result.indexed)
}

func TestDeepSearchCmd_EmptyQueryReturnsMainConversations(t *testing.T) {
	t.Parallel()

	mainConvs := []conversation{
		testConversation("s1", "slug-1"),
		testConversation("s2", "slug-2"),
	}
	msg := deepSearchCmd(context.Background(), "", 1, mainConvs, nil, nil, nil)()
	result := requireMsgType[deepSearchResultMsg](t, msg)
	assert.Len(t, result.conversations, 2)
}

func TestDeepSearchCmd_SearchesSubagentContentOnCacheMiss(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	parentID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	parentPath := filepath.Join(dir, parentID+".jsonl")
	parentContent := strings.Join([]string{
		strings.Join([]string{
			`{"type":"user","sessionId":"`, parentID,
			`","slug":"demo","timestamp":"2024-01-01T00:00:00Z","cwd":"/tmp/demo",`,
			`"message":{"role":"user","content":"parent"}}`,
		}, ""),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2024-01-01T00:01:00Z",`,
			`"message":{"role":"assistant","model":"claude-3","content":[`,
			`{"type":"text","text":"parent response"}]}}`,
		}, ""),
	}, "\n")
	require.NoError(t, os.WriteFile(parentPath, []byte(parentContent), 0o644))

	subDir := filepath.Join(dir, parentID, "subagents")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	subContent := strings.Join([]string{
		strings.Join([]string{
			`{"type":"user","sessionId":"sub-session","slug":"demo",`,
			`"timestamp":"2024-01-01T00:02:00Z","cwd":"/tmp/demo",`,
			`"message":{"role":"user","content":"subagent needle"}}`,
		}, ""),
		strings.Join([]string{
			`{"type":"assistant","timestamp":"2024-01-01T00:03:00Z",`,
			`"message":{"role":"assistant","model":"claude-3","content":[`,
			`{"type":"text","text":"done"}]}}`,
		}, ""),
	}, "\n")
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "agent-1.jsonl"), []byte(subContent), 0o644))

	conv := conversation{
		name:    "demo",
		project: project{dirName: "proj", displayName: "proj"},
		sessions: []sessionMeta{
			{
				id:        parentID,
				slug:      "demo",
				filePath:  parentPath,
				timestamp: time.Now(),
				project:   project{dirName: "proj", displayName: "proj"},
			},
		},
	}

	msg := deepSearchCmd(context.Background(), "needle", 1, []conversation{conv}, nil, nil, nil)()
	result := requireMsgType[deepSearchResultMsg](t, msg)
	require.Len(t, result.conversations, 1)
	assert.Equal(t, parentID, result.conversations[0].id())
}
