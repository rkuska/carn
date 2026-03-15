package canonical

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
	src "github.com/rkuska/carn/internal/source"
	"github.com/rkuska/carn/internal/source/claude"
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

func makeBenchRawArchive(
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

	return archiveDir
}

func makeBenchConversations(
	b *testing.B,
	projects, sessionsPerProject, assistantTurns int,
) (string, []conversation, Source) {
	b.Helper()

	ctx := context.Background()
	source := claude.New()
	archiveDir := makeBenchRawArchive(b, projects, sessionsPerProject, assistantTurns)
	rawDir := src.ProviderRawDir(archiveDir, conv.ProviderClaude)

	conversations, err := source.Scan(ctx, rawDir)
	if err != nil {
		b.Fatalf("source.Scan: %v", err)
	}

	return archiveDir, conversations, source
}

func makeBenchCanonicalStore(
	b *testing.B,
	projects, sessionsPerProject, assistantTurns int,
) (string, *Store) {
	b.Helper()

	source := claude.New()
	store := New(source)
	archiveDir := makeBenchRawArchive(b, projects, sessionsPerProject, assistantTurns)

	if err := store.RebuildAll(context.Background(), archiveDir, nil); err != nil {
		b.Fatalf("store.RebuildAll: %v", err)
	}

	return archiveDir, store
}

func BenchmarkLoadCatalogCold(b *testing.B) {
	ctx := context.Background()
	archiveDir, store := makeBenchCanonicalStore(b, 6, 60, 12)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := store.invalidateDB(archiveDir); err != nil {
			b.Fatalf("store.invalidateDB: %v", err)
		}
		conversations, err := store.List(ctx, archiveDir)
		if err != nil {
			b.Fatalf("store.List: %v", err)
		}
		if len(conversations) == 0 {
			b.Fatal("store.List returned no conversations")
		}
	}
}

func BenchmarkLoadCatalogWarm(b *testing.B) {
	ctx := context.Background()
	archiveDir, store := makeBenchCanonicalStore(b, 6, 60, 12)

	conversations, err := store.List(ctx, archiveDir)
	if err != nil {
		b.Fatalf("store.List: %v", err)
	}
	if len(conversations) == 0 {
		b.Fatal("store.List returned no conversations")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conversations, err := store.List(ctx, archiveDir)
		if err != nil {
			b.Fatalf("store.List: %v", err)
		}
		if len(conversations) == 0 {
			b.Fatal("store.List returned no conversations")
		}
	}
}

func BenchmarkLoadSearchIndex(b *testing.B) {
	ctx := context.Background()
	archiveDir, store := makeBenchCanonicalStore(b, 6, 60, 12)
	db, err := store.loadDB(ctx, archiveDir)
	if err != nil {
		b.Fatalf("store.loadDB: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM search_chunks`).Scan(&count); err != nil {
			b.Fatalf("db.QueryRowContext: %v", err)
		}
		if count == 0 {
			b.Fatal("search_chunks returned no rows")
		}
	}
}

func BenchmarkDeepSearchFuzzy(b *testing.B) {
	ctx := context.Background()
	archiveDir, store := makeBenchCanonicalStore(b, 4, 50, 12)
	conversations, err := store.List(ctx, archiveDir)
	if err != nil {
		b.Fatalf("store.List: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, available, err := store.DeepSearch(ctx, archiveDir, "important needle", conversations)
		if err != nil {
			b.Fatalf("store.DeepSearch: %v", err)
		}
		if !available || len(results) == 0 {
			b.Fatalf("unexpected deep search result: available=%v len=%d", available, len(results))
		}
	}
}

func BenchmarkCanonicalTranscriptOpen(b *testing.B) {
	ctx := context.Background()
	archiveDir, store := makeBenchCanonicalStore(b, 4, 50, 12)
	conversations, err := store.List(ctx, archiveDir)
	if err != nil {
		b.Fatalf("store.List: %v", err)
	}
	if len(conversations) == 0 {
		b.Fatal("store.List returned no conversations")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session, err := store.Load(ctx, archiveDir, conversations[0])
		if err != nil {
			b.Fatalf("store.Load: %v", err)
		}
		if len(session.Messages) == 0 {
			b.Fatal("store.Load returned no messages")
		}
	}
}

func BenchmarkCanonicalStoreFullRebuild(b *testing.B) {
	archiveDir := makeBenchRawArchive(b, 6, 60, 12)
	store := New(claude.New())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := store.RebuildAll(context.Background(), archiveDir, nil); err != nil {
			b.Fatalf("store.RebuildAll: %v", err)
		}
	}
}

func BenchmarkCanonicalStoreIncrementalRebuild(b *testing.B) {
	archiveDir, store := makeBenchCanonicalStore(b, 6, 60, 12)
	rawDir := src.ProviderRawDir(archiveDir, conv.ProviderClaude)
	changedPaths := []string{
		filepath.Join(rawDir, "project-0", "session-00-0000.jsonl"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := store.RebuildAll(
			context.Background(),
			archiveDir,
			map[conv.Provider][]string{conv.ProviderClaude: changedPaths},
		); err != nil {
			b.Fatalf("store.RebuildAll: %v", err)
		}
	}
}

func BenchmarkCanonicalStoreParseConversations(b *testing.B) {
	ctx := context.Background()
	_, conversations, source := makeBenchConversations(b, 6, 60, 12)
	if len(conversations) == 0 {
		b.Fatal("makeBenchConversations returned no conversations")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transcripts, corpus, err := parseConversationsParallel(ctx, source, conversations)
		if err != nil {
			b.Fatalf("parseConversationsParallel: %v", err)
		}
		if len(transcripts) != len(conversations) {
			b.Fatalf("unexpected transcript count: got %d want %d", len(transcripts), len(conversations))
		}
		if corpus.Len() == 0 {
			b.Fatal("parseConversationsParallel returned no search units")
		}
	}
}
