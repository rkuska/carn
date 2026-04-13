package archive

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/rkuska/carn/internal/canonical"
	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
	"github.com/rkuska/carn/internal/source/claude"
	"github.com/rkuska/carn/internal/source/codex"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineRunSyncsAllClaudeArchiveFilesAndStoresUsefulOnesInSQLite(t *testing.T) {
	t.Parallel()

	syncedPaths, sqlitePaths := assertProviderArchiveCoverage(
		t,
		conv.ProviderClaude,
		claudeFixtureCorpusDir(t),
		claude.New(),
		usefulClaudeJSONLPaths,
	)

	commandOnly := filepath.ToSlash(filepath.Join("project-a", "session-command-only.jsonl"))
	assert.Contains(t, syncedPaths, commandOnly)
	assert.NotContains(t, sqlitePaths, commandOnly)
}

func TestPipelineRunSyncsAllCodexArchiveFilesAndStoresAllInSQLite(t *testing.T) {
	t.Parallel()

	assertProviderArchiveCoverage(
		t,
		conv.ProviderCodex,
		codexFixtureCorpusDir(t),
		codex.New(),
		relativeJSONLPaths,
	)
}

func assertProviderArchiveCoverage(
	t *testing.T,
	provider conv.Provider,
	sourceDir string,
	backend src.Backend,
	sqliteExpected func(t *testing.T, root string) []string,
) ([]string, []string) {
	t.Helper()

	archiveDir := filepath.Join(t.TempDir(), "archive")
	store := canonical.New(nil, backend)
	pipeline := New(Config{
		SourceDirs: map[conv.Provider]string{provider: sourceDir},
		ArchiveDir: archiveDir,
	}, store, backend)

	result, err := pipeline.Run(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, result.StoreBuilt)

	rawDir := src.ProviderRawDir(archiveDir, provider)
	wantSynced := relativeJSONLPaths(t, sourceDir)
	gotSynced := relativeJSONLPaths(t, rawDir)
	assert.Equal(t, wantSynced, gotSynced)

	wantSQLite := sqliteExpected(t, rawDir)
	gotSQLite := sqliteSessionRelPathsForProvider(t, archiveDir, provider)
	assert.Equal(t, wantSQLite, gotSQLite)

	return gotSynced, gotSQLite
}

func relativeJSONLPaths(t *testing.T, root string) []string {
	t.Helper()

	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	require.NoError(t, err)
	slices.Sort(paths)
	return paths
}

func usefulClaudeJSONLPaths(t *testing.T, root string) []string {
	t.Helper()

	paths := make([]string, 0)
	for _, relPath := range relativeJSONLPaths(t, root) {
		if claudeFileHasUsefulConversation(t, filepath.Join(root, filepath.FromSlash(relPath))) {
			paths = append(paths, relPath)
		}
	}
	return paths
}

func sqliteSessionRelPathsForProvider(t *testing.T, archiveDir string, provider conv.Provider) []string {
	t.Helper()

	rawDir := src.ProviderRawDir(archiveDir, provider)
	db, err := sql.Open("sqlite", filepath.Join(archiveDir, "store", "canonical.sqlite"))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	rows, err := db.Query(
		`SELECT conversation_sessions.file_path
		   FROM conversation_sessions
		   JOIN conversations ON conversations.id = conversation_sessions.conversation_id
		  WHERE conversations.provider = ?
		  ORDER BY conversation_sessions.file_path`,
		string(provider),
	)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rows.Close())
	}()

	paths := make([]string, 0)
	for rows.Next() {
		var path string
		require.NoError(t, rows.Scan(&path))
		rel, err := filepath.Rel(rawDir, path)
		require.NoError(t, err)
		paths = append(paths, filepath.ToSlash(rel))
	}
	require.NoError(t, rows.Err())
	return paths
}

func claudeFileHasUsefulConversation(t *testing.T, path string) bool {
	t.Helper()

	file, err := os.Open(path)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, file.Close())
	}()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if claudeLineHasUsefulConversation(scanner.Bytes()) {
			return true
		}
	}
	require.NoError(t, scanner.Err())
	return false
}

type coverageRecord struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
}

type coverageMessage struct {
	Content json.RawMessage `json:"content"`
}

type coverageBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Thinking string `json:"thinking"`
}

func claudeLineHasUsefulConversation(line []byte) bool {
	var rec coverageRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return false
	}

	var msg coverageMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return false
	}

	switch rec.Type {
	case "assistant":
		return claudeAssistantContentIsUseful(msg.Content)
	case "user":
		return claudeUserContentIsUseful(msg.Content)
	default:
		return false
	}
}

func claudeAssistantContentIsUseful(raw json.RawMessage) bool {
	var blocks []coverageBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return false
	}

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				return true
			}
		case "thinking":
			if block.Thinking != "" {
				return true
			}
		case "tool_use":
			return true
		}
	}
	return false
}

func claudeUserContentIsUseful(raw json.RawMessage) bool {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text != "" && !isClaudeHiddenSystemText(text)
	}

	var blocks []coverageBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return false
	}

	for _, block := range blocks {
		switch block.Type {
		case "tool_result":
			return true
		case "text":
			if block.Text != "" && !isClaudeHiddenSystemText(block.Text) {
				return true
			}
		}
	}
	return false
}

func isClaudeHiddenSystemText(text string) bool {
	if text == "[Request interrupted by user for tool use]" || text == "[Request interrupted by user]" {
		return true
	}
	for _, prefix := range []string{
		"<command-name>",
		"<local-command-stdout>",
		"<local-command-caveat>",
	} {
		if len(text) >= len(prefix) && text[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func claudeFixtureCorpusDir(t *testing.T) string {
	t.Helper()
	return testdataDir(t, "claude_raw")
}

func codexFixtureCorpusDir(t *testing.T) string {
	t.Helper()
	return testdataDir(t, "codex_raw")
}

func testdataDir(t *testing.T, name string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)

	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", name)
}
