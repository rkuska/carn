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
)

func benchSessionJSONL(tb testing.TB, sessionID string, includeNeedle bool) string {
	tb.Helper()

	needle := ""
	if includeNeedle {
		needle = " IMPORTANT_NEEDLE "
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	userMsg := map[string]any{
		"type":      "user",
		"sessionId": sessionID,
		"slug":      "bench",
		"cwd":       "/tmp/bench",
		"gitBranch": "main",
		"version":   "1",
		"timestamp": now,
		"message": map[string]any{
			"role":    "user",
			"content": "first message " + needle,
		},
	}
	assistantMsg := map[string]any{
		"type":      "assistant",
		"timestamp": now,
		"message": map[string]any{
			"role":  "assistant",
			"model": "claude",
			"content": []map[string]any{
				{"type": "text", "text": "assistant reply " + needle},
				{"type": "tool_use", "id": "t1", "name": "Read", "input": map[string]any{"file_path": "/tmp/test.go"}},
			},
			"usage": map[string]any{
				"input_tokens":                100,
				"output_tokens":               50,
				"cache_read_input_tokens":     10,
				"cache_creation_input_tokens": 5,
			},
		},
	}
	userToolMsg := map[string]any{
		"type":      "user",
		"sessionId": sessionID,
		"timestamp": now,
		"message": map[string]any{
			"role": "user",
			"content": []map[string]any{
				{
					"type":        "tool_result",
					"tool_use_id": "t1",
					"content":     "tool output " + needle,
				},
			},
		},
	}

	lines := []map[string]any{userMsg, assistantMsg, userToolMsg}
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

func benchViewerSession(messages int, withNeedle bool) sessionFull {
	msgs := make([]message, 0, messages*2)
	for i := range messages {
		msgs = append(msgs, message{role: roleUser, text: fmt.Sprintf("user line %d", i)})
		text := fmt.Sprintf("assistant line %d", i)
		if withNeedle && i%25 == 0 {
			text += " IMPORTANT_NEEDLE"
		}
		msgs = append(msgs, message{role: roleAssistant, text: text})
	}

	return sessionFull{
		meta: sessionMeta{
			id:        "bench-viewer",
			timestamp: time.Now(),
			project:   project{displayName: "bench/project"},
		},
		messages: msgs,
	}
}

func BenchmarkLoadCatalog(b *testing.B) {
	ctx := context.Background()
	archiveDir := makeBenchCanonicalStore(b, 6, 60, 12)
	repo := newDefaultConversationRepository()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conversations, err := repo.scan(ctx, archiveDir)
		if err != nil {
			b.Fatalf("repo.scan: %v", err)
		}
		if len(conversations) == 0 {
			b.Fatal("repo.scan returned no conversations")
		}
	}
}

func BenchmarkLoadSearchIndex(b *testing.B) {
	ctx := context.Background()
	archiveDir := makeBenchCanonicalStore(b, 6, 60, 12)
	repo := newDefaultConversationRepository()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		corpus, err := repo.searchCorpus(ctx, archiveDir)
		if err != nil {
			b.Fatalf("repo.searchCorpus: %v", err)
		}
		if corpus.Len() == 0 {
			b.Fatal("repo.searchCorpus returned no search units")
		}
	}
}

func BenchmarkDeepSearchFuzzy(b *testing.B) {
	ctx := context.Background()
	archiveDir := makeBenchCanonicalStore(b, 4, 50, 12)
	repo := newDefaultConversationRepository()
	conversations, err := repo.scan(ctx, archiveDir)
	if err != nil {
		b.Fatalf("repo.scan: %v", err)
	}
	corpus, err := repo.searchCorpus(ctx, archiveDir)
	if err != nil {
		b.Fatalf("repo.searchCorpus: %v", err)
	}
	mainConvs := filterMainConversations(conversations)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := deepSearchCmd(ctx, "important_needle", 1, mainConvs, corpus)()
		result, ok := msg.(deepSearchResultMsg)
		if !ok || len(result.conversations) == 0 {
			b.Fatalf("unexpected deep search result: %#v", msg)
		}
	}
}

func BenchmarkCanonicalTranscriptOpen(b *testing.B) {
	ctx := context.Background()
	archiveDir := makeBenchCanonicalStore(b, 4, 50, 12)
	repo := newDefaultConversationRepository()
	conversations, err := repo.scan(ctx, archiveDir)
	if err != nil {
		b.Fatalf("repo.scan: %v", err)
	}
	if len(conversations) == 0 {
		b.Fatal("repo.scan returned no conversations")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session, err := repo.load(ctx, archiveDir, conversations[0])
		if err != nil {
			b.Fatalf("repo.load: %v", err)
		}
		if len(session.messages) == 0 {
			b.Fatal("repo.load returned no messages")
		}
	}
}

