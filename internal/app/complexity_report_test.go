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

func TestCollectModuleComplexityMetricsGroupsByPackageDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	internalRoot := filepath.Join(root, "internal")

	writeComplexityTestFile(t, filepath.Join(internalRoot, "source", "source.go"), "package source\n\nfunc Source() {}\n")
	writeComplexityTestFile(
		t,
		filepath.Join(internalRoot, "source", "source_test.go"),
		"package source\n\nfunc TestSource() {}\n",
	)
	writeComplexityTestFile(t, filepath.Join(internalRoot, "source", "claude", "scan.go"), "package claude\n\nfunc Scan() {}\n")
	writeComplexityTestFile(
		t,
		filepath.Join(internalRoot, "source", "claude", "scan_test.go"),
		"package claude\n\nfunc TestScan() {}\n",
	)

	fileMetrics, err := collectComplexityMetrics(root)
	require.NoError(t, err)

	modules := collectModuleComplexityMetrics(fileMetrics)
	require.Len(t, modules, 2)

	modulesByPath := make(map[string]complexityModuleMetrics, len(modules))
	for _, module := range modules {
		modulesByPath[module.relPath] = module
	}

	sourceModule := modulesByPath["internal/source"]
	assert.EqualValues(t, 1, sourceModule.sourceFiles)
	assert.EqualValues(t, 1, sourceModule.testFiles)
	assert.Greater(t, sourceModule.sourceCodeLines, int64(0))
	assert.Greater(t, sourceModule.testCodeLines, int64(0))

	claudeModule := modulesByPath["internal/source/claude"]
	assert.EqualValues(t, 1, claudeModule.sourceFiles)
	assert.EqualValues(t, 1, claudeModule.testFiles)
	assert.Greater(t, claudeModule.sourceCodeLines, int64(0))
	assert.Greater(t, claudeModule.testCodeLines, int64(0))
}

func TestRenderComplexityBaselineIncludesFailingAndWatchlistSections(t *testing.T) {
	t.Parallel()

	doc := renderComplexityBaseline(
		[]complexityFileMetrics{
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
		},
		[]complexityModuleMetrics{
			{
				relPath:          "internal/app",
				sourceFiles:      12,
				sourceCodeLines:  6400,
				sourceComplexity: 1300,
				sourceTotalLines: 7000,
				testFiles:        3,
				testCodeLines:    900,
				testComplexity:   12,
				testTotalLines:   1020,
				thresholds:       defaultModuleThresholds,
				isFailing:        true,
				isWatchlist:      true,
			},
			{
				relPath:          "internal/canonical",
				sourceFiles:      10,
				sourceCodeLines:  4800,
				sourceComplexity: 950,
				sourceTotalLines: 5300,
				testFiles:        2,
				testCodeLines:    600,
				testComplexity:   10,
				testTotalLines:   720,
				thresholds:       defaultModuleThresholds,
				isWatchlist:      true,
			},
		},
	)

	assert.Contains(t, doc, "go test ./internal/app -run TestComplexityBaselineDocument -count=1 -update")
	assert.Contains(t, doc, "## Failing files")
	assert.Contains(t, doc, "| internal/app/scanner.go | source | 401 | 81 | 500 |")
	assert.Contains(t, doc, "## File watchlist")
	assert.Contains(t, doc, "| internal/app/viewer.go | source | 320 | 62 | 400 |")
	assert.Contains(t, doc, "Thresholds enforced by `TestModuleComplexityGuard`")
	assert.Contains(t, doc, "## Failing modules")
	assert.Contains(t, doc, "| internal/app | 12 | 6400 | 1300 | 7000 | 3 | 900 | 12 | 1020 |")
	assert.Contains(t, doc, "## Module watchlist")
	assert.Contains(t, doc, "| internal/canonical | 10 | 4800 | 950 | 5300 | 2 | 600 | 10 | 720 |")
	assert.Contains(t, doc, "COMPLEXITY_GUIDE.md")
	assert.Contains(t, doc, "COMPLEXITY_BASELINE.md")
}

func TestComplexityBaselineDocument(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	fileMetrics, err := collectComplexityMetrics(root)
	require.NoError(t, err)

	moduleMetrics := collectModuleComplexityMetrics(fileMetrics)
	got := renderComplexityBaseline(fileMetrics, moduleMetrics)
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
