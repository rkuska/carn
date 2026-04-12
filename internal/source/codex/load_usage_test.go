package codex

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestLoadAttachesLastTokenUsageToVisibleAssistantTurn(t *testing.T) {
	t.Parallel()

	session := loadSingleCodexTestSession(t, []map[string]any{
		codexSessionMetaLine("thread-visible-usage"),
		codexUserMessageLine("2026-03-16T10:00:01Z", "Explain the parser."),
		{
			"timestamp": "2026-03-16T10:00:02Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type": responseTypeMessage,
				"role": responseRoleAssistant,
				"content": []map[string]any{
					{"type": "output_text", "text": "Parser updated."},
				},
			},
		},
		codexTokenCountLine(
			"2026-03-16T10:00:03Z",
			map[string]any{
				"input_tokens":            500,
				"cached_input_tokens":     50,
				"output_tokens":           140,
				"reasoning_output_tokens": 10,
			},
			map[string]any{
				"input_tokens":            120,
				"cached_input_tokens":     15,
				"output_tokens":           30,
				"reasoning_output_tokens": 5,
			},
		),
		codexUserMessageLine("2026-03-16T10:00:04Z", "Anything else?"),
	})

	require.Len(t, session.Messages, 3)
	assert.Equal(t, "Parser updated.", session.Messages[1].Text)
	assert.Equal(t, conv.TokenUsage{
		CacheCreationInputTokens: 105,
		CacheReadInputTokens:     15,
		OutputTokens:             25,
		ReasoningOutputTokens:    5,
	}, session.Messages[1].Usage)
	assert.Equal(t, conv.TokenUsage{
		CacheCreationInputTokens: 450,
		CacheReadInputTokens:     50,
		OutputTokens:             130,
		ReasoningOutputTokens:    10,
	}, session.Meta.TotalUsage)
}

func TestLoadAttachesLastTokenUsageToToolOnlyAssistantTurn(t *testing.T) {
	t.Parallel()

	session := loadSingleCodexTestSession(t, []map[string]any{
		codexSessionMetaLine("thread-tool-usage"),
		codexUserMessageLine("2026-03-16T10:00:01Z", "Run the tests."),
		{
			"timestamp": "2026-03-16T10:00:02Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":      responseTypeFunctionCall,
				"name":      "exec_command",
				"arguments": `{"cmd":"go test ./..."}`,
				"call_id":   "call-1",
			},
		},
		{
			"timestamp": "2026-03-16T10:00:03Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":    responseTypeFunctionCallOutput,
				"call_id": "call-1",
				"output":  "ok\n",
				"status":  "completed",
			},
		},
		codexTokenCountLine(
			"2026-03-16T10:00:04Z",
			map[string]any{
				"input_tokens":            300,
				"cached_input_tokens":     30,
				"output_tokens":           80,
				"reasoning_output_tokens": 20,
			},
			map[string]any{
				"input_tokens":            90,
				"cached_input_tokens":     10,
				"output_tokens":           20,
				"reasoning_output_tokens": 5,
			},
		),
		codexUserMessageLine("2026-03-16T10:00:05Z", "Done?"),
	})

	require.Len(t, session.Messages, 3)
	require.Len(t, session.Messages[1].ToolCalls, 1)
	require.Len(t, session.Messages[1].ToolResults, 1)
	assert.Empty(t, session.Messages[1].Text)
	assert.Equal(t, conv.TokenUsage{
		CacheCreationInputTokens: 80,
		CacheReadInputTokens:     10,
		OutputTokens:             15,
		ReasoningOutputTokens:    5,
	}, session.Messages[1].Usage)
}

func TestLoadIgnoresTokenCountWithoutLastTokenUsage(t *testing.T) {
	t.Parallel()

	session := loadSingleCodexTestSession(t, []map[string]any{
		codexSessionMetaLine("thread-missing-last-usage"),
		codexUserMessageLine("2026-03-16T10:00:01Z", "Explain the parser."),
		{
			"timestamp": "2026-03-16T10:00:02Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type": responseTypeMessage,
				"role": responseRoleAssistant,
				"content": []map[string]any{
					{"type": "output_text", "text": "Parser updated."},
				},
			},
		},
		{
			"timestamp": "2026-03-16T10:00:03Z",
			"type":      recordTypeEventMsg,
			"payload": map[string]any{
				"type": eventTypeTokenCount,
				"info": nil,
			},
		},
	})

	require.Len(t, session.Messages, 2)
	assert.Equal(t, conv.TokenUsage{}, session.Messages[1].Usage)
}

