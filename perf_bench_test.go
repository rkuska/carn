package main

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

func makeBenchArchive(b *testing.B, projects, sessionsPerProject int) string {
	b.Helper()
	base := b.TempDir()

	for p := range projects {
		projDir := filepath.Join(base, fmt.Sprintf("project-%d", p))
		if err := os.MkdirAll(projDir, 0o755); err != nil {
			b.Fatalf("MkdirAll: %v", err)
		}
		for s := range sessionsPerProject {
			sessionID := fmt.Sprintf("session-%02d-%04d", p, s)
			content := benchSessionJSONL(b, sessionID, s%9 == 0)
			path := filepath.Join(projDir, sessionID+".jsonl")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				b.Fatalf("WriteFile: %v", err)
			}
		}
	}

	return base
}

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

func BenchmarkScanSessions(b *testing.B) {
	ctx := context.Background()
	archive := makeBenchArchive(b, 6, 60)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessions, err := scanSessions(ctx, archive)
		if err != nil {
			b.Fatalf("scanSessions: %v", err)
		}
		if len(sessions) == 0 {
			b.Fatal("scanSessions returned no sessions")
		}
	}
}

func BenchmarkDeepSearch(b *testing.B) {
	ctx := context.Background()
	archive := makeBenchArchive(b, 4, 50)
	sessions, err := scanSessions(ctx, archive)
	if err != nil {
		b.Fatalf("scanSessions: %v", err)
	}
	mainConvs := filterMainConversations(groupConversations(sessions))

	b.Run("cold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			msg := deepSearchCmd(ctx, "important_needle", mainConvs, nil, nil)()
			result, ok := msg.(deepSearchResultMsg)
			if !ok || len(result.conversations) == 0 {
				b.Fatalf("unexpected deep search result: %#v", msg)
			}
		}
	})

	warm := deepSearchCmd(ctx, "important_needle", mainConvs, nil, nil)().(deepSearchResultMsg)
	index := warm.indexed

	b.Run("warm", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			msg := deepSearchCmd(ctx, "important_needle", mainConvs, index, nil)()
			result, ok := msg.(deepSearchResultMsg)
			if !ok || len(result.conversations) == 0 {
				b.Fatalf("unexpected deep search result: %#v", msg)
			}
		}
	})
}

func BenchmarkViewerRenderContent(b *testing.B) {
	session := benchViewerSession(180, true)
	m := newViewerModel(session, "dark", 140, 45)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.renderContent()
	}
}

func BenchmarkViewerSearch(b *testing.B) {
	session := benchViewerSession(180, true)
	m := newViewerModel(session, "dark", 140, 45)
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

func BenchmarkStreamImportAnalysis(b *testing.B) {
	dir := b.TempDir()
	source := filepath.Join(dir, "source")
	archive := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archive, 0o755); err != nil {
		b.Fatalf("MkdirAll: %v", err)
	}

	// Create 6 projects × 60 sessions
	for p := range 6 {
		projDir := filepath.Join(source, fmt.Sprintf("project-%d", p))
		if err := os.MkdirAll(projDir, 0o755); err != nil {
			b.Fatalf("MkdirAll: %v", err)
		}
		for s := range 60 {
			sessionID := fmt.Sprintf("session-%02d-%04d", p, s)
			content := benchSessionJSONL(b, sessionID, false)
			path := filepath.Join(projDir, sessionID+".jsonl")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				b.Fatalf("WriteFile: %v", err)
			}
		}
	}

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
