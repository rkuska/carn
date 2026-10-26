package claude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

// requestUsage is the raw usage object repeated on every record of one API
// response.
type requestUsage struct {
	input, creation, read, output int
}

func (u requestUsage) raw() map[string]any {
	return map[string]any{
		"input_tokens":                u.input,
		"cache_creation_input_tokens": u.creation,
		"cache_read_input_tokens":     u.read,
		"output_tokens":               u.output,
	}
}

func (u requestUsage) tokenUsage() conv.TokenUsage {
	return conv.TokenUsage{
		InputTokens:              u.input,
		CacheCreationInputTokens: u.creation,
		CacheReadInputTokens:     u.read,
		OutputTokens:             u.output,
	}
}

func textBlock(text string) map[string]any {
	return map[string]any{"type": "text", "text": text}
}

func toolUseBlock(id string) map[string]any {
	return map[string]any{
		"type":  "tool_use",
		"id":    id,
		"name":  "Read",
		"input": map[string]any{"file_path": "/workspace/main.go"},
	}
}

func thinkingBlock(text string) map[string]any {
	return map[string]any{"type": "thinking", "thinking": text, "signature": "sig"}
}

// assistantRecordLine renders one assistant record carrying a single content
// block, the way Claude splits an API response across records.
func assistantRecordLine(t testing.TB, id string, sidechain bool, block map[string]any, usage requestUsage) string {
	t.Helper()

	message := map[string]any{
		"role":    "assistant",
		"model":   "claude-sonnet-4",
		"content": []map[string]any{block},
		"usage":   usage.raw(),
	}
	if id != "" {
		message["id"] = id
	}
	return marshalTestJSONLRecord(t, map[string]any{
		"type":        "assistant",
		"sessionId":   "session-request",
		"slug":        "request-usage",
		"isSidechain": sidechain,
		"timestamp":   "2026-03-08T08:00:01Z",
		"message":     message,
	})
}

func writeRequestSession(t *testing.T, lines []string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "session-request.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644))
	return path
}

func loadRequestMessages(t *testing.T, path string) ([]message, tokenUsage) {
	t.Helper()

	var pc parseContext
	messages, usage, err := parseSessionProjectedWithContext(context.Background(), path, &pc)
	require.NoError(t, err)
	return messages, usage
}

func scanRequestUsage(t *testing.T, path string) tokenUsage {
	t.Helper()

	result, err := scanMetadataResult(context.Background(), path, project{DisplayName: "demo"})
	require.NoError(t, err)
	return result.meta.TotalUsage
}

func messageUsages(messages []message) []conv.TokenUsage {
	usages := make([]conv.TokenUsage, len(messages))
	for i, msg := range messages {
		usages[i] = msg.Usage
	}
	return usages
}

func TestRequestUsageIsCountedOncePerMessageID(t *testing.T) {
	t.Parallel()

	first := requestUsage{input: 3, creation: 100, read: 900, output: 50}
	second := requestUsage{input: 4, creation: 20, read: 1100, output: 70}

	testCases := []struct {
		name       string
		lines      []string
		wantUsages []conv.TokenUsage
		wantTotal  conv.TokenUsage
	}{
		{
			name: "split response keeps usage on the first record",
			lines: []string{
				makeTestUserRecord(t, "session-request", "request-usage", "hello"),
				assistantRecordLine(t, "msg_a", false, thinkingBlock("plan"), first),
				assistantRecordLine(t, "msg_a", false, textBlock("answer"), first),
				assistantRecordLine(t, "msg_a", false, toolUseBlock("toolu_1"), first),
				assistantRecordLine(t, "msg_b", false, textBlock("done"), second),
			},
			wantUsages: []conv.TokenUsage{
				{},
				first.tokenUsage(),
				{},
				{},
				second.tokenUsage(),
			},
			wantTotal: conv.TokenUsage{
				InputTokens:              7,
				CacheCreationInputTokens: 120,
				CacheReadInputTokens:     2000,
				OutputTokens:             120,
			},
		},
		{
			name: "records without a message id each keep their usage",
			lines: []string{
				makeTestUserRecord(t, "session-request", "request-usage", "hello"),
				assistantRecordLine(t, "", false, textBlock("one"), first),
				assistantRecordLine(t, "", false, textBlock("two"), first),
			},
			wantUsages: []conv.TokenUsage{{}, first.tokenUsage(), first.tokenUsage()},
			wantTotal: conv.TokenUsage{
				InputTokens:              6,
				CacheCreationInputTokens: 200,
				CacheReadInputTokens:     1800,
				OutputTokens:             100,
			},
		},
		{
			name: "sidechain records of another id do not split a main response",
			lines: []string{
				makeTestUserRecord(t, "session-request", "request-usage", "hello"),
				assistantRecordLine(t, "msg_a", false, textBlock("main one"), first),
				assistantRecordLine(t, "msg_s", true, textBlock("side"), second),
				assistantRecordLine(t, "msg_a", false, toolUseBlock("toolu_1"), first),
			},
			wantUsages: []conv.TokenUsage{{}, first.tokenUsage(), second.tokenUsage(), {}},
			wantTotal: conv.TokenUsage{
				InputTokens:              7,
				CacheCreationInputTokens: 120,
				CacheReadInputTokens:     2000,
				OutputTokens:             120,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := writeRequestSession(t, tc.lines)
			messages, loadTotal := loadRequestMessages(t, path)

			assert.Equal(t, tc.wantUsages, messageUsages(messages))
			assert.Equal(t, tc.wantTotal, loadTotal)
			assert.Equal(t, tc.wantTotal, scanRequestUsage(t, path))
		})
	}
}

