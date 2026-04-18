package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rkuska/carn/internal/coverage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunUpdateWritesCoverageBaseline(t *testing.T) {
	t.Parallel()

	root := writeTestRepo(t)
	runner := fakeRunner{
		profile: strings.Join([]string{
			"mode: set",
			"github.com/rkuska/carn/internal/app/app.go:1.1,2.1 3 1",
			"github.com/rkuska/carn/internal/archive/archive.go:1.1,4.1 5 1",
		}, "\n"),
	}

	require.NoError(t, run([]string{"-update"}, dependencies{
		getwd: func() (string, error) {
			return filepath.Join(root, "internal"), nil
		},
		runner: runner,
	}))

	got, err := coverage.ReadBaseline(filepath.Join(root, "COVERAGE_BASELINE.json"))
	require.NoError(t, err)

	assert.Equal(t, coverage.Baseline{
		SchemaVersion: 1,
		ModulePath:    "github.com/rkuska/carn",
		Total: coverage.Ratio{
			Covered:    8,
			Statements: 8,
		},
		Packages: map[string]coverage.Ratio{
			"github.com/rkuska/carn/internal/app": {
				Covered:    3,
				Statements: 3,
			},
			"github.com/rkuska/carn/internal/archive": {
				Covered:    5,
				Statements: 5,
			},
		},
	}, got)
}

func TestRunFailsWhenCoverageRegresses(t *testing.T) {
	t.Parallel()

	root := writeTestRepo(t)
	require.NoError(t, coverage.WriteBaseline(filepath.Join(root, "COVERAGE_BASELINE.json"), coverage.Baseline{
		SchemaVersion: 1,
		ModulePath:    "github.com/rkuska/carn",
		Total: coverage.Ratio{
			Covered:    9,
			Statements: 10,
		},
		Packages: map[string]coverage.Ratio{
			"github.com/rkuska/carn/internal/app": {
				Covered:    4,
				Statements: 5,
			},
		},
	}))

	err := run([]string{}, dependencies{
		getwd: func() (string, error) {
			return root, nil
		},
		runner: fakeRunner{
			profile: strings.Join([]string{
				"mode: set",
				"github.com/rkuska/carn/internal/app/app.go:1.1,2.1 3 1",
				"github.com/rkuska/carn/internal/app/app.go:3.1,5.1 2 0",
				"github.com/rkuska/carn/internal/archive/archive.go:1.1,4.1 5 1",
			}, "\n"),
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "coverage regressed")
	assert.Contains(t, err.Error(), "total")
	assert.Contains(t, err.Error(), "github.com/rkuska/carn/internal/app")
}

func TestRunFailsWhenBaselineIsMissing(t *testing.T) {
	t.Parallel()

	root := writeTestRepo(t)

	err := run([]string{}, dependencies{
		getwd: func() (string, error) {
			return root, nil
		},
		runner: fakeRunner{
			profile: strings.Join([]string{
				"mode: set",
				"github.com/rkuska/carn/internal/app/app.go:1.1,2.1 1 1",
			}, "\n"),
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "COVERAGE_BASELINE.json")
	assert.Contains(t, err.Error(), "-update")
}

type fakeRunner struct {
	profile string
	err     error
}

func (r fakeRunner) Run(_ context.Context, _ string, profilePath string, _ io.Writer, _ io.Writer) error {
	if r.err != nil {
		return r.err
	}
	return os.WriteFile(profilePath, []byte(r.profile), 0o644)
}

func writeTestRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	goMod := "module github.com/rkuska/carn\n\ngo 1.25.2\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal"), 0o755))
	return root
}
