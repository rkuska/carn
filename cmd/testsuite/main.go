package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rkuska/carn/internal/coverage"
)

type testRunner interface {
	Run(context.Context, string, string, io.Writer, io.Writer) error
}

type dependencies struct {
	getwd  func() (string, error)
	runner testRunner
}

type runConfig struct {
	update       bool
	baselinePath string
}

func main() {
	err := run(os.Args[1:], dependencies{
		getwd:  os.Getwd,
		runner: goTestRunner{},
	})
	if err == nil {
		return
	}

	if _, writeErr := fmt.Fprintln(os.Stderr, err); writeErr != nil {
		os.Exit(1)
	}
	os.Exit(1)
}

func run(args []string, deps dependencies) error {
	config, repoRoot, baselinePath, err := resolveRunContext(args, deps)
	if err != nil {
		return fmt.Errorf("run_resolveRunContext: %w", err)
	}

	snapshot, err := collectSnapshot(repoRoot, deps.runner)
	if err != nil {
		return fmt.Errorf("run_collectSnapshot: %w", err)
	}

	if config.update {
		if err := updateBaseline(repoRoot, baselinePath, snapshot); err != nil {
			return fmt.Errorf("run_updateBaseline: %w", err)
		}
		return nil
	}

	if err := checkBaseline(baselinePath, snapshot); err != nil {
		return fmt.Errorf("run_checkBaseline: %w", err)
	}

	return nil
}

type goTestRunner struct{}

func (goTestRunner) Run(
	ctx context.Context,
	repoRoot, profilePath string,
	stdout, stderr io.Writer,
) error {
	cmd := exec.CommandContext(
		ctx,
		"go",
		"test",
		"./...",
		"-count=1",
		"-covermode=set",
		"-coverpkg=./...",
		"-coverprofile="+profilePath,
	)
	cmd.Dir = repoRoot
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("goTestRunner_Run_cmd.Run: %w", err)
	}
	return nil
}

func resolveRunContext(args []string, deps dependencies) (runConfig, string, string, error) {
	config, err := parseArgs(args)
	if err != nil {
		return runConfig{}, "", "", fmt.Errorf("resolveRunContext_parseArgs: %w", err)
	}

	cwd, err := deps.getwd()
	if err != nil {
		return runConfig{}, "", "", fmt.Errorf("resolveRunContext_getwd: %w", err)
	}

	repoRoot, err := findRepoRoot(cwd)
	if err != nil {
		return runConfig{}, "", "", fmt.Errorf("resolveRunContext_findRepoRoot: %w", err)
	}

	baselinePath := config.baselinePath
	if baselinePath == "" {
		baselinePath = filepath.Join(repoRoot, "COVERAGE_BASELINE.json")
	}

	return config, repoRoot, baselinePath, nil
}

func collectSnapshot(repoRoot string, runner testRunner) (coverage.Snapshot, error) {
	profilePath, err := createProfilePath()
	if err != nil {
		return coverage.Snapshot{}, fmt.Errorf("collectSnapshot_createProfilePath: %w", err)
	}
	defer func() {
		if removeErr := removeProfile(profilePath); removeErr != nil {
			if _, writeErr := fmt.Fprintln(os.Stderr, removeErr); writeErr != nil {
				return
			}
		}
	}()

	runErr := runner.Run(context.Background(), repoRoot, profilePath, os.Stdout, os.Stderr)
	if runErr != nil {
		return coverage.Snapshot{}, fmt.Errorf("collectSnapshot_runner.Run: %w", runErr)
	}

	snapshot, err := readSnapshot(profilePath)
	if err != nil {
		return coverage.Snapshot{}, fmt.Errorf("collectSnapshot_readSnapshot: %w", err)
	}
	return snapshot, nil
}

