package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/boyter/scc/v3/processor"
)

var initSCCOnce sync.Once

func initSCC() {
	initSCCOnce.Do(func() {
		processor.ProcessConstants()
	})
}

type fileThresholds struct {
	maxComplexity int64
	maxCodeLines  int64
}

// defaultSourceThresholds are the default limits for non-test source files.
var defaultSourceThresholds = fileThresholds{
	maxComplexity: 120,
	maxCodeLines:  500,
}

// defaultTestThresholds are the default limits for test files.
// Complexity is not checked for test files.
var defaultTestThresholds = fileThresholds{
	maxCodeLines: 800,
}

// complexityExceptions lists files that are allowed to exceed default
// thresholds. Set caps just above current values — add entries consciously.
var complexityExceptions = map[string]fileThresholds{
	"canonical_store.go": {maxComplexity: 400, maxCodeLines: 1300},
	"scanner.go":         {maxComplexity: 120, maxCodeLines: 1300},
	"scanner_test.go":    {maxComplexity: 120, maxCodeLines: 1900},
}

func thresholdsForFile(name string) (fileThresholds, bool) {
	isTest := strings.HasSuffix(name, "_test.go")
	th := defaultSourceThresholds
	if isTest {
		th = defaultTestThresholds
	}
	if exc, ok := complexityExceptions[name]; ok {
		th = exc
	}
	return th, isTest
}

func assertFileMetrics(t *testing.T, name string, job *processor.FileJob) {
	t.Helper()

	th, isTest := thresholdsForFile(name)

	if !isTest && th.maxComplexity > 0 && job.Complexity > th.maxComplexity {
		t.Errorf("complexity %d exceeds limit %d", job.Complexity, th.maxComplexity)
	}

	if th.maxCodeLines > 0 && job.Code > th.maxCodeLines {
		t.Errorf("code lines %d exceeds limit %d", job.Code, th.maxCodeLines)
	}
}

func TestFileComplexityGuard(t *testing.T) {
	t.Parallel()
	initSCC()

	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(thisFile)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}

			job := &processor.FileJob{
				Filename: name,
				Content:  content,
				Bytes:    int64(len(content)),
				Language: "Go",
			}
			processor.CountStats(job)

			assertFileMetrics(t, name, job)
		})
	}
}