func TestRequestUsageMovesToNextKeptRecordWhenFirstIsDropped(t *testing.T) {
	t.Parallel()

	usage := requestUsage{input: 3, creation: 100, read: 900, output: 50}
	path := writeRequestSession(t, []string{
		makeTestUserRecord(t, "session-request", "request-usage", "hello"),
		assistantRecordLine(t, "msg_a", false, textBlock(""), usage),
		assistantRecordLine(t, "msg_a", false, textBlock("answer"), usage),
	})

	messages, loadTotal := loadRequestMessages(t, path)

	require.Len(t, messages, 2)
	assert.Equal(t, usage.tokenUsage(), messages[1].Usage)
	assert.Equal(t, usage.tokenUsage(), loadTotal)
	assert.Equal(t, usage.tokenUsage(), scanRequestUsage(t, path))
}

func TestRequestUsageNullMessageIDKeepsTheSessionReadable(t *testing.T) {
	t.Parallel()

	usage := requestUsage{input: 3, creation: 100, read: 900, output: 50}
	path := writeRequestSession(t, []string{
		makeTestUserRecord(t, "session-request", "request-usage", "hello"),
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "assistant",
			"sessionId": "session-request",
			"slug":      "request-usage",
			"timestamp": "2026-03-08T08:00:01Z",
			"message": map[string]any{
				"id":      nil,
				"role":    "assistant",
				"model":   "claude-sonnet-4",
				"content": []map[string]any{textBlock("answer")},
				"usage":   usage.raw(),
			},
		}),
	})

	messages, loadTotal := loadRequestMessages(t, path)

	require.Len(t, messages, 2)
	assert.Equal(t, usage.tokenUsage(), messages[1].Usage)
	assert.Equal(t, usage.tokenUsage(), loadTotal)
	assert.Equal(t, usage.tokenUsage(), scanRequestUsage(t, path))
}

func TestRequestUsageSpansConversationParts(t *testing.T) {
	t.Parallel()

	usage := requestUsage{input: 3, creation: 100, read: 900, output: 50}
	lines := []string{
		makeTestUserRecord(t, "session-request", "request-usage", "hello"),
		assistantRecordLine(t, "msg_a", false, textBlock("one"), usage),
		assistantRecordLine(t, "msg_a", false, textBlock("two"), usage),
	}
	pathA := writeRequestSession(t, lines)
	pathB := writeRequestSession(t, lines)

	var pc parseContext
	for _, path := range []string{pathA, pathB} {
		messages, total, err := parseSessionProjectedWithContext(context.Background(), path, &pc)
		require.NoError(t, err)
		assert.Equal(t, usage.tokenUsage(), total, "a reused parse context must not leak the last id")
		assert.Equal(t, usage.tokenUsage(), messages[1].Usage)
	}
}
