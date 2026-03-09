package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func fixtureCorpusDir(tb testing.TB) string {
	tb.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(tb, ok)

	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "claude_raw")
}

func copyFixtureCorpusToSource(tb testing.TB, sourceDir string) {
	tb.Helper()
	copyFixtureDir(tb, fixtureCorpusDir(tb), sourceDir)
}

func copyFixtureCorpusToArchive(tb testing.TB, archiveDir string) {
	tb.Helper()
	copyFixtureDir(
		tb,
		fixtureCorpusDir(tb),
		providerRawDir(archiveDir, conversationProviderClaude),
	)
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
