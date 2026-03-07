package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestImportOverviewModelInit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(cfg)

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() should return a batch command")
	}
	if m.phase != phaseAnalyzing {
		t.Errorf("initial phase = %d, want phaseAnalyzing", m.phase)
	}
}

func TestImportOverviewAnalyzingDisablesEnter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(cfg)
	m.width = 120
	m.height = 40

	// Pressing Enter during analysis should not change phase
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.phase != phaseAnalyzing {
		t.Errorf("phase = %d, want phaseAnalyzing (Enter should be disabled)", m.phase)
	}
	if m.done {
		t.Error("should not be done during analysis")
	}
}

func TestImportOverviewAnalysisCompletionEnablesEnter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(cfg)
	m.width = 120
	m.height = 40

	// Simulate analysis finished with nothing to sync
	m, _ = m.Update(analysisFinishedMsg{
		analysis: importAnalysis{
			sourceDir:  cfg.sourceDir,
			archiveDir: cfg.archiveDir,
		},
	})

	if m.phase != phaseReady {
		t.Errorf("phase = %d, want phaseReady", m.phase)
	}

	// Now Enter should set done (nothing to sync → skip to browser)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if !m.done {
		t.Error("expected done = true after Enter on empty analysis")
	}
}

func TestImportOverviewEnterStartsSync(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(cfg)
	m.width = 120
	m.height = 40

	// Simulate analysis finished with files to sync
	m, _ = m.Update(analysisFinishedMsg{
		analysis: importAnalysis{
			sourceDir:        cfg.sourceDir,
			archiveDir:       cfg.archiveDir,
			filesToSync:      []string{filepath.Join(dir, "source", "a.jsonl")},
			newConversations: 1,
			conversations:    1,
		},
	})

	if m.phase != phaseReady {
		t.Fatalf("phase = %d, want phaseReady", m.phase)
	}

	// Enter should start sync
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.phase != phaseSyncing {
		t.Errorf("phase = %d, want phaseSyncing", m.phase)
	}
	if cmd == nil {
		t.Error("expected copy batch command")
	}
	if m.total != 1 {
		t.Errorf("total = %d, want 1", m.total)
	}
}

func TestImportOverviewSyncCompletion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(cfg)
	m.width = 120
	m.height = 40

	// Set up syncing phase directly
	m.phase = phaseSyncing
	m.files = []string{filepath.Join(dir, "source", "a.jsonl")}
	m.total = 1
	m.nextIndex = 1
	m.inFlight = 1
	m.startTime = time.Now()

	// Simulate file copied
	m, _ = m.Update(importSyncFileCopiedMsg{
		file: filepath.Join(dir, "source", "a.jsonl"),
	})

	if m.phase != phaseDone {
		t.Errorf("phase = %d, want phaseDone", m.phase)
	}
	if m.result.copied != 1 {
		t.Errorf("copied = %d, want 1", m.result.copied)
	}

	// Enter in done phase should set done
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if !m.done {
		t.Error("expected done = true after Enter on sync complete")
	}
}

func TestImportOverviewEmptyProjectDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(cfg)
	m.width = 120
	m.height = 40

	// No project dirs
	m, _ = m.Update(listProjectDirsMsg{dirs: nil})

	if m.phase != phaseReady {
		t.Errorf("phase = %d, want phaseReady", m.phase)
	}
	if m.analysis.needsSync() {
		t.Error("expected no sync needed for empty source")
	}
}

func TestImportOverviewListProjectDirsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(cfg)
	m.width = 120
	m.height = 40

	// Error listing dirs
	m, _ = m.Update(listProjectDirsMsg{err: fmt.Errorf("permission denied")})

	if m.phase != phaseReady {
		t.Errorf("phase = %d, want phaseReady (should recover from error)", m.phase)
	}
}

func TestImportOverviewWindowResize(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(cfg)

	m, _ = m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})

	if m.width != 200 {
		t.Errorf("width = %d, want 200", m.width)
	}
	if m.height != 50 {
		t.Errorf("height = %d, want 50", m.height)
	}
}

func TestImportOverviewSpinnerTick(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(cfg)
	m.width = 120
	m.height = 40

	// Should not panic
	m, _ = m.Update(spinner.TickMsg{})
	_ = m.View()
}

