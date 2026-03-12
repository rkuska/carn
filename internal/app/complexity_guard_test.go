package app

import "testing"

func assertFileMetrics(t *testing.T, metric complexityFileMetrics) {
	t.Helper()

	if metric.thresholds.maxComplexity > 0 && metric.complexity > metric.thresholds.maxComplexity {
		t.Errorf(
			"%s complexity %d exceeds limit %d",
			metric.relPath,
			metric.complexity,
			metric.thresholds.maxComplexity,
		)
	}

	if metric.thresholds.maxCodeLines > 0 && metric.codeLines > metric.thresholds.maxCodeLines {
		t.Errorf(
			"%s code lines %d exceeds limit %d",
			metric.relPath,
			metric.codeLines,
			metric.thresholds.maxCodeLines,
		)
	}
}

func TestFileComplexityGuard(t *testing.T) {
	t.Parallel()

	metrics, err := collectComplexityMetrics(repoRoot(t))
	if err != nil {
		t.Fatalf("collectComplexityMetrics: %v", err)
	}

	for _, metric := range metrics {
		assertFileMetrics(t, metric)
	}
}
