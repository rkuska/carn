package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func benchSessionJSONLLongConversation(
	tb testing.TB,
	sessionID string,
	assistantTurns int,
	includeNeedle bool,
) string {
	tb.Helper()

	needle := ""
	if includeNeedle {
		needle = " IMPORTANT_NEEDLE "
	}
	now := time.Now().UTC()

	lines := make([]map[string]any, 0, 1+assistantTurns*2)
	lines = append(lines, map[string]any{
		"type":      "user",
		"sessionId": sessionID,
		"slug":      "bench",
		"cwd":       "/tmp/bench",
		"gitBranch": "main",
		"version":   "1",
		"timestamp": now.Format(time.RFC3339Nano),
		"message": map[string]any{
			"role":    "user",
			"content": "first message " + needle,
		},
	})

	for i := range assistantTurns {
		ts := now.Add(time.Duration(i+1) * time.Second).Format(time.RFC3339Nano)
		lines = append(lines, map[string]any{
			"type":      "assistant",
			"timestamp": ts,
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude",
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("assistant reply %d%s", i, needle)},
					{
						"type":  "tool_use",
						"id":    fmt.Sprintf("t-%d", i),
						"name":  "Read",
						"input": map[string]any{"file_path": "/tmp/test.go"},
					},
				},
				"usage": map[string]any{
					"input_tokens":                100,
					"output_tokens":               50,
					"cache_read_input_tokens":     10,
					"cache_creation_input_tokens": 5,
				},
			},
		})
		lines = append(lines, map[string]any{
			"type":      "user",
			"sessionId": sessionID,
			"timestamp": now.Add(time.Duration(i+1)*time.Second + 500*time.Millisecond).Format(time.RFC3339Nano),
			"message": map[string]any{
				"role":    "user",
				"content": fmt.Sprintf("followup %d", i),
			},
		})
	}

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

func makeBenchRawCorpus(
	b *testing.B,
	projects, sessionsPerProject, assistantTurns int,
) string {
	b.Helper()

	rawDir := b.TempDir()
	for p := range projects {
		projDir := filepath.Join(rawDir, fmt.Sprintf("project-%d", p))
		if err := os.MkdirAll(projDir, 0o755); err != nil {
			b.Fatalf("os.MkdirAll: %v", err)
		}
		for s := range sessionsPerProject {
			sessionID := fmt.Sprintf("session-%02d-%04d", p, s)
			content := benchSessionJSONLLongConversation(
				b,
				sessionID,
				assistantTurns,
				s%9 == 0,
			)
			path := filepath.Join(projDir, sessionID+".jsonl")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				b.Fatalf("os.WriteFile: %v", err)
			}
		}
	}

	return rawDir
}

func makeBenchConversations(
	b *testing.B,
	projects, sessionsPerProject, assistantTurns int,
) []conversation {
	b.Helper()

	ctx := context.Background()
	rawDir := makeBenchRawCorpus(b, projects, sessionsPerProject, assistantTurns)
	sessions, _, _, err := scanSessions(ctx, rawDir)
	if err != nil {
		b.Fatalf("scanSessions: %v", err)
	}

	return groupConversations(sessions)
}

func BenchmarkCanonicalStoreScanSessions(b *testing.B) {
	ctx := context.Background()
	rawDir := makeBenchRawCorpus(b, 6, 60, 12)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessions, _, _, err := scanSessions(ctx, rawDir)
		if err != nil {
			b.Fatalf("scanSessions: %v", err)
		}
		if len(sessions) == 0 {
			b.Fatal("scanSessions returned no sessions")
		}
	}
}

func BenchmarkCanonicalStoreParseConversationWithSubagents(b *testing.B) {
	ctx := context.Background()
	conversations := makeBenchConversations(b, 6, 60, 12)
	if len(conversations) == 0 {
		b.Fatal("makeBenchConversations returned no conversations")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session, err := parseConversationWithSubagents(ctx, conversations[0])
		if err != nil {
			b.Fatalf("parseConversationWithSubagents: %v", err)
		}
		if len(session.Messages) == 0 {
			b.Fatal("parseConversationWithSubagents returned no messages")
		}
	}
}
