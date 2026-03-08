package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportOverviewModelInit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(context.Background(), cfg)

	cmd := m.Init()
	require.NotNil(t, cmd)
	assert.Equal(t, phaseAnalyzing, m.phase)
}

func TestImportOverviewAnalyzingDisablesEnter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(context.Background(), cfg)
	m.width = 120
	m.height = 40

	// Pressing Enter during analysis should not change phase
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.Equal(t, phaseAnalyzing, m.phase)
	assert.False(t, m.done)
}

func TestImportOverviewAnalysisCompletionEnablesEnter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(context.Background(), cfg)
	m.width = 120
	m.height = 40

	// Simulate analysis finished with nothing to sync
	m, _ = m.Update(analysisFinishedMsg{
		analysis: importAnalysis{
			sourceDir:  cfg.sourceDir,
			archiveDir: cfg.archiveDir,
		},
	})

	assert.Equal(t, phaseReady, m.phase)

	// Now Enter should set done (nothing to sync → skip to browser)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.True(t, m.done)
}

func TestImportOverviewEnterStartsSync(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(context.Background(), cfg)
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

	assert.Equal(t, phaseReady, m.phase)

	// Enter should start sync
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.Equal(t, phaseSyncing, m.phase)
	require.NotNil(t, cmd)
	assert.Equal(t, 1, m.total)
}

func TestImportOverviewSyncCompletion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(context.Background(), cfg)
	m.width = 120
	m.height = 40

	// Set up syncing phase directly
	m.phase = phaseSyncing
	m.files = []string{filepath.Join(dir, "source", "a.jsonl")}
	m.total = 1

	// Simulate progress and completion
	m, _ = m.Update(importSyncProgressMsg{
		progress: syncProgress{
			current: 1,
			total:   1,
			file:    "a.jsonl",
			copied:  1,
		},
	})
	m, _ = m.Update(importSyncFinishedMsg{
		result: syncResult{
			copied:  1,
			elapsed: time.Second,
		},
	})

	assert.Equal(t, phaseDone, m.phase)
	assert.Equal(t, 1, m.result.copied)

	// Enter in done phase should set done
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.True(t, m.done)
}

func TestImportOverviewEmptyProjectDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(context.Background(), cfg)
	m.width = 120
	m.height = 40

	// No project dirs
	m, _ = m.Update(listProjectDirsMsg{dirs: nil})

	assert.Equal(t, phaseReady, m.phase)
	assert.False(t, m.analysis.needsSync())
}

func TestImportOverviewListProjectDirsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(context.Background(), cfg)
	m.width = 120
	m.height = 40

	// Error listing dirs
	m, _ = m.Update(listProjectDirsMsg{err: fmt.Errorf("permission denied")})

	assert.Equal(t, phaseReady, m.phase)
}

func TestImportOverviewWindowResize(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(context.Background(), cfg)

	m, _ = m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})

	assert.Equal(t, 200, m.width)
	assert.Equal(t, 50, m.height)
}

func TestImportOverviewSpinnerTick(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(context.Background(), cfg)
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
		m := newImportOverviewModel(context.Background(), cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseAnalyzing
		m.analysisProgress = importProgress{
			filesInspected: 42,
			conversations:  10,
		}

		view := ansi.Strip(m.View())
		require.NotEmpty(t, view)
		assertContainsAll(t, view,
			"Import Workspace",
			"Source",
			"Archive",
			"Projects",
			"Files",
			"Conversations",
			"Scanning Claude projects",
		)
	})

	t.Run("ready with sync needed", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(context.Background(), cfg)
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
		require.NotEmpty(t, view)
		assertContainsAll(t, view, "Ready to Import", "Projects", "Current", "Will import", "Press Enter to import")
	})

	t.Run("ready without sync", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(context.Background(), cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseReady
		m.analysis = importAnalysis{
			sourceDir:  cfg.sourceDir,
			archiveDir: cfg.archiveDir,
			upToDate:   10,
		}

		view := ansi.Strip(m.View())
		require.NotEmpty(t, view)
		assertContainsAll(t, view, "No import needed", "Press Enter to continue")
	})

	t.Run("syncing", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(context.Background(), cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseSyncing
		m.total = 5
		m.current = 2
		m.result = syncResult{copied: 1, failed: 1}
		m.currentFile = "test.jsonl"

		view := ansi.Strip(m.View())
		require.NotEmpty(t, view)
		assertContainsAll(t, view, "Importing", "2/5", "Copied", "Failed", "test.jsonl")
	})

	t.Run("done", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(context.Background(), cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseDone
		m.total = 3
		m.result = syncResult{copied: 3, failed: 0, elapsed: time.Second}

		view := ansi.Strip(m.View())
		require.NotEmpty(t, view)
		assertContainsAll(t, view, "Import Workspace", "Complete", "Elapsed", "Press Enter to continue")
	})

	t.Run("zero width returns empty", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(context.Background(), cfg)

		view := m.View()
		assert.Empty(t, view)
	})
}

func TestImportOverviewViewPreservesBottomContentWhenPathsWrap(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, strings.Repeat("very-long-source-path-segment-", 4)),
		archiveDir: filepath.Join(dir, strings.Repeat("very-long-archive-path-segment-", 4)),
	}

	m := newImportOverviewModel(context.Background(), cfg)
	m.width = 52
	m.height = 20
	m.phase = phaseReady
	m.analysis = importAnalysis{
		sourceDir:  cfg.sourceDir,
		archiveDir: cfg.archiveDir,
	}

	view := ansi.Strip(m.View())
	assertContainsAll(t, view, "Press Enter to continue", "Source", "Archive")
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
	m := newImportOverviewModel(context.Background(), cfg)
	m.width = 120
	m.height = 40

	// Step 1: List project dirs
	m, _ = m.Update(listProjectDirsMsg{
		dirs: []string{
			filepath.Join(srcDir, "proj1"),
			filepath.Join(srcDir, "proj2"),
		},
	})

	assert.Zero(t, m.projIndex)
	assert.Len(t, m.projectDirs, 2)

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

	assert.Equal(t, 1, m.projIndex)
	assert.Equal(t, 1, m.totalInspected)
	require.NotNil(t, cmd)

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

	assert.Equal(t, 2, m.totalInspected)
	assert.Len(t, m.syncCandidates, 2)
	require.NotNil(t, cmd)

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

	assert.Equal(t, phaseReady, m.phase)
	assert.True(t, m.analysis.needsSync())
}

func TestImportOverviewSyncingDisablesEnter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(context.Background(), cfg)
	m.width = 120
	m.height = 40
	m.phase = phaseSyncing

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.False(t, m.done)
	assert.Equal(t, phaseSyncing, m.phase)
}

func TestImportOverviewSyncFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newImportOverviewModel(context.Background(), cfg)
	m.width = 120
	m.height = 40
	m.phase = phaseSyncing
	m.files = []string{"a.jsonl"}
	m.total = 1

	// Simulate failed copy
	m, _ = m.Update(importSyncProgressMsg{
		progress: syncProgress{
			current: 1,
			total:   1,
			file:    "a.jsonl",
			failed:  1,
		},
	})
	m, _ = m.Update(importSyncFinishedMsg{
		result: syncResult{
			failed:  1,
			elapsed: time.Second,
		},
	})

	assert.Equal(t, phaseDone, m.phase)
	assert.Equal(t, 1, m.result.failed)
	assert.Zero(t, m.result.copied)
}
