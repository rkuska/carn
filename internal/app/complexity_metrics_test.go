package app

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/boyter/scc/v3/processor"
)

type fileThresholds struct {
	maxComplexity int64
	maxCodeLines  int64
}

type complexityFileMetrics struct {
	relPath     string
	codeLines   int64
	complexity  int64
	totalLines  int64
	thresholds  fileThresholds
	isTest      bool
	isFailing   bool
	isWatchlist bool
}

const complexityWatchFraction = 0.75

var (
	updateComplexityDocs = flag.Bool(
		"update",
		false,
		"update generated complexity docs",
	)
	initSCCOnce sync.Once
)

// defaultSourceThresholds are the hard limits for non-test source files.
var defaultSourceThresholds = fileThresholds{
	maxComplexity: 80,
	maxCodeLines:  400,
}

// defaultTestThresholds are the hard limits for test files.
// Complexity is not checked for test files.
var defaultTestThresholds = fileThresholds{
	maxCodeLines: 800,
}

func initSCC() {
	initSCCOnce.Do(func() {
		processor.ProcessConstants()
	})
}

func repoRoot(t testing.TB) string {
	t.Helper()

	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func thresholdsForPath(path string) (fileThresholds, bool) {
	if strings.HasSuffix(path, "_test.go") {
		return defaultTestThresholds, true
	}
	return defaultSourceThresholds, false
}

func collectComplexityMetrics(repoRoot string) ([]complexityFileMetrics, error) {
	initSCC()

	internalRoot := filepath.Join(repoRoot, "internal")
	var metrics []complexityFileMetrics

	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("filepath.WalkDir: %w", walkErr)
		}
		if !isGoMetricFile(d) {
			return nil
		}

		metric, err := collectComplexityMetric(repoRoot, path)
		if err != nil {
			return fmt.Errorf("collectComplexityMetric_%s: %w", d.Name(), err)
		}
		metrics = append(metrics, metric)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collectComplexityMetrics: %w", err)
	}

	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].isFailing != metrics[j].isFailing {
			return metrics[i].isFailing
		}
		if metrics[i].complexity != metrics[j].complexity {
			return metrics[i].complexity > metrics[j].complexity
		}
		if metrics[i].codeLines != metrics[j].codeLines {
			return metrics[i].codeLines > metrics[j].codeLines
		}
		return metrics[i].relPath < metrics[j].relPath
	})

	return metrics, nil
}

func isGoMetricFile(d os.DirEntry) bool {
	return !d.IsDir() && strings.HasSuffix(d.Name(), ".go")
}

func collectComplexityMetric(repoRoot, path string) (complexityFileMetrics, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return complexityFileMetrics{}, fmt.Errorf("os.ReadFile: %w", err)
	}

	job := &processor.FileJob{
		Filename: path,
		Content:  content,
		Bytes:    int64(len(content)),
		Language: "Go",
	}
	processor.CountStats(job)

	relPath, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return complexityFileMetrics{}, fmt.Errorf("filepath.Rel: %w", err)
	}

	thresholds, isTest := thresholdsForPath(relPath)
	metric := complexityFileMetrics{
		relPath:    filepath.ToSlash(relPath),
		codeLines:  job.Code,
		complexity: job.Complexity,
		totalLines: job.Lines,
		thresholds: thresholds,
		isTest:     isTest,
	}
	metric.isFailing = metric.exceedsThresholds()
	metric.isWatchlist = metric.isFailing || metric.nearThresholds()
	return metric, nil
}

func (m complexityFileMetrics) exceedsThresholds() bool {
	if m.thresholds.maxComplexity > 0 && m.complexity > m.thresholds.maxComplexity {
		return true
	}
	if m.thresholds.maxCodeLines > 0 && m.codeLines > m.thresholds.maxCodeLines {
		return true
	}
	return false
}

func (m complexityFileMetrics) nearThresholds() bool {
	return nearThreshold(m.complexity, m.thresholds.maxComplexity) ||
		nearThreshold(m.codeLines, m.thresholds.maxCodeLines)
}

func nearThreshold(value, limit int64) bool {
	if limit <= 0 {
		return false
	}
	return float64(value) >= float64(limit)*complexityWatchFraction
}

func renderComplexityBaseline(metrics []complexityFileMetrics) string {
	var sb strings.Builder

	sb.WriteString("# Complexity Baseline\n\n")
	sb.WriteString("Generated from the current repository state.\n\n")
	sb.WriteString("Refresh command:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("go test ./internal/app -run TestComplexityBaselineDocument -count=1 -update\n")
	sb.WriteString("```\n\n")
	sb.WriteString("Thresholds enforced by `TestFileComplexityGuard`:\n\n")
	sb.WriteString("| Metric | Source files | Test files |\n")
	sb.WriteString("| --- | ---: | ---: |\n")
	fmt.Fprintf(&sb, "| Complexity | %d | not checked |\n", defaultSourceThresholds.maxComplexity)
	fmt.Fprintf(
		&sb,
		"| Code lines | %d | %d |\n\n",
		defaultSourceThresholds.maxCodeLines,
		defaultTestThresholds.maxCodeLines,
	)
	sb.WriteString("Files at or above 75% of a limit stay on the watchlist.\n\n")

	failing := filterComplexityMetrics(metrics, func(metric complexityFileMetrics) bool {
		return metric.isFailing
	})
	renderMetricTable(&sb, "## Failing files", failing)
	sb.WriteString("\n")

	watchlist := filterComplexityMetrics(metrics, func(metric complexityFileMetrics) bool {
		return metric.isWatchlist && !metric.isFailing
	})
	renderMetricTable(&sb, "## Watchlist", watchlist)
	sb.WriteString("\n")

	sb.WriteString("## Notes\n\n")
	sb.WriteString("- The hard gate is file-level and recursive across `internal/**/*.go`.\n")
	sb.WriteString("- Raising limits or adding exemptions is not part of the normal fix path.\n")
	sb.WriteString("- Use `COMPLEXITY_GUIDE.md` when the guard fails.\n")
	sb.WriteString("- Treat the watchlist in `COMPLEXITY_BASELINE.md` as the current queue.\n")

	return sb.String()
}

func filterComplexityMetrics(
	metrics []complexityFileMetrics,
	keep func(complexityFileMetrics) bool,
) []complexityFileMetrics {
	result := make([]complexityFileMetrics, 0, len(metrics))
	for _, metric := range metrics {
		if keep(metric) {
			result = append(result, metric)
		}
	}
	return result
}

func renderMetricTable(sb *strings.Builder, heading string, metrics []complexityFileMetrics) {
	sb.WriteString(heading)
	sb.WriteString("\n\n")
	if len(metrics) == 0 {
		sb.WriteString("None.\n")
		return
	}

	sb.WriteString("| File | Kind | Code | Complexity | Lines |\n")
	sb.WriteString("| --- | --- | ---: | ---: | ---: |\n")
	for _, metric := range metrics {
		kind := "source"
		if metric.isTest {
			kind = "test"
		}
		fmt.Fprintf(sb,
			"| %s | %s | %d | %d | %d |\n",
			metric.relPath,
			kind,
			metric.codeLines,
			metric.complexity,
			metric.totalLines,
		)
	}
}
