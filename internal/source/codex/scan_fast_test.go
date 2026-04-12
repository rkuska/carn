package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
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
	assert.Empty(t, state.meta.Model)
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
	assert.Equal(t, 0, state.meta.UserMessageCount)
	assert.Equal(t, 1, state.meta.AssistantMessageCount)
	assert.Equal(t, "Parser updated.", state.lastText)
}

func TestScanRolloutLineParsesSubagentSource(t *testing.T) {
	t.Parallel()

	line := []byte(
		`{"timestamp":"2026-03-16T10:00:00Z","type":"session_meta","payload":{` +
			`"id":"thread-child","timestamp":"2026-03-16T10:00:00Z",` +
			`"cwd":"/workspace/project","cli_version":"0.114.0",` +
			`"model_provider":"openai","source":{"subagent":{"thread_spawn":{` +
			`"parent_thread_id":"thread-main","agent_nickname":"worker","agent_role":"worker"}}}}}`,
	)

	state := newScanState("/tmp/thread.jsonl")
	require.NoError(t, scanRolloutLine(line, &state))

	assert.True(t, state.meta.IsSubagent)
	assert.Equal(t, "thread-main", state.link.parentThreadID)
	assert.Equal(t, "worker", state.link.agentNickname)
	assert.Equal(t, "worker", state.link.agentRole)
}

func TestScanRolloutLineTracksToolCountsAndTokenUsage(t *testing.T) {
	t.Parallel()

	state := newScanState("/tmp/thread.jsonl")

	require.NoError(t, scanRolloutLine([]byte(
		`{"timestamp":"2026-03-16T10:00:03Z","type":"response_item","payload":{`+
			`"type":"function_call","name":"exec_command","arguments":"{}","call_id":"call-1"}}`,
	), &state))
	require.NoError(t, scanRolloutLine([]byte(
		`{"timestamp":"2026-03-16T10:00:04Z","type":"event_msg","payload":{`+
			`"type":"token_count","info":{"total_token_usage":{`+
			`"input_tokens":100,"cached_input_tokens":10,"output_tokens":50,"reasoning_output_tokens":5}}}}`,
	), &state))

	require.NotNil(t, state.meta.ToolCounts)
	assert.Equal(t, 1, state.meta.ToolCounts["exec_command"])
	assert.Equal(t, 0, state.meta.TotalUsage.InputTokens)
	assert.Equal(t, 90, state.meta.TotalUsage.CacheCreationInputTokens)
	assert.Equal(t, 10, state.meta.TotalUsage.CacheReadInputTokens)
	assert.Equal(t, 45, state.meta.TotalUsage.OutputTokens)
	assert.Equal(t, 5, state.meta.TotalUsage.ReasoningOutputTokens)
}

func TestScanRolloutLineTracksStoredTokenCountShape(t *testing.T) {
	t.Parallel()

	state := newScanState("/tmp/thread.jsonl")

	require.NoError(t, scanRolloutLine([]byte(
		`{"timestamp":"2026-03-16T10:00:04Z","type":"event_msg","payload":{`+
			`"type":"token_count","info":{"total_token_usage":{`+
			`"input_tokens":85420,"cached_input_tokens":62150,`+
			`"output_tokens":12340,"reasoning_output_tokens":4870,"total_tokens":97760},`+
			`"last_token_usage":{"input_tokens":85420,"cached_input_tokens":62150,`+
			`"output_tokens":12340,"reasoning_output_tokens":4870,"total_tokens":97760}}}}`,
	), &state))

	assert.Equal(t, conv.TokenUsage{
		CacheCreationInputTokens: 23270,
		CacheReadInputTokens:     62150,
		OutputTokens:             7470,
		ReasoningOutputTokens:    4870,
	}, state.meta.TotalUsage)
}