func BenchmarkViewerRenderContent(b *testing.B) {
	session := benchViewerSession(180, true)
	m := newViewerModel(session, singleSessionConversation(session.meta), "dark", 140, 45)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.renderContent()
	}
}

func BenchmarkViewerSearch(b *testing.B) {
	session := benchViewerSession(180, true)
	m := newViewerModel(session, singleSessionConversation(session.meta), "dark", 140, 45)
	m.searchQuery = "important_needle"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.performSearch()
	}
}

func BenchmarkCollectFilesToSync(b *testing.B) {
	dir := b.TempDir()
	source := filepath.Join(dir, "source")
	archive := filepath.Join(dir, "archive")
	if err := os.MkdirAll(source, 0o755); err != nil {
		b.Fatalf("MkdirAll: %v", err)
	}
	if err := os.MkdirAll(archive, 0o755); err != nil {
		b.Fatalf("MkdirAll: %v", err)
	}

	for i := range 400 {
		path := filepath.Join(source, "proj", fmt.Sprintf("s-%04d.jsonl", i))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			b.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, []byte(benchSessionJSONL(b, fmt.Sprintf("s-%d", i), false)), 0o644); err != nil {
			b.Fatalf("WriteFile: %v", err)
		}
	}

	cfg := archiveConfig{sourceDir: source, archiveDir: archive}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := collectFilesToSync(cfg)
		if err != nil {
			b.Fatalf("collectFilesToSync: %v", err)
		}
		if len(files) == 0 {
			b.Fatal("collectFilesToSync returned no files")
		}
	}
}

func setupBenchProjectDirs(b *testing.B, source string, projects, sessions int) {
	b.Helper()
	for p := range projects {
		projDir := filepath.Join(source, fmt.Sprintf("project-%d", p))
		if err := os.MkdirAll(projDir, 0o755); err != nil {
			b.Fatalf("MkdirAll: %v", err)
		}
		for s := range sessions {
			sessionID := fmt.Sprintf("session-%02d-%04d", p, s)
			content := benchSessionJSONL(b, sessionID, false)
			path := filepath.Join(projDir, sessionID+".jsonl")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				b.Fatalf("WriteFile: %v", err)
			}
		}
	}
}

func BenchmarkStreamImportAnalysis(b *testing.B) {
	dir := b.TempDir()
	source := filepath.Join(dir, "source")
	archive := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archive, 0o755); err != nil {
		b.Fatalf("MkdirAll: %v", err)
	}

	setupBenchProjectDirs(b, source, 6, 60)

	cfg := archiveConfig{sourceDir: source, archiveDir: archive}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dirs, err := listProjectDirs(source)
		if err != nil {
			b.Fatalf("listProjectDirs: %v", err)
		}

		seen := make(map[groupKey]*conversationState)
		var syncCandidates []string
		var totalInspected int

		for _, projDir := range dirs {
			inspected, err := analyzeProjectDir(projDir, cfg, seen, &syncCandidates)
			if err != nil {
				b.Fatalf("analyzeProjectDir: %v", err)
			}
			totalInspected += inspected
		}

		if totalInspected == 0 {
			b.Fatal("no files inspected")
		}
		if len(seen) == 0 {
			b.Fatal("no conversations found")
		}
	}
}

func BenchmarkCanonicalStoreIncrementalRebuild(b *testing.B) {
	archiveDir := makeBenchCanonicalStore(b, 6, 60, 12)
	rawDir := providerRawDir(archiveDir, conversationProviderClaude)

	changedPaths := []string{
		filepath.Join(rawDir, "project-0", "session-00-0000.jsonl"),
		filepath.Join(rawDir, "project-1", "session-01-0001.jsonl"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := rebuildCanonicalStore(
			context.Background(), archiveDir, conversationProviderClaude, changedPaths,
		); err != nil {
			b.Fatalf("rebuildCanonicalStore: %v", err)
		}
	}
}

func makeBenchCanonicalStore(
	b *testing.B,
	projects, sessionsPerProject, assistantTurns int,
) string {
	b.Helper()

	archiveDir := b.TempDir()
	rawDir := providerRawDir(archiveDir, conversationProviderClaude)

	for p := range projects {
		projDir := filepath.Join(rawDir, fmt.Sprintf("project-%d", p))
		if err := os.MkdirAll(projDir, 0o755); err != nil {
			b.Fatalf("MkdirAll: %v", err)
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
				b.Fatalf("WriteFile: %v", err)
			}
		}
	}

	if err := rebuildCanonicalStore(context.Background(), archiveDir, conversationProviderClaude, nil); err != nil {
		b.Fatalf("rebuildCanonicalStore: %v", err)
	}

	return archiveDir
}
