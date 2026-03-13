package archive

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
	"github.com/rkuska/carn/internal/source/claude"
)

func benchSessionJSONL(tb testing.TB, sessionID string, includeNeedle bool) string {
	tb.Helper()

	needle := ""
	if includeNeedle {
		needle = " IMPORTANT_NEEDLE "
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	lines := []map[string]any{
		{
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
		},
		{
			"type":      "assistant",
			"timestamp": now,
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude",
				"content": []map[string]any{
					{"type": "text", "text": "assistant reply " + needle},
				},
				"usage": map[string]any{
					"input_tokens":  100,
					"output_tokens": 50,
				},
			},
		},
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

func setupBenchProjectDirs(b *testing.B, source string, projects, sessions int) {
	b.Helper()

	for p := range projects {
		projDir := filepath.Join(source, fmt.Sprintf("project-%d", p))
		if err := os.MkdirAll(projDir, 0o755); err != nil {
			b.Fatalf("os.MkdirAll: %v", err)
		}
		for s := range sessions {
			sessionID := fmt.Sprintf("session-%02d-%04d", p, s)
			content := benchSessionJSONL(b, sessionID, false)
			path := filepath.Join(projDir, sessionID+".jsonl")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				b.Fatalf("os.WriteFile: %v", err)
			}
		}
	}
}

func BenchmarkCollectFilesToSync(b *testing.B) {
	dir := b.TempDir()
	sourceDir := filepath.Join(dir, "source")
	archiveDir := filepath.Join(dir, "archive")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		b.Fatalf("os.MkdirAll: %v", err)
	}

	for i := range 400 {
		path := filepath.Join(sourceDir, "proj", fmt.Sprintf("s-%04d.jsonl", i))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			b.Fatalf("os.MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, []byte(benchSessionJSONL(b, fmt.Sprintf("s-%d", i), false)), 0o644); err != nil {
			b.Fatalf("os.WriteFile: %v", err)
		}
	}

	cfg := syncRootsConfig{
		sourceDir: sourceDir,
		destDir:   providerRawDir(archiveDir, conv.ProviderClaude),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := collectSyncCandidates(cfg)
		if err != nil {
			b.Fatalf("collectSyncCandidates: %v", err)
		}
		if len(files) == 0 {
			b.Fatal("collectSyncCandidates returned no files")
		}
	}
}

func BenchmarkStreamImportAnalysis(b *testing.B) {
	dir := b.TempDir()
	sourceDir := filepath.Join(dir, "source")
	archiveDir := filepath.Join(dir, "archive")
	setupBenchProjectDirs(b, sourceDir, 6, 60)

	source := claude.New()
	store := canonical.New(source)
	pipeline := New(
		Config{
			SourceDir:  sourceDir,
			ArchiveDir: archiveDir,
		},
		store,
		source,
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analysis, err := pipeline.Analyze(context.Background(), nil)
		if err != nil {
			b.Fatalf("pipeline.Analyze: %v", err)
		}
		if analysis.Conversations == 0 {
			b.Fatal("pipeline.Analyze returned no conversations")
		}
	}
}
