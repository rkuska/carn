package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rkuska/carn/internal/canonical"
	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
	"github.com/rkuska/carn/internal/source/claude"
)

func benchBrowserSessionJSONL(
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

func makeBenchBrowserArchive(
	b *testing.B,
	projects, sessionsPerProject, assistantTurns int,
) string {
	b.Helper()

	archiveDir := b.TempDir()
	rawDir := src.ProviderRawDir(archiveDir, conv.ProviderClaude)

	for p := range projects {
		projDir := filepath.Join(rawDir, fmt.Sprintf("project-%d", p))
		if err := os.MkdirAll(projDir, 0o755); err != nil {
			b.Fatalf("os.MkdirAll: %v", err)
		}
		for s := range sessionsPerProject {
			sessionID := fmt.Sprintf("session-%02d-%04d", p, s)
			content := benchBrowserSessionJSONL(
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

	return archiveDir
}

func rebuildBenchBrowserStore(b *testing.B, archiveDir string) *canonical.Store {
	b.Helper()

	store := canonical.New(claude.New())
	if _, err := store.RebuildAll(context.Background(), archiveDir, nil); err != nil {
		b.Fatalf("store.RebuildAll: %v", err)
	}
	return store
}

func mustLoadBenchBrowserConversations(
	tb testing.TB,
	ctx context.Context,
	archiveDir string,
	store browserStore,
) []conv.Conversation {
	tb.Helper()

	return mustBenchLoadedConversations(tb, loadSessionsCmdWithStore(ctx, archiveDir, store)())
}

func mustBenchLoadedConversations(tb testing.TB, msg any) []conv.Conversation {
	tb.Helper()

	loaded, ok := msg.(conversationsLoadedMsg)
	if !ok {
		tb.Fatalf("unexpected message type %T", msg)
	}
	if len(loaded.conversations) == 0 {
		tb.Fatal("loadSessionsCmdWithStore returned no conversations")
	}
	return loaded.conversations
}

func mustBenchOpenViewerMsg(tb testing.TB, msg any) openViewerMsg {
	tb.Helper()

	opened, ok := msg.(openViewerMsg)
	if !ok {
		tb.Fatalf("unexpected message type %T", msg)
	}
	if len(opened.session.Messages) == 0 {
		tb.Fatal("openConversationCmdWithStore returned no messages")
	}
	return opened
}

func mustBenchDeepSearchResult(tb testing.TB, msg any) deepSearchResultMsg {
	tb.Helper()

	result, ok := msg.(deepSearchResultMsg)
	if !ok {
		tb.Fatalf("unexpected message type %T", msg)
	}
	if result.err != nil {
		tb.Fatalf("deepSearchRepositoryCmd: %v", result.err)
	}
	if len(result.conversations) == 0 {
		tb.Fatal("deepSearchRepositoryCmd returned no conversations")
	}
	return result
}

func BenchmarkBrowserLoadSessionsCold(b *testing.B) {
	ctx := context.Background()
	archiveDir := makeBenchBrowserArchive(b, 6, 60, 12)
	rebuildBenchBrowserStore(b, archiveDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := canonical.New()
		msg := loadSessionsCmdWithStore(ctx, archiveDir, newBrowserStore(store))()
		loaded, ok := msg.(conversationsLoadedMsg)
		if !ok {
			b.Fatalf("unexpected message type %T", msg)
		}
		if len(loaded.conversations) == 0 {
			b.Fatal("loadSessionsCmdWithStore returned no conversations")
		}
	}
}

func BenchmarkBrowserLoadSessionsWarm(b *testing.B) {
	ctx := context.Background()
	archiveDir := makeBenchBrowserArchive(b, 6, 60, 12)
	store := rebuildBenchBrowserStore(b, archiveDir)
	cmd := loadSessionsCmdWithStore(ctx, archiveDir, newBrowserStore(store))
	_ = mustBenchLoadedConversations(b, cmd())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mustBenchLoadedConversations(b, cmd())
	}
}

func BenchmarkBrowserOpenConversationWarm(b *testing.B) {
	ctx := context.Background()
	archiveDir := makeBenchBrowserArchive(b, 4, 50, 12)
	store := rebuildBenchBrowserStore(b, archiveDir)
	browserStore := newBrowserStore(store)
	conversations := mustLoadBenchBrowserConversations(b, ctx, archiveDir, browserStore)
	cmd := openConversationCmdWithStore(ctx, archiveDir, conversations[0], browserStore)
	_ = mustBenchOpenViewerMsg(b, cmd())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mustBenchOpenViewerMsg(b, cmd())
	}
}

func BenchmarkBrowserDeepSearchWarm(b *testing.B) {
	ctx := context.Background()
	archiveDir := makeBenchBrowserArchive(b, 4, 50, 12)
	store := rebuildBenchBrowserStore(b, archiveDir)
	browserStore := newBrowserStore(store)
	conversations := mustLoadBenchBrowserConversations(b, ctx, archiveDir, browserStore)
	cmd := deepSearchRepositoryCmd(
		ctx,
		archiveDir,
		"important needle",
		1,
		conversations,
		browserStore,
	)
	_ = mustBenchDeepSearchResult(b, cmd())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mustBenchDeepSearchResult(b, cmd())
	}
}
