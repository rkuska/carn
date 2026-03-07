package helpers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var scenarioTimestamp = time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)

// Workspace contains isolated source and archive directories for a scenario.
type Workspace struct {
	RootDir    string
	SourceDir  string
	ArchiveDir string
}

// SessionSpec describes a minimal Claude session fixture for a scenario.
type SessionSpec struct {
	Project       string
	FileName      string
	Slug          string
	SessionID     string
	UserText      string
	AssistantText string
	Timestamp     time.Time
}

// NewWorkspace creates an isolated source/archive layout for a scenario.
func NewWorkspace(tb testing.TB) Workspace {
	tb.Helper()

	rootDir := tb.TempDir()
	sourceDir := filepath.Join(rootDir, "source")
	archiveDir := filepath.Join(rootDir, "archive")

	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		tb.Fatalf("os.MkdirAll source: %v", err)
	}
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		tb.Fatalf("os.MkdirAll archive: %v", err)
	}

	return Workspace{
		RootDir:    rootDir,
		SourceDir:  sourceDir,
		ArchiveDir: archiveDir,
	}
}

// WriteSession writes a minimal session JSONL file into the source workspace.
func (w Workspace) WriteSession(tb testing.TB, spec SessionSpec) string {
	tb.Helper()

	spec = withSessionDefaults(spec)

	projectDir := filepath.Join(w.SourceDir, spec.Project)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		tb.Fatalf("os.MkdirAll project: %v", err)
	}

	path := filepath.Join(projectDir, spec.FileName)
	content := strings.Join([]string{
		mustJSON(tb, map[string]any{
			"type":      "user",
			"sessionId": spec.SessionID,
			"slug":      spec.Slug,
			"timestamp": spec.Timestamp.Format(time.RFC3339Nano),
			"message": map[string]any{
				"role":    "user",
				"content": spec.UserText,
			},
		}),
		mustJSON(tb, map[string]any{
			"type":      "assistant",
			"sessionId": spec.SessionID,
			"slug":      spec.Slug,
			"timestamp": spec.Timestamp.Add(time.Second).Format(time.RFC3339Nano),
			"message": map[string]any{
				"role":  "assistant",
				"model": "claude-opus-4-1",
				"content": []map[string]any{
					{"type": "text", "text": spec.AssistantText},
				},
			},
		}),
	}, "\n")

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		tb.Fatalf("os.WriteFile: %v", err)
	}

	return path
}

func withSessionDefaults(spec SessionSpec) SessionSpec {
	if spec.Project == "" {
		spec.Project = "project-a"
	}
	if spec.FileName == "" {
		spec.FileName = "session-1.jsonl"
	}
	if spec.Slug == "" {
		spec.Slug = "sample-session"
	}
	if spec.SessionID == "" {
		spec.SessionID = "session-1"
	}
	if spec.UserText == "" {
		spec.UserText = "Test session question"
	}
	if spec.AssistantText == "" {
		spec.AssistantText = "Assistant response for transcript"
	}
	if spec.Timestamp.IsZero() {
		spec.Timestamp = scenarioTimestamp
	}
	return spec
}

func mustJSON(tb testing.TB, value any) string {
	tb.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		tb.Fatalf("json.Marshal: %v", err)
	}

	return string(raw)
}
