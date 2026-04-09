package claude

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanMetadataCapturesCountsAndUsage(t *testing.T) {
	t.Parallel()

	baseDir := copyScannerFixtureCorpus(t)
	path := filepath.Join(baseDir, "project-a", "session-with-tools.jsonl")

	result, err := scanMetadataResult(context.Background(), path, project{DisplayName: "demo"})
	require.NoError(t, err)
	meta := result.meta
	assert.Equal(t, "session-tools", meta.ID)
	assert.Equal(t, "tool-runbook", meta.Slug)
	assert.Equal(t, "Inspect the main package and run the tests.", meta.FirstMessage)
	assert.Equal(t, "claude-sonnet-4", meta.Model)
	assert.Equal(t, 1, meta.ToolCounts["Read"])
	assert.Equal(t, 1, meta.ToolCounts["Bash"])
	assert.Equal(t, 2, meta.UserMessageCount)
	assert.Equal(t, 3, meta.AssistantMessageCount)
	assert.Nil(t, meta.ToolErrorCounts)
	assert.Equal(t, 440, meta.TotalUsage.TotalTokens())
	assert.Equal(t, 10*time.Second, meta.Duration())
	assert.Equal(t, 5, meta.MessageCount)
	assert.Equal(t, 4, meta.MainMessageCount)
}

func TestScanMetadataCapturesToolErrorCounts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session-errors.jsonl")
	content := strings.Join([]string{
		makeTestUserRecord(t, "s1", "demo", "inspect"),
		makeTestAssistantToolUseRecord(t, "s1", "toolu_1"),
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "user",
			"sessionId": "s1",
			"slug":      "demo",
			"timestamp": "2024-01-01T00:00:02Z",
			"message": map[string]any{
				"role": "user",
				"content": []map[string]any{
					{
						"type":        "tool_result",
						"tool_use_id": "toolu_1",
						"is_error":    true,
						"content":     "read failed",
					},
				},
			},
		}),
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	result, err := scanMetadataResult(context.Background(), path, project{DisplayName: "demo"})
	require.NoError(t, err)

	assert.Equal(t, map[string]int{"Read": 1}, result.meta.ToolCounts)
	assert.Equal(t, map[string]int{"Read": 1}, result.meta.ToolErrorCounts)
}

func TestScanMetadataCapturesToolRejectCounts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session-rejects.jsonl")
	content := strings.Join([]string{
		makeTestUserRecord(t, "s1", "demo", "inspect"),
		makeTestAssistantToolUseRecord(t, "s1", "toolu_1"),
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "user",
			"sessionId": "s1",
			"slug":      "demo",
			"timestamp": "2024-01-01T00:00:02Z",
			"message": map[string]any{
				"role": "user",
				"content": []map[string]any{
					{
						"type":        "tool_result",
						"tool_use_id": "toolu_1",
						"is_error":    true,
						"content":     "The tool use was rejected by the user.",
					},
				},
			},
		}),
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	result, err := scanMetadataResult(context.Background(), path, project{DisplayName: "demo"})
	require.NoError(t, err)

	assert.Equal(t, map[string]int{"Read": 1}, result.meta.ToolCounts)
	assert.Nil(t, result.meta.ToolErrorCounts)
	assert.Equal(t, map[string]int{"Read": 1}, result.meta.ToolRejectCounts)
}

func TestJSONLLinesHandlesLargeLines(t *testing.T) {
	t.Parallel()

	reader := strings.NewReader(strings.Repeat("x", jsonlMetadataBufferSize+32) + "\n")
	lines, err := collectJSONLLines(jsonlLines(bufio.NewReaderSize(reader, 32)))
	require.NoError(t, err)
	require.Len(t, lines, 1)
	assert.Len(t, lines[0], jsonlMetadataBufferSize+32)
}

func TestScanMetadataHandlesLargeAssistantContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	largeText := strings.Repeat("assistant ", jsonlMetadataBufferSize/4)
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
					{"type": "text", "text": largeText},
				},
				"usage": map[string]any{
					"input_tokens":  100,
					"output_tokens": 50,
				},
			},
		}),
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	result, err := scanMetadataResult(context.Background(), path, project{DisplayName: "demo"})
	require.NoError(t, err)
	assert.Equal(t, "s1", result.meta.ID)
	assert.Equal(t, "inspect", result.meta.FirstMessage)
	assert.Equal(t, "claude", result.meta.Model)
	assert.Equal(t, 2, result.meta.MessageCount)
}

func TestAssistantSignedThinkingCountsAsConversationContent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		raw  string
		want bool
	}{
		{
			name: "visible thinking",
			raw:  `[{"type":"thinking","thinking":"reasoning here"}]`,
			want: true,
		},
		{
			name: "signed empty thinking",
			raw:  `[{"type":"thinking","thinking":"","signature":"Ev8DCkYFakeSignature"}]`,
			want: true,
		},
		{
			name: "empty thinking without signature",
			raw:  `[{"type":"thinking","thinking":""}]`,
			want: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, assistantContentHasConversationContent([]byte(testCase.raw)))
		})
	}
}

func TestExtractType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		line string
		want string
	}{
		{
			name: "user record",
			line: `{"type":"user","message":{"role":"user","content":"hi"}}`,
			want: "user",
		},
		{
			name: "assistant record",
			line: `{"type":"assistant","message":{"role":"assistant","content":"hi"}}`,
			want: "assistant",
		},
		{
			name: "system record",
			line: `{"type":"system","content":"turn finished"}`,
			want: "system",
		},
		{
			name: "other record",
			line: `{"type":"summary","message":{"role":"assistant","content":"hi"}}`,
			want: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, extractType([]byte(testCase.line)))
		})
	}
}