func TestScanRolloutLineTracksToolErrorCounts(t *testing.T) {
	t.Parallel()

	state := newScanState("/tmp/thread.jsonl")

	require.NoError(t, scanRolloutLine([]byte(
		`{"timestamp":"2026-03-16T10:00:03Z","type":"response_item","payload":{`+
			`"type":"custom_tool_call","name":"apply_patch","input":"*** Begin Patch","call_id":"call-1"}}`,
	), &state))
	require.NoError(t, scanRolloutLine([]byte(
		`{"timestamp":"2026-03-16T10:00:04Z","type":"response_item","payload":{`+
			`"type":"custom_tool_call_output","call_id":"call-1",`+
			`"output":"apply_patch verification failed: failed to find expected lines",`+
			`"status":"completed"}}`,
	), &state))

	assert.Equal(t, map[string]int{"apply_patch": 1}, state.meta.ToolCounts)
	assert.Equal(t, map[string]int{"apply_patch": 1}, state.meta.ToolErrorCounts)
}

func TestScanRolloutLineTracksToolRejectCounts(t *testing.T) {
	t.Parallel()

	state := newScanState("/tmp/thread.jsonl")

	require.NoError(t, scanRolloutLine([]byte(
		`{"timestamp":"2026-03-16T10:00:03Z","type":"response_item","payload":{`+
			`"type":"function_call","name":"exec_command","arguments":"{}","call_id":"call-1"}}`,
	), &state))
	require.NoError(t, scanRolloutLine([]byte(
		`{"timestamp":"2026-03-16T10:00:04Z","type":"response_item","payload":{`+
			`"type":"function_call_output","call_id":"call-1",`+
			`"output":"The user doesn't want to proceed with this tool use.","status":"failed"}}`,
	), &state))

	assert.Equal(t, map[string]int{"exec_command": 1}, state.meta.ToolCounts)
	assert.Nil(t, state.meta.ToolErrorCounts)
	assert.Equal(t, map[string]int{"exec_command": 1}, state.meta.ToolRejectCounts)
}

func TestScanRolloutLineTracksToolErrorCountsFromFailedStatusWithoutStringOutput(t *testing.T) {
	t.Parallel()

	state := newScanState("/tmp/thread.jsonl")

	require.NoError(t, scanRolloutLine([]byte(
		`{"timestamp":"2026-03-16T10:00:03Z","type":"response_item","payload":{`+
			`"type":"function_call","name":"exec_command","arguments":"{}","call_id":"call-1"}}`,
	), &state))
	require.NoError(t, scanRolloutLine([]byte(
		`{"timestamp":"2026-03-16T10:00:04Z","type":"response_item","payload":{`+
			`"type":"function_call_output","call_id":"call-1","output":{"code":1},"status":"failed"}}`,
	), &state))

	assert.Equal(t, map[string]int{"exec_command": 1}, state.meta.ToolCounts)
	assert.Equal(t, map[string]int{"exec_command": 1}, state.meta.ToolErrorCounts)
}

func TestScanRolloutLineIgnoresVerificationFailedTextInExecCommandOutput(t *testing.T) {
	t.Parallel()

	state := newScanState("/tmp/thread.jsonl")

	require.NoError(t, scanRolloutLine([]byte(
		`{"timestamp":"2026-03-16T10:00:03Z","type":"response_item","payload":{`+
			`"type":"function_call","name":"exec_command","arguments":"{}","call_id":"call-1"}}`,
	), &state))
	require.NoError(t, scanRolloutLine([]byte(
		`{"timestamp":"2026-03-16T10:00:04Z","type":"response_item","payload":{`+
			`"type":"function_call_output","call_id":"call-1",`+
			`"output":"func isCodexToolError(output string) bool { `+
			`return strings.Contains(strings.ToLower(output), \"verification failed\") }",`+
			`"status":"completed"}}`,
	), &state))

	assert.Equal(t, map[string]int{"exec_command": 1}, state.meta.ToolCounts)
	assert.Nil(t, state.meta.ToolErrorCounts)
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
	assert.Empty(t, rollout.meta.Model)
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
