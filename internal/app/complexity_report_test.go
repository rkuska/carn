package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectComplexityMetricsRecursesUnderInternal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	internalRoot := filepath.Join(root, "internal")

	writeComplexityTestFile(t, filepath.Join(internalRoot, "alpha.go"), "package alpha\n\nfunc A() {}\n")
	writeComplexityTestFile(
		t,
		filepath.Join(internalRoot, "nested", "beta_test.go"),
		"package nested\n\nfunc TestB() {}\n",
	)
	writeComplexityTestFile(t, filepath.Join(root, "outside.go"), "package outside\n\nfunc C() {}\n")

	metrics, err := collectComplexityMetrics(root)
	require.NoError(t, err)

	require.Len(t, metrics, 2)
	assert.Equal(t, "internal/alpha.go", metrics[0].relPath)
	assert.Equal(t, "internal/nested/beta_test.go", metrics[1].relPath)
	assert.False(t, metrics[0].isTest)
	assert.True(t, metrics[1].isTest)
}

func TestRenderComplexityBaselineIncludesFailingAndWatchlistSections(t *testing.T) {
	t.Parallel()

	doc := renderComplexityBaseline([]complexityFileMetrics{
		{
			relPath:     "internal/app/scanner.go",
			codeLines:   401,
			complexity:  81,
			totalLines:  500,
			thresholds:  defaultSourceThresholds,
			isFailing:   true,
			isWatchlist: true,
		},
		{
			relPath:     "internal/app/viewer.go",
			codeLines:   320,
			complexity:  62,
			totalLines:  400,
			thresholds:  defaultSourceThresholds,
			isWatchlist: true,
		},
	})

	assert.Contains(t, doc, "go test ./internal/app -run TestComplexityBaselineDocument -count=1 -update")
	assert.Contains(t, doc, "## Failing files")
	assert.Contains(t, doc, "| internal/app/scanner.go | source | 401 | 81 | 500 |")
	assert.Contains(t, doc, "## Watchlist")
	assert.Contains(t, doc, "| internal/app/viewer.go | source | 320 | 62 | 400 |")
	assert.Contains(t, doc, "COMPLEXITY_GUIDE.md")
	assert.Contains(t, doc, "COMPLEXITY_BASELINE.md")
}

func TestComplexityBaselineDocument(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	metrics, err := collectComplexityMetrics(root)
	require.NoError(t, err)

	got := renderComplexityBaseline(metrics)
	path := filepath.Join(root, "COMPLEXITY_BASELINE.md")
	if *updateComplexityDocs {
		require.NoError(t, os.WriteFile(path, []byte(got), 0o644))
	}

	wantBytes, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, strings.TrimSpace(string(wantBytes)), strings.TrimSpace(got))
}

func writeComplexityTestFile(t *testing.T, path, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