func updateBaseline(repoRoot, baselinePath string, snapshot coverage.Snapshot) error {
	modulePath, err := readModulePath(repoRoot)
	if err != nil {
		return fmt.Errorf("updateBaseline_readModulePath: %w", err)
	}

	if err := coverage.WriteBaseline(baselinePath, coverage.NewBaseline(modulePath, snapshot)); err != nil {
		return fmt.Errorf("updateBaseline_WriteBaseline: %w", err)
	}

	if _, err := fmt.Fprintf(
		os.Stdout,
		"updated %s: total %s across %d packages\n",
		baselinePath,
		formatRatio(snapshot.Total),
		len(snapshot.Packages),
	); err != nil {
		return fmt.Errorf("updateBaseline_fmt.Fprintf: %w", err)
	}

	return nil
}

func checkBaseline(baselinePath string, snapshot coverage.Snapshot) error {
	baseline, err := coverage.ReadBaseline(baselinePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf(
				"checkBaseline: coverage baseline missing at %s; run go run ./cmd/testsuite -update",
				baselinePath,
			)
		}
		return fmt.Errorf("checkBaseline_ReadBaseline: %w", err)
	}

	regressions := coverage.Compare(baseline, snapshot)
	if len(regressions) > 0 {
		return fmt.Errorf("checkBaseline: %s", formatRegressions(regressions))
	}

	if _, err := fmt.Fprintf(
		os.Stdout,
		"coverage ok: total %s across %d packages\n",
		formatRatio(snapshot.Total),
		len(snapshot.Packages),
	); err != nil {
		return fmt.Errorf("checkBaseline_fmt.Fprintf: %w", err)
	}

	return nil
}

func parseArgs(args []string) (runConfig, error) {
	var config runConfig

	fs := flag.NewFlagSet("testsuite", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&config.update, "update", false, "update the committed coverage baseline")
	fs.StringVar(&config.baselinePath, "baseline", "", "override the coverage baseline path")
	if err := fs.Parse(args); err != nil {
		return runConfig{}, fmt.Errorf("parseArgs_Parse: %w", err)
	}
	if len(fs.Args()) > 0 {
		return runConfig{}, fmt.Errorf("parseArgs: unexpected args: %s", strings.Join(fs.Args(), ", "))
	}

	return config, nil
}

func findRepoRoot(cwd string) (string, error) {
	current := filepath.Clean(cwd)
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("findRepoRoot: go.mod not found")
		}
		current = parent
	}
}

func createProfilePath() (string, error) {
	file, err := os.CreateTemp("", "carn-coverage-*.out")
	if err != nil {
		return "", fmt.Errorf("createProfilePath_os.CreateTemp: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("createProfilePath_file.Close: %w", err)
	}
	return file.Name(), nil
}

func removeProfile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removeProfile_os.Remove: %w", err)
	}
	return nil
}

func readSnapshot(profilePath string) (coverage.Snapshot, error) {
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return coverage.Snapshot{}, fmt.Errorf("readSnapshot_os.ReadFile: %w", err)
	}

	snapshot, err := coverage.ParseSnapshot(bytes.NewReader(data))
	if err != nil {
		return coverage.Snapshot{}, fmt.Errorf("readSnapshot_ParseSnapshot: %w", err)
	}
	return snapshot, nil
}

func readModulePath(repoRoot string) (string, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("readModulePath_os.ReadFile: %w", err)
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after), nil
		}
	}

	return "", fmt.Errorf("readModulePath: module directive not found")
}

func formatRatio(ratio coverage.Ratio) string {
	return fmt.Sprintf("%.2f%%", ratio.Percent())
}

func formatRegressions(regressions []coverage.Regression) string {
	lines := []string{"coverage regressed:"}
	for _, regression := range regressions {
		lines = append(lines, fmt.Sprintf(
			"- %s: %s -> %s",
			regression.Name,
			formatRatio(regression.Baseline),
			formatRatio(regression.Current),
		))
	}
	return strings.Join(lines, "\n")
}