func TestImportOverviewViewRendersInAllPhases(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}

	t.Run("analyzing", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseAnalyzing
		m.analysisProgress = importProgress{
			filesInspected: 42,
			conversations:  10,
		}

		view := ansi.Strip(m.View())
		if view == "" {
			t.Error("expected non-empty view")
		}
		if !strings.Contains(view, "Import Workspace") {
			t.Fatalf("expected unified import title, got: %s", view)
		}
		for _, label := range []string{"Source", "Archive", "Projects", "Files", "Conversations"} {
			if !strings.Contains(view, label) {
				t.Fatalf("expected persistent dashboard section %q, got: %s", label, view)
			}
		}
		if !strings.Contains(view, "Scanning Claude projects") {
			t.Fatalf("expected analyzing status copy, got: %s", view)
		}
	})

	t.Run("ready with sync needed", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseReady
		m.analysis = importAnalysis{
			sourceDir:        cfg.sourceDir,
			archiveDir:       cfg.archiveDir,
			filesInspected:   100,
			projects:         5,
			conversations:    50,
			newConversations: 10,
			toUpdate:         5,
			upToDate:         35,
			filesToSync:      []string{"a.jsonl"},
		}

		view := ansi.Strip(m.View())
		if view == "" {
			t.Error("expected non-empty view")
		}
		if !strings.Contains(view, "Ready to Import") {
			t.Fatalf("expected ready status, got: %s", view)
		}
		if !strings.Contains(view, "Projects") || !strings.Contains(view, "Current") {
			t.Fatalf("expected review summary metrics, got: %s", view)
		}
		if !strings.Contains(view, "Will import") {
			t.Fatalf("expected import summary, got: %s", view)
		}
		if !strings.Contains(view, "Press Enter to import") {
			t.Fatalf("expected import CTA, got: %s", view)
		}
	})

	t.Run("ready without sync", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseReady
		m.analysis = importAnalysis{
			sourceDir:  cfg.sourceDir,
			archiveDir: cfg.archiveDir,
			upToDate:   10,
		}

		view := ansi.Strip(m.View())
		if view == "" {
			t.Error("expected non-empty view")
		}
		if !strings.Contains(view, "No import needed") {
			t.Fatalf("expected no-op outcome, got: %s", view)
		}
		if !strings.Contains(view, "Press Enter to continue") {
			t.Fatalf("expected continue CTA, got: %s", view)
		}
	})

	t.Run("syncing", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseSyncing
		m.total = 5
		m.current = 2
		m.result = syncResult{copied: 1, failed: 1}
		m.currentFile = "test.jsonl"

		view := ansi.Strip(m.View())
		if view == "" {
			t.Error("expected non-empty view")
		}
		if !strings.Contains(view, "Importing") {
			t.Fatalf("expected syncing status, got: %s", view)
		}
		if !strings.Contains(view, "2/5") {
			t.Fatalf("expected progress counts, got: %s", view)
		}
		if !strings.Contains(view, "Copied") || !strings.Contains(view, "Failed") {
			t.Fatalf("expected sync counters, got: %s", view)
		}
		if !strings.Contains(view, "test.jsonl") {
			t.Fatalf("expected current file, got: %s", view)
		}
	})

	t.Run("done", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseDone
		m.total = 3
		m.result = syncResult{copied: 3, failed: 0, elapsed: time.Second}

		view := ansi.Strip(m.View())
		if view == "" {
			t.Error("expected non-empty view")
		}
		if !strings.Contains(view, "Import Workspace") {
			t.Fatalf("expected unified import title, got: %s", view)
		}
		if !strings.Contains(view, "Complete") {
			t.Fatalf("expected completion status, got: %s", view)
		}
		if !strings.Contains(view, "Elapsed") {
			t.Fatalf("expected elapsed summary, got: %s", view)
		}
		if !strings.Contains(view, "Press Enter to continue") {
			t.Fatalf("expected continue CTA, got: %s", view)
		}
	})

	t.Run("zero width returns empty", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(cfg)

		view := m.View()
		if view != "" {
			t.Error("expected empty view for zero width")
		}
	})
}

func TestImportOverviewViewPreservesBottomContentWhenPathsWrap(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, strings.Repeat("very-long-source-path-segment-", 4)),
		archiveDir: filepath.Join(dir, strings.Repeat("very-long-archive-path-segment-", 4)),
	}

	m := newImportOverviewModel(cfg)
	m.width = 52
	m.height = 20
	m.phase = phaseReady
	m.analysis = importAnalysis{
		sourceDir:  cfg.sourceDir,
		archiveDir: cfg.archiveDir,
	}

	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Press Enter to continue") {
		t.Fatalf("expected wrapped import overview to keep continue CTA visible, got: %s", view)
	}
	if !strings.Contains(view, "Source") || !strings.Contains(view, "Archive") {
		t.Fatalf("expected wrapped import overview to keep path context visible, got: %s", view)
	}
}

