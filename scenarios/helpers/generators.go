package helpers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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

// SeedFixtureCorpus copies the shared raw-session corpus into the workspace source.
func (w Workspace) SeedFixtureCorpus(tb testing.TB) {
	tb.Helper()
	copyFixtureDir(tb, fixtureCorpusDir(tb), w.SourceDir)
}

// SeedCodexFixtureCorpus copies the shared Codex raw-session corpus into an isolated source directory.
func (w Workspace) SeedCodexFixtureCorpus(tb testing.TB) string {
	tb.Helper()

	sourceDir := filepath.Join(w.RootDir, "codex-source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		tb.Fatalf("os.MkdirAll codex source: %v", err)
	}
	copyFixtureDir(tb, codexFixtureCorpusDir(tb), sourceDir)
	return sourceDir
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

// WriteRawSession writes raw JSONL content into the source workspace.
func (w Workspace) WriteRawSession(tb testing.TB, project, fileName, content string) string {
	tb.Helper()

	if project == "" {
		project = "project-a"
	}
	if fileName == "" {
		fileName = "session-1.jsonl"
	}

	projectDir := filepath.Join(w.SourceDir, project)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		tb.Fatalf("os.MkdirAll project: %v", err)
	}

	path := filepath.Join(projectDir, fileName)
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

// MustJSONForScenario marshals a value for scenario JSONL fixtures.
func MustJSONForScenario(tb testing.TB, value any) string {
	tb.Helper()
	return mustJSON(tb, value)
}

func fixtureCorpusDir(tb testing.TB) string {
	tb.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("runtime.Caller: no caller information")
	}

	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "claude_raw")
}

func codexFixtureCorpusDir(tb testing.TB) string {
	tb.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("runtime.Caller: no caller information")
	}

	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "codex_raw")
}

func copyFixtureDir(tb testing.TB, srcDir, dstDir string) {
	tb.Helper()

	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		dstPath := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0o644)
	})
	if err != nil {
		tb.Fatalf("copyFixtureDir: %v", err)
	}
}
