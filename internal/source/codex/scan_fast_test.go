package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanRolloutLineAppliesSessionMetaPayload(t *testing.T) {
	t.Parallel()

	line := []byte(
		`{"timestamp":"2026-03-16T10:00:00Z","type":"session_meta","payload":{` +
			`"id":"thread-string-summary","timestamp":"2026-03-16T10:00:00Z",` +
			`"cwd":"/workspace/project","cli_version":"0.114.0",` +
			`"model_provider":"openai","git":{"branch":"main"}}}`,
	)

	state := newScanState("/tmp/thread.jsonl")
	require.NoError(t, scanRolloutLine(line, &state))

	assert.Equal(t, "thread-string-summary", state.meta.ID)
	assert.Equal(t, "thread-strin", state.meta.Slug)
	assert.Equal(t, "/workspace/project", state.meta.CWD)
	assert.Equal(t, "0.114.0", state.meta.Version)
	assert.Equal(t, "openai", state.meta.Model)
	assert.Equal(t, "main", state.meta.GitBranch)
}

func TestScanRolloutLineCountsVisibleResponseMessage(t *testing.T) {
	t.Parallel()

	line := []byte(
		`{"timestamp":"2026-03-16T10:00:03Z","type":"response_item","payload":{` +
			`"type":"message","role":"assistant","content":[` +
			`{"type":"output_text","text":"Parser updated."}]}}`,
	)

	state := newScanState("/tmp/thread.jsonl")
	require.NoError(t, scanRolloutLine(line, &state))

	assert.Equal(t, 1, state.meta.MessageCount)
	assert.Equal(t, 1, state.meta.MainMessageCount)
	assert.Equal(t, "Parser updated.", state.lastText)
}

func TestScanRolloutParsesSingleFile(t *testing.T) {
	t.Parallel()

	rawDir := t.TempDir()
	path := filepath.Join(rawDir, "thread.jsonl")
	data := []byte(
		`{"timestamp":"2026-03-16T10:00:00Z","type":"session_meta","payload":{` +
			`"id":"thread-string-summary","timestamp":"2026-03-16T10:00:00Z",` +
			`"cwd":"/workspace/project","cli_version":"0.114.0",` +
			`"model_provider":"openai","git":{"branch":"main"}}}` +
			"\n" +
			`{"timestamp":"2026-03-16T10:00:01Z","type":"event_msg","payload":{` +
			`"type":"user_message","message":"Explain the parser."}}`,
	)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	rollout, ok, err := scanRollout(path)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "thread-string-summary", rollout.meta.ID)
	assert.Equal(t, 1, rollout.meta.MessageCount)
	assert.Equal(t, "Explain the parser.", rollout.meta.FirstMessage)
}

func TestScanRolloutParsesStringSummaryFile(t *testing.T) {
	t.Parallel()

	rawDir := t.TempDir()
	writeCodexRolloutFixture(t, rawDir, "thread-string-summary.jsonl", []map[string]any{
		{
			"timestamp": "2026-03-16T10:00:00Z",
			"type":      recordTypeSessionMeta,
			"payload": map[string]any{
				"id":             "thread-string-summary",
				"timestamp":      "2026-03-16T10:00:00Z",
				"cwd":            "/workspace/project",
				"cli_version":    "0.114.0",
				"model_provider": "openai",
				"git":            map[string]any{"branch": "main"},
			},
		},
		{
			"timestamp": "2026-03-16T10:00:01Z",
			"type":      recordTypeEventMsg,
			"payload": map[string]any{
				"type":    eventTypeUserMessage,
				"message": "Explain the parser.",
			},
		},
		{
			"timestamp": "2026-03-16T10:00:02Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":    responseTypeReasoning,
				"summary": "Inspecting rollout schema drift.",
			},
		},
		{
			"timestamp": "2026-03-16T10:00:03Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type": responseTypeMessage,
				"role": responseRoleAssistant,
				"content": []map[string]any{
					{"type": "output_text", "text": "Parser updated."},
				},
			},
		},
	})

	rollout, ok, err := scanRollout(filepath.Join(rawDir, "thread-string-summary.jsonl"))
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "thread-string-summary", rollout.meta.ID)
	assert.Equal(t, 2, rollout.meta.MessageCount)
	assert.Equal(t, "Explain the parser.", rollout.meta.FirstMessage)
}
