package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

func benchRolloutJSONL(
	tb testing.TB,
	threadID string,
	assistantTurns int,
	includeNeedle bool,
	parentThreadID string,
) string {
	tb.Helper()

	needle := ""
	if includeNeedle {
		needle = " IMPORTANT_NEEDLE "
	}
	now := time.Now().UTC()

	lines := make([]map[string]any, 0, 4+assistantTurns*6)

	var source any = "cli"
	if parentThreadID != "" {
		source = map[string]any{
			"subagent": map[string]any{
				"thread_spawn": map[string]any{
					"parent_thread_id": parentThreadID,
					"agent_nickname":   "worker",
					"agent_role":       "worker",
				},
			},
		}
	}

	lines = append(lines, map[string]any{
		"timestamp": now.Format(time.RFC3339Nano),
		"type":      recordTypeSessionMeta,
		"payload": map[string]any{
			"id":             threadID,
			"timestamp":      now.Format(time.RFC3339Nano),
			"cwd":            "/tmp/bench",
			"originator":     "codex_cli_rs",
			"cli_version":    "0.114.0",
			"source":         source,
			"model_provider": "openai",
			"git":            map[string]any{"branch": "main"},
		},
	})

	lines = append(lines, map[string]any{
		"timestamp": now.Add(time.Millisecond).Format(time.RFC3339Nano),
		"type":      recordTypeTurnContext,
		"payload": map[string]any{
			"cwd":   "/tmp/bench",
			"model": "gpt-5.4",
		},
	})

	lines = append(lines, map[string]any{
		"timestamp": now.Add(2 * time.Millisecond).Format(time.RFC3339Nano),
		"type":      recordTypeEventMsg,
		"payload": map[string]any{
			"type":    eventTypeUserMessage,
			"message": "first message " + needle,
		},
	})

	for i := range assistantTurns {
		base := now.Add(time.Duration(i+1) * time.Second)
		callID := fmt.Sprintf("call_%s_%d", threadID, i)

		lines = append(lines, map[string]any{
			"timestamp": base.Format(time.RFC3339Nano),
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":    responseTypeReasoning,
				"summary": []map[string]any{{"type": "summary_text", "text": fmt.Sprintf("reasoning %d", i)}},
			},
		})

		lines = append(lines, map[string]any{
			"timestamp": base.Add(100 * time.Millisecond).Format(time.RFC3339Nano),
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":      responseTypeFunctionCall,
				"name":      "exec_command",
				"arguments": fmt.Sprintf(`{"cmd":"echo %d","workdir":"/tmp/bench"}`, i),
				"call_id":   callID,
			},
		})

		lines = append(lines, map[string]any{
			"timestamp": base.Add(200 * time.Millisecond).Format(time.RFC3339Nano),
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":    responseTypeFunctionCallOutput,
				"call_id": callID,
				"output":  fmt.Sprintf("Exit code: 0\nOutput: %d\n", i),
			},
		})

		lines = append(lines, map[string]any{
			"timestamp": base.Add(300 * time.Millisecond).Format(time.RFC3339Nano),
			"type":      recordTypeEventMsg,
			"payload": map[string]any{
				"type":    eventTypeAgentMessage,
				"phase":   "commentary",
				"message": fmt.Sprintf("assistant reply %d%s", i, needle),
			},
		})

		lines = append(lines, map[string]any{
			"timestamp": base.Add(400 * time.Millisecond).Format(time.RFC3339Nano),
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type": responseTypeMessage,
				"role": responseRoleAssistant,
				"content": []map[string]any{
					{"type": "output_text", "text": fmt.Sprintf("assistant reply %d%s", i, needle)},
				},
			},
		})

		lines = append(lines, map[string]any{
			"timestamp": base.Add(500 * time.Millisecond).Format(time.RFC3339Nano),
			"type":      recordTypeEventMsg,
			"payload": map[string]any{
				"type":    eventTypeUserMessage,
				"message": fmt.Sprintf("followup %d", i),
			},
		})
	}

	lines = append(lines, map[string]any{
		"timestamp": now.Add(time.Duration(assistantTurns+1) * time.Second).Format(time.RFC3339Nano),
		"type":      recordTypeEventMsg,
		"payload": map[string]any{
			"type": eventTypeTokenCount,
			"info": map[string]any{
				"total_token_usage": map[string]any{
					"input_tokens":            100,
					"cached_input_tokens":     10,
					"output_tokens":           50,
					"reasoning_output_tokens": 5,
				},
			},
		},
	})

	encoded := make([]string, 0, len(lines))
	for _, line := range lines {
		raw, err := json.Marshal(line)
		if err != nil {
			tb.Fatalf("json.Marshal: %v", err)
		}
		encoded = append(encoded, string(raw))
	}
	return strings.Join(encoded, "\n")
}

func makeBenchRawCodexCorpus(
	b *testing.B,
	rolloutCount, assistantTurns, subagentRatio int,
) string {
	b.Helper()

	rawDir := b.TempDir()
	dateDir := filepath.Join(rawDir, "2026", "01", "01")
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		b.Fatalf("os.MkdirAll: %v", err)
	}

	lastRootID := ""
	for i := range rolloutCount {
		threadID := fmt.Sprintf("bench-thread-%06d", i)
		parentThreadID := ""
		if subagentRatio > 0 && i > 0 && i%subagentRatio == 0 && lastRootID != "" {
			parentThreadID = lastRootID
		} else {
			lastRootID = threadID
		}

		content := benchRolloutJSONL(b, threadID, assistantTurns, i%9 == 0, parentThreadID)
		filename := fmt.Sprintf("rollout-2026-01-01T00-00-00-%s.jsonl", threadID)
		path := filepath.Join(dateDir, filename)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			b.Fatalf("os.WriteFile: %v", err)
		}
	}
	return rawDir
}

func makeBenchCodexConversations(
	b *testing.B,
	rolloutCount, assistantTurns, subagentRatio int,
) []conv.Conversation {
	b.Helper()

	rawDir := makeBenchRawCodexCorpus(b, rolloutCount, assistantTurns, subagentRatio)
	conversations, _, err := scanRollouts(context.Background(), rawDir)
	if err != nil {
		b.Fatalf("scanRollouts: %v", err)
	}
	return conversations
}

func BenchmarkScanRollouts(b *testing.B) {
	ctx := context.Background()
	rawDir := makeBenchRawCodexCorpus(b, 360, 12, 6)

	for b.Loop() {
		conversations, _, err := scanRollouts(ctx, rawDir)
		if err != nil {
			b.Fatalf("scanRollouts: %v", err)
		}
		if len(conversations) == 0 {
			b.Fatal("scanRollouts returned no conversations")
		}
	}
}

func BenchmarkLoadConversation(b *testing.B) {
	ctx := context.Background()
	conversations := makeBenchCodexConversations(b, 360, 12, 6)
	if len(conversations) == 0 {
		b.Fatal("makeBenchCodexConversations returned no conversations")
	}

	var target conv.Conversation
	for _, c := range conversations {
		if len(c.Sessions) > 1 {
			target = c
			break
		}
	}
	if len(target.Sessions) == 0 {
		b.Fatal("no conversation with subagent sessions found")
	}

	for b.Loop() {
		session, err := loadConversation(ctx, target)
		if err != nil {
			b.Fatalf("loadConversation: %v", err)
		}
		if len(session.Messages) == 0 {
			b.Fatal("loadConversation returned no messages")
		}
	}
}