func TestImportOverviewAnalysisPipeline(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	archDir := filepath.Join(dir, "archive")

	// Create two projects with some files
	writeTestFile(t, filepath.Join(srcDir, "proj1", "s1.jsonl"),
		makeJSONLRecord("user", "feat-a", "id1"))
	writeTestFile(t, filepath.Join(srcDir, "proj2", "s2.jsonl"),
		makeJSONLRecord("user", "feat-b", "id2"))

	cfg := archiveConfig{sourceDir: srcDir, archiveDir: archDir}
	m := newImportOverviewModel(cfg)
	m.width = 120
	m.height = 40

	// Step 1: List project dirs
	m, _ = m.Update(listProjectDirsMsg{
		dirs: []string{
			filepath.Join(srcDir, "proj1"),
			filepath.Join(srcDir, "proj2"),
		},
	})

	if m.projIndex != 0 {
		t.Errorf("projIndex = %d, want 0", m.projIndex)
	}
	if len(m.projectDirs) != 2 {
		t.Errorf("projectDirs = %d, want 2", len(m.projectDirs))
	}

	// Step 2: First project analyzed
	m, cmd := m.Update(analysisProgressMsg{
		progress: importProgress{
			filesInspected: 1,
			currentProject: "proj1",
		},
		seen: map[groupKey]*conversationState{
			{dirName: "proj1", slug: "feat-a"}: {allNew: true, hasStale: true},
		},
		syncCandidates: []string{filepath.Join(srcDir, "proj1", "s1.jsonl")},
	})

	if m.projIndex != 1 {
		t.Errorf("projIndex = %d, want 1", m.projIndex)
	}
	if m.totalInspected != 1 {
		t.Errorf("totalInspected = %d, want 1", m.totalInspected)
	}
	if cmd == nil {
		t.Error("expected command to analyze next project")
	}

	// Step 3: Second project analyzed
	m, cmd = m.Update(analysisProgressMsg{
		progress: importProgress{
			filesInspected: 1,
			currentProject: "proj2",
		},
		seen: map[groupKey]*conversationState{
			{dirName: "proj2", slug: "feat-b"}: {allNew: true, hasStale: true},
		},
		syncCandidates: []string{filepath.Join(srcDir, "proj2", "s2.jsonl")},
	})

	if m.totalInspected != 2 {
		t.Errorf("totalInspected = %d, want 2", m.totalInspected)
	}
	if len(m.syncCandidates) != 2 {
		t.Errorf("syncCandidates = %d, want 2", len(m.syncCandidates))
	}
	if cmd == nil {
		t.Error("expected command to finish analysis")
	}

	// Step 4: Analysis finished message
	m, _ = m.Update(analysisFinishedMsg{
		analysis: importAnalysis{
			sourceDir:        cfg.sourceDir,
			archiveDir:       cfg.archiveDir,
			filesInspected:   2,
			projects:         2,
			conversations:    2,
			newConversations: 2,
			filesToSync:      m.syncCandidates,
		},
	})

	if m.phase != phaseReady {
		t.Errorf("phase = %d, want phaseReady", m.phase)
	}
	if !m.analysis.needsSync() {
		t.Error("expected needsSync() = true")
	}
}

func TestImportOverviewSyncingDisablesEnter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(cfg)
	m.width = 120
	m.height = 40
	m.phase = phaseSyncing

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.done {
		t.Error("Enter should be disabled during syncing")
	}
	if m.phase != phaseSyncing {
		t.Errorf("phase should remain phaseSyncing, got %d", m.phase)
	}
}

func TestImportOverviewSyncFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(cfg)
	m.width = 120
	m.height = 40
	m.phase = phaseSyncing
	m.files = []string{"a.jsonl"}
	m.total = 1
	m.nextIndex = 1
	m.inFlight = 1
	m.startTime = time.Now()

	// Simulate failed copy
	m, _ = m.Update(importSyncFileCopiedMsg{
		file: "a.jsonl",
		err:  fmt.Errorf("copy failed"),
	})

	if m.phase != phaseDone {
		t.Errorf("phase = %d, want phaseDone", m.phase)
	}
	if m.result.failed != 1 {
		t.Errorf("failed = %d, want 1", m.result.failed)
	}
	if m.result.copied != 0 {
		t.Errorf("copied = %d, want 0", m.result.copied)
	}
}
