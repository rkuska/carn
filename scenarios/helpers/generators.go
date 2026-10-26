package helpers

import (
	"encoding/json"
	"fmt"
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

// GenerateSessionSpecs creates deterministic session specs spread across
// five projects and roughly ninety days for tests and benchmarks.
func GenerateSessionSpecs(count int) []SessionSpec {
	if count <= 0 {
		return nil
	}

	projects := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	modelTimespan := 90 * 24 * time.Hour
	step := time.Duration(0)
	if count > 1 {
		step = modelTimespan / time.Duration(count-1)
	}

	specs := make([]SessionSpec, 0, count)
	start := scenarioTimestamp.Add(-modelTimespan)
	for i := range count {
		sessionID := fmt.Sprintf("bench-session-%05d", i)
		specs = append(specs, SessionSpec{
			Project:       projects[i%len(projects)],
			FileName:      sessionID + ".jsonl",
			Slug:          sessionID,
			SessionID:     sessionID,
			UserText:      fmt.Sprintf("benchmark question %d", i),
			AssistantText: fmt.Sprintf("benchmark response %d", i),
			Timestamp:     start.Add(step * time.Duration(i)),
		})
	}
	return specs
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

// contextGrowthRequest describes one model request of the context growth
// session: the prompt size it reports and the tool it calls, if any.
type contextGrowthRequest struct {
	prompt   int
	text     string
	toolName string
	toolArg  string
}

var contextGrowthRequests = []contextGrowthRequest{
	{prompt: 24_000, text: "Reading the parser first.", toolName: "Read", toolArg: "internal/parser.go"},
	{prompt: 27_000, text: "Running the suite.", toolName: "Bash", toolArg: "go test ./internal/parser/..."},
	{prompt: 71_000, text: "Searching for the failing case.", toolName: "Grep", toolArg: "ErrUnexpected"},
	{prompt: 74_000, text: "The parser rejects empty input."},
	{prompt: 76_000, text: "Checking the fix.", toolName: "Read", toolArg: "internal/parser_test.go"},
	{prompt: 20_000, text: "Context was compacted."},
	{prompt: 23_000, text: "Adding the missing test.", toolName: "Edit", toolArg: "internal/parser_test.go"},
	{prompt: 25_000, text: "Tests pass now."},
}

// WriteContextGrowthSession writes a synthetic Claude session whose context
// grows with every request, jumps by 44k after one tool result and drops back
// after a compaction. Every request is split over a thinking, a text and (for
// tool calls) a tool_use record that repeat the same message id and usage, the
// way Claude writes them.
func (w Workspace) WriteContextGrowthSession(tb testing.TB) string {
	tb.Helper()

	base := scenarioTimestamp
	var lines []string
	record := func(kind string, offset int, message map[string]any) {
		lines = append(lines, mustJSON(tb, map[string]any{
			"type":      kind,
			"sessionId": "context-growth",
			"slug":      "context-growth",
			"cwd":       "/workspace/project",
			"timestamp": base.Add(time.Duration(offset) * time.Second).Format(time.RFC3339Nano),
			"message":   message,
		}))
	}

	record("user", 0, map[string]any{"role": "user", "content": "Fix the failing parser test."})
	for i, request := range contextGrowthRequests {
		offset := 10 + i*10
		id := fmt.Sprintf("msg_context_%02d", i)
		usage := map[string]any{
			"input_tokens":                3,
			"cache_creation_input_tokens": 500,
			"cache_read_input_tokens":     request.prompt - 503,
			"output_tokens":               200,
		}
		assistant := func(block map[string]any) {
			record("assistant", offset, map[string]any{
				"role": "assistant", "model": "claude-sonnet-4", "id": id,
				"content": []map[string]any{block}, "usage": usage,
			})
		}
		assistant(map[string]any{"type": "thinking", "thinking": "Step " + id, "signature": "sig"})
		assistant(map[string]any{"type": "text", "text": request.text})
		if request.toolName == "" {
			continue
		}
		toolID := fmt.Sprintf("toolu_context_%02d", i)
		assistant(map[string]any{
			"type": "tool_use", "id": toolID, "name": request.toolName,
			"input": map[string]any{"file_path": request.toolArg, "command": request.toolArg, "pattern": request.toolArg},
		})
		record("user", offset+1, map[string]any{
			"role": "user",
			"content": []map[string]any{{
				"type": "tool_result", "tool_use_id": toolID, "content": "output of " + request.toolName,
			}},
		})
	}
	return w.WriteRawSession(tb, "project-a", "session-context-growth.jsonl", strings.Join(lines, "\n")+"\n")
}
