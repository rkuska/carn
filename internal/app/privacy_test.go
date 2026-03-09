package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTestArtifactsDoNotContainAuthorSpecificPaths(t *testing.T) {
	t.Parallel()

	repoRoot := testRepoRoot(t)
	roots := []string{
		filepath.Join(repoRoot, "internal", "app"),
		filepath.Join(repoRoot, "scenarios"),
		filepath.Join(repoRoot, "testdata"),
	}
	banned := []string{
		"xku" + "ska",
		"apro" + "pos",
		"/var/" + "folders/",
	}

	var hits []string
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if !shouldScanTestArtifact(path) {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			content := string(data)

			for _, needle := range banned {
				if strings.Contains(content, needle) {
					rel, err := filepath.Rel(repoRoot, path)
					if err != nil {
						return err
					}
					hits = append(hits, fmt.Sprintf("%s contains %q", rel, needle))
				}
			}
			return nil
		})
		require.NoError(t, err)
	}

	require.Empty(t, hits, strings.Join(hits, "\n"))
}

func testRepoRoot(t testing.TB) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func shouldScanTestArtifact(path string) bool {
	switch {
	case strings.HasSuffix(path, "_test.go"):
		return true
	case strings.HasSuffix(path, ".golden"):
		return true
	case strings.HasSuffix(path, ".jsonl"):
		return true
	case strings.Contains(path, string(filepath.Separator)+"testdata"+string(filepath.Separator)) &&
		strings.HasSuffix(path, ".md"):
		return true
	default:
		return false
	}
}
