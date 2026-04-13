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

func TestSourceLoadConversationBundleMatchesSeparateLoadsWithoutLinkedTranscripts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	source := New()
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project-a")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	firstPath := filepath.Join(projectDir, "session-1.jsonl")
	secondPath := filepath.Join(projectDir, "session-2.jsonl")
	require.NoError(t, os.WriteFile(firstPath, []byte(strings.Join([]string{
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "user",
			"sessionId": "s1",
			"slug":      "demo",
			"timestamp": "2024-01-01T00:00:00Z",
			"cwd":       "/tmp",
			"message": map[string]any{
				"role":    "user",
				"content": "start",
			},
		}),
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "assistant",
			"sessionId": "s1",
			"timestamp": "2024-01-01T00:00:01Z",
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude",
				"content": []map[string]any{
					{"type": "text", "text": "first reply"},
				},
				"usage": map[string]any{
					"input_tokens":  10,
					"output_tokens": 5,
				},
			},
		}),
	}, "\n")), 0o644))
	require.NoError(t, os.WriteFile(secondPath, []byte(strings.Join([]string{
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "user",
			"sessionId": "s2",
			"slug":      "demo",
			"timestamp": "2024-01-01T01:00:00Z",
			"cwd":       "/tmp",
			"message": map[string]any{
				"role":    "user",
				"content": "continue",
			},
		}),
		marshalTestJSONLRecord(t, map[string]any{
			"type":      "assistant",
			"sessionId": "s2",
			"timestamp": "2024-01-01T01:00:01Z",
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude",
				"content": []map[string]any{
					{"type": "text", "text": "second reply"},
				},
				"usage": map[string]any{
					"input_tokens":  20,
					"output_tokens": 7,
				},
			},
		}),
	}, "\n")), 0o644))

	scanResult, err := source.Scan(ctx, dir)
	require.NoError(t, err)
	require.Len(t, scanResult.Conversations, 1)

	conv := scanResult.Conversations[0]
	full, err := source.Load(ctx, conv)
	require.NoError(t, err)

	bundledFull, bundledSessions, err := source.LoadConversationBundle(ctx, conv)
	require.NoError(t, err)
	assert.Equal(t, full, bundledFull)
	require.Len(t, bundledSessions, len(conv.Sessions))

	for i, meta := range conv.Sessions {
		want, err := source.LoadSession(ctx, conv, meta)
		require.NoError(t, err)
		assert.Equal(t, want, bundledSessions[i])
	}
}
