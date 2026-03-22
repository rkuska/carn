package claude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConversationParsesToolSummariesAndResults(t *testing.T) {
	t.Parallel()

	content := strings.Join([]string{
		makeTestUserRecord(t, "s1", "demo", "inspect"),
		makeTestAssistantToolUseRecord(t, "s1", "toolu_1"),
		makeTestUserToolResultRecord(t, "s1", "demo", "toolu_1", "package main", "done"),
	}, "\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	session, err := parseConversationWithSubagents(context.Background(), conversation{
		Name:    "demo",
		Project: project{DisplayName: "demo"},
		Sessions: []sessionMeta{{
			ID:       "s1",
			Slug:     "demo",
			FilePath: path,
			Project:  project{DisplayName: "demo"},
		}},
	})
	require.NoError(t, err)
	require.Len(t, session.Messages, 3)
	require.Len(t, session.Messages[1].ToolCalls, 1)
	require.Len(t, session.Messages[2].ToolResults, 1)
	assert.Equal(t, "Read", session.Messages[1].ToolCalls[0].Name)
	assert.Equal(t, "/tmp/large.txt", session.Messages[1].ToolCalls[0].Summary)
	assert.Equal(t, "Read", session.Messages[2].ToolResults[0].ToolName)
	assert.Equal(t, "/tmp/large.txt", session.Messages[2].ToolResults[0].ToolSummary)
}

func TestExtractUserContentReturnsToolResultsAndTrailingText(t *testing.T) {
	t.Parallel()

	text, results := extractUserContent([]byte(
		`[` +
			`{"type":"tool_result","tool_use_id":"toolu_1","is_error":true,"content":"command failed"},` +
			`{"type":"text","text":"fix it"}` +
			`]`,
	))

	assert.Equal(t, "fix it", text)
	require.Len(t, results, 1)
	assert.Equal(t, "command failed", results[0].Content)
	assert.True(t, results[0].IsError)
}

func TestExtractStructuredPatchReturnsDiffHunks(t *testing.T) {
	t.Parallel()

	patch := extractStructuredPatch([]byte(`{
		"structuredPatch":[
			{
				"oldStart":3,
				"oldLines":1,
				"newStart":3,
				"newLines":2,
				"lines":["-old line","+new line","+second line"]
			}
		]
	}`))

	require.Len(t, patch, 1)
	assert.Equal(t, 3, patch[0].OldStart)
	assert.Equal(t, 1, patch[0].OldLines)
	assert.Equal(t, 3, patch[0].NewStart)
	assert.Equal(t, 2, patch[0].NewLines)
	assert.Equal(t, []string{"-old line", "+new line", "+second line"}, patch[0].Lines)
}

func TestParseConversationSetsHiddenThinkingForSignedBlocks(t *testing.T) {
	t.Parallel()

	content := strings.Join([]string{
		makeTestUserRecord(t, "s1", "demo", "analyze"),
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "assistant",
			"sessionId": "s1",
			"timestamp": "2024-01-01T00:00:01Z",
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude",
				"content": []map[string]any{
					{"type": "thinking", "thinking": "", "signature": "Ev8DCkYFakeSignature"},
					{"type": "text", "text": "analysis done"},
				},
			},
		}),
	}, "\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	session, err := parseConversationWithSubagents(context.Background(), conversation{
		Name:    "demo",
		Project: project{DisplayName: "demo"},
		Sessions: []sessionMeta{{
			ID:       "s1",
			Slug:     "demo",
			FilePath: path,
			Project:  project{DisplayName: "demo"},
		}},
	})
	require.NoError(t, err)
	require.Len(t, session.Messages, 2)
	assert.True(t, session.Messages[1].HasHiddenThinking)
	assert.Empty(t, session.Messages[1].Thinking)
	assert.Equal(t, "analysis done", session.Messages[1].Text)
}

func TestParseConversationVisibleThinkingWinsOverHidden(t *testing.T) {
	t.Parallel()

	content := strings.Join([]string{
		makeTestUserRecord(t, "s1", "demo", "analyze"),
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "assistant",
			"sessionId": "s1",
			"timestamp": "2024-01-01T00:00:01Z",
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude",
				"content": []map[string]any{
					{"type": "thinking", "thinking": "visible reasoning", "signature": "Ev8DCkYFakeSignature"},
					{"type": "text", "text": "analysis done"},
				},
			},
		}),
	}, "\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	session, err := parseConversationWithSubagents(context.Background(), conversation{
		Name:    "demo",
		Project: project{DisplayName: "demo"},
		Sessions: []sessionMeta{{
			ID:       "s1",
			Slug:     "demo",
			FilePath: path,
			Project:  project{DisplayName: "demo"},
		}},
	})
	require.NoError(t, err)
	require.Len(t, session.Messages, 2)
	assert.False(t, session.Messages[1].HasHiddenThinking)
	assert.Equal(t, "visible reasoning", session.Messages[1].Thinking)
}

func TestParseConversationWithoutLinkedTranscriptsMatchesProjectedParse(t *testing.T) {
	t.Parallel()

	content := strings.Join([]string{
		makeTestUserRecord(t, "s1", "demo", "inspect"),
		makeTestAssistantToolUseRecord(t, "s1", "toolu_1"),
		makeTestUserToolResultRecord(t, "s1", "demo", "toolu_1", "package main", "done"),
	}, "\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	conv := conversation{
		Name:    "demo",
		Project: project{DisplayName: "demo"},
		Sessions: []sessionMeta{{
			ID:       "s1",
			Slug:     "demo",
			FilePath: path,
			Project:  project{DisplayName: "demo"},
		}},
	}

	got, err := parseConversationWithoutLinkedTranscripts(context.Background(), conv)
	require.NoError(t, err)

	parsed, usage, err := parseConversationMessagesDetailed(context.Background(), conv)
	require.NoError(t, err)
	deduplicatePlans(parsed)
	want := sessionFull{
		Meta: sessionMeta{
			ID:         "s1",
			Slug:       "demo",
			Project:    project{DisplayName: "demo"},
			FilePath:   path,
			TotalUsage: usage,
		},
		Messages: messagesFromParsed(parsed),
	}

	assert.Equal(t, want.Meta.TotalUsage, got.Meta.TotalUsage)
	assert.Equal(t, want.Messages, got.Messages)
}

func TestParseConversationProjectedCarriesPerMessageUsage(t *testing.T) {
	t.Parallel()

	content := strings.Join([]string{
		makeTestUserRecord(t, "s1", "demo", "inspect"),
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "assistant",
			"sessionId": "s1",
			"timestamp": "2024-01-01T00:00:01Z",
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude",
				"content": []map[string]any{
					{"type": "text", "text": "done"},
				},
				"usage": map[string]any{
					"input_tokens":  120,
					"output_tokens": 30,
				},
			},
		}),
	}, "\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	got, err := parseConversationWithoutLinkedTranscripts(context.Background(), conversation{
		Name:    "demo",
		Project: project{DisplayName: "demo"},
		Sessions: []sessionMeta{{
			ID:       "s1",
			Slug:     "demo",
			FilePath: path,
			Project:  project{DisplayName: "demo"},
		}},
	})
	require.NoError(t, err)
	require.Len(t, got.Messages, 2)
	assert.Equal(t, tokenUsage{InputTokens: 120, OutputTokens: 30}, got.Messages[1].Usage)
}
