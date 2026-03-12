package claude

import (
	"encoding/json"
	"iter"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

const testToolRead = "Read"

func fixtureCorpusDir(tb testing.TB) string {
	tb.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	return filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "claude_raw")
}

func copyFixtureCorpusToSource(tb testing.TB, sourceDir string) {
	tb.Helper()
	copyFixtureDir(tb, fixtureCorpusDir(tb), sourceDir)
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
	require.NoError(tb, err)
}

func copyScannerFixtureCorpus(t *testing.T) string {
	t.Helper()

	baseDir := t.TempDir()
	copyFixtureCorpusToSource(t, baseDir)
	return baseDir
}

func collectJSONLLines(seq iter.Seq2[[]byte, error]) ([]string, error) {
	lines := make([]string, 0)
	for line, err := range seq {
		if err != nil {
			return lines, err
		}
		lines = append(lines, string(append([]byte(nil), line...)))
	}
	return lines, nil
}

func makeTestUserRecord(t testing.TB, sessionID, slug, text string) string {
	t.Helper()

	return marshalTestJSONLRecord(t, map[string]any{
		"type":      "user",
		"sessionId": sessionID,
		"slug":      slug,
		"timestamp": "2024-01-01T00:00:00Z",
		"cwd":       "/tmp",
		"message": map[string]any{
			"role":    "user",
			"content": text,
		},
	})
}

func makeTestUserToolResultRecord(
	t testing.TB,
	sessionID, slug, toolUseID, toolContent, text string,
) string {
	t.Helper()

	return marshalTestJSONLRecord(t, map[string]any{
		"type":      "user",
		"sessionId": sessionID,
		"slug":      slug,
		"timestamp": "2024-01-01T00:00:01Z",
		"cwd":       "/tmp",
		"message": map[string]any{
			"role": "user",
			"content": []map[string]any{
				{
					"type":        "tool_result",
					"tool_use_id": toolUseID,
					"content":     toolContent,
				},
				{
					"type": "text",
					"text": text,
				},
			},
		},
	})
}

func makeTestAssistantToolUseRecord(t testing.TB, sessionID, toolUseID string) string {
	t.Helper()

	return marshalTestJSONLRecord(t, map[string]any{
		"type":      "assistant",
		"sessionId": sessionID,
		"timestamp": "2024-01-01T00:00:00Z",
		"message": map[string]any{
			"role":  "assistant",
			"model": "claude",
			"content": []map[string]any{
				{
					"type": "text",
					"text": "reading file",
				},
				{
					"type": "tool_use",
					"id":   toolUseID,
					"name": testToolRead,
					"input": map[string]any{
						"file_path": "/tmp/large.txt",
					},
				},
			},
		},
	})
}

func marshalTestJSONLRecord(t testing.TB, rec map[string]any) string {
	t.Helper()

	raw, err := json.Marshal(rec)
	require.NoError(t, err)
	return string(raw)
}