func TestLoadDerivesCacheCreationWhenFieldAbsent(t *testing.T) {
	t.Parallel()

	session := loadSingleCodexTestSession(t, []map[string]any{
		codexSessionMetaLine("thread-derive-cache"),
		codexUserMessageLine("2026-03-16T10:00:01Z", "Hello."),
		{
			"timestamp": "2026-03-16T10:00:02Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type": responseTypeMessage,
				"role": responseRoleAssistant,
				"content": []map[string]any{
					{"type": "output_text", "text": "Hi."},
				},
			},
		},
		codexTokenCountLine(
			"2026-03-16T10:00:03Z",
			map[string]any{
				"input_tokens":        200,
				"cached_input_tokens": 60,
				"output_tokens":       40,
			},
			map[string]any{
				"input_tokens":        200,
				"cached_input_tokens": 60,
				"output_tokens":       40,
			},
		),
	})

	require.Len(t, session.Messages, 2)
	assert.Equal(t, conv.TokenUsage{
		CacheCreationInputTokens: 140,
		CacheReadInputTokens:     60,
		OutputTokens:             40,
	}, session.Messages[1].Usage, "non-cached tokens should be derived as cache writes")
	assert.Equal(t, conv.TokenUsage{
		CacheCreationInputTokens: 140,
		CacheReadInputTokens:     60,
		OutputTokens:             40,
	}, session.Meta.TotalUsage)
}

func TestLoadUsesExplicitCacheCreationWhenPresent(t *testing.T) {
	t.Parallel()

	session := loadSingleCodexTestSession(t, []map[string]any{
		codexSessionMetaLine("thread-explicit-cache"),
		codexUserMessageLine("2026-03-16T10:00:01Z", "Hello."),
		{
			"timestamp": "2026-03-16T10:00:02Z",
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type": responseTypeMessage,
				"role": responseRoleAssistant,
				"content": []map[string]any{
					{"type": "output_text", "text": "Hi."},
				},
			},
		},
		codexTokenCountLine(
			"2026-03-16T10:00:03Z",
			map[string]any{
				"input_tokens":                200,
				"cached_input_tokens":         60,
				"cache_creation_input_tokens": 100,
				"output_tokens":               40,
			},
			map[string]any{
				"input_tokens":                200,
				"cached_input_tokens":         60,
				"cache_creation_input_tokens": 100,
				"output_tokens":               40,
			},
		),
	})

	require.Len(t, session.Messages, 2)
	assert.Equal(t, conv.TokenUsage{
		InputTokens:              40,
		CacheCreationInputTokens: 100,
		CacheReadInputTokens:     60,
		OutputTokens:             40,
	}, session.Messages[1].Usage, "explicit cache_creation should be used as-is")
	assert.Equal(t, conv.TokenUsage{
		InputTokens:              40,
		CacheCreationInputTokens: 100,
		CacheReadInputTokens:     60,
		OutputTokens:             40,
	}, session.Meta.TotalUsage)
}

func loadSingleCodexTestSession(tb testing.TB, lines []map[string]any) conv.Session {
	tb.Helper()

	rawDir := tb.TempDir()
	writeCodexRolloutFixture(
		tb,
		rawDir,
		"2026/03/16/rollout-2026-03-16T10-00-00-test.jsonl",
		lines,
	)

	source := New()
	scanResult, err := source.Scan(context.Background(), rawDir)
	require.NoError(tb, err)
	require.Len(tb, scanResult.Conversations, 1)

	session, err := source.Load(context.Background(), scanResult.Conversations[0])
	require.NoError(tb, err)
	return session
}

func codexSessionMetaLine(id string) map[string]any {
	return map[string]any{
		"timestamp": "2026-03-16T10:00:00Z",
		"type":      recordTypeSessionMeta,
		"payload": map[string]any{
			"id":             id,
			"timestamp":      "2026-03-16T10:00:00Z",
			"cwd":            "/workspace/project",
			"cli_version":    "0.114.0",
			"model_provider": "openai",
			"git":            map[string]any{"branch": "main"},
		},
	}
}

func codexUserMessageLine(timestamp, message string) map[string]any {
	return map[string]any{
		"timestamp": timestamp,
		"type":      recordTypeEventMsg,
		"payload": map[string]any{
			"type":    eventTypeUserMessage,
			"message": message,
		},
	}
}

func codexTokenCountLine(
	timestamp string,
	totalUsage map[string]any,
	lastUsage map[string]any,
) map[string]any {
	info := map[string]any{
		"total_token_usage": totalUsage,
	}
	if lastUsage != nil {
		info["last_token_usage"] = lastUsage
	}

	return map[string]any{
		"timestamp": timestamp,
		"type":      recordTypeEventMsg,
		"payload": map[string]any{
			"type": eventTypeTokenCount,
			"info": info,
		},
	}
}
