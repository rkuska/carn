package app

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	arch "github.com/rkuska/carn/internal/archive"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubImportPipeline struct {
	analyzeFn func(context.Context, func(arch.ImportProgress)) (arch.ImportAnalysis, error)
	runFn     func(context.Context, func(arch.SyncProgress)) (arch.SyncResult, error)
}

func (p stubImportPipeline) Analyze(
	ctx context.Context,
	onProgress func(arch.ImportProgress),
) (arch.ImportAnalysis, error) {
	if p.analyzeFn != nil {
		return p.analyzeFn(ctx, onProgress)
	}
	return arch.ImportAnalysis{}, nil
}

func (p stubImportPipeline) Run(
	ctx context.Context,
	onProgress func(arch.SyncProgress),
) (arch.SyncResult, error) {
	if p.runFn != nil {
		return p.runFn(ctx, onProgress)
	}
	return arch.SyncResult{}, nil
}

func testImportOverviewConfig(t *testing.T) arch.Config {
	t.Helper()
	dir := t.TempDir()
	return arch.Config{
		SourceDirs: map[conv.Provider]string{
			conv.ProviderClaude: filepath.Join(dir, "source"),
		},
		ArchiveDir: filepath.Join(dir, "archive"),
	}
}

func testImportOverviewSourceDir(cfg arch.Config) string {
	return cfg.SourceDirFor(conv.ProviderClaude)
}

func TestImportOverviewModelInit(t *testing.T) {
	t.Parallel()

	m := newImportOverviewModel(context.Background(), testImportOverviewConfig(t))

	cmd := m.Init()
	require.NotNil(t, cmd)
	assert.Equal(t, phaseAnalyzing, m.phase)
}

func TestImportOverviewAnalyzingDisablesEnter(t *testing.T) {
	t.Parallel()

	m := newImportOverviewModel(context.Background(), testImportOverviewConfig(t))
	m.width = 120
	m.height = 40

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.Equal(t, phaseAnalyzing, m.phase)
	assert.False(t, m.done)
}

func TestImportOverviewReadyWithoutSyncContinuesToBrowser(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	m := newImportOverviewModel(context.Background(), cfg)
	m.width = 120
	m.height = 40

	m, _ = m.Update(analysisFinishedMsg{
		analysis: arch.ImportAnalysis{
			ArchiveDir: cfg.ArchiveDir,
		},
	})

	require.Equal(t, phaseReady, m.phase)

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.True(t, m.done)
}

func TestImportOverviewReadyWithSyncStartsSync(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	m := newImportOverviewModel(context.Background(), cfg)
	m.width = 120
	m.height = 40

	m, _ = m.Update(analysisFinishedMsg{
		analysis: arch.ImportAnalysis{
			ArchiveDir:       cfg.ArchiveDir,
			QueuedFiles:      []string{filepath.Join(testImportOverviewSourceDir(cfg), "a.jsonl")},
			NewConversations: 1,
			Conversations:    1,
		},
	})

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.Equal(t, phaseSyncing, m.phase)
	assert.Equal(t, 1, m.total)
	require.NotNil(t, cmd)
}

func TestImportOverviewSyncCompletion(t *testing.T) {
	t.Parallel()

	m := newImportOverviewModel(context.Background(), testImportOverviewConfig(t))
	m.width = 120
	m.height = 40
	m.phase = phaseSyncing
	m.files = []string{"a.jsonl"}
	m.total = 1

	m, _ = m.Update(importSyncProgressMsg{
		progress: arch.SyncProgress{
			Current: 1,
			Total:   1,
			File:    "a.jsonl",
			Copied:  1,
		},
	})
	m, _ = m.Update(importSyncFinishedMsg{
		result: arch.SyncResult{
			Copied:  1,
			Elapsed: time.Second,
		},
	})

	assert.Equal(t, phaseDone, m.phase)
	assert.Equal(t, 1, m.result.Copied)

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.True(t, m.done)
}

func TestImportOverviewAnalysisProgressUpdatesDashboardState(t *testing.T) {
	t.Parallel()

	m := newImportOverviewModel(context.Background(), testImportOverviewConfig(t))
	m.width = 120
	m.height = 40

	m, _ = m.Update(analysisProgressMsg{
		progress: arch.ImportProgress{
			ProjectsCompleted: 1,
			ProjectsTotal:     2,
			FilesInspected:    3,
			Conversations:     2,
			NewConversations:  1,
			ToUpdate:          1,
			CurrentProject:    "proj-a",
		},
	})

	assert.Equal(t, 3, m.analysisProgress.FilesInspected)
	assert.Equal(t, "proj-a", m.analysisProgress.CurrentProject)
	assert.Equal(t, 2, m.analysisProgress.ProjectsTotal)
}

func TestImportOverviewAnalysisErrorBlocksEnter(t *testing.T) {
	t.Parallel()

	m := newImportOverviewModel(context.Background(), testImportOverviewConfig(t))
	m.width = 120
	m.height = 40

	m, _ = m.Update(analysisFinishedMsg{
		analysis: arch.ImportAnalysis{Err: errors.New("permission denied")},
	})

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.Equal(t, phaseReady, m.phase)
	assert.False(t, m.done)
}

func TestImportOverviewWindowResize(t *testing.T) {
	t.Parallel()

	m := newImportOverviewModel(context.Background(), testImportOverviewConfig(t))

	m, _ = m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})

	assert.Equal(t, 200, m.width)
	assert.Equal(t, 50, m.height)
}

func TestImportOverviewSpinnerTick(t *testing.T) {
	t.Parallel()

	m := newImportOverviewModel(context.Background(), testImportOverviewConfig(t))
	m.width = 120
	m.height = 40

	m, _ = m.Update(spinner.TickMsg{})
	_ = m.View()
}

func TestImportOverviewViewRendersInAllPhases(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)

	t.Run("analyzing", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(context.Background(), cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseAnalyzing
		m.analysisProgress = arch.ImportProgress{
			ProjectsTotal:  4,
			FilesInspected: 42,
			Conversations:  10,
		}

		view := ansi.Strip(m.View())
		assertContainsAll(t, view,
			"Import Workspace",
			"Scanning configured sources",
			"Sources",
			"Files",
			"Conversations",
		)
	})

	t.Run("ready with sync needed", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(context.Background(), cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseReady
		m.analysis = arch.ImportAnalysis{
			ArchiveDir:       cfg.ArchiveDir,
			FilesInspected:   100,
			Projects:         5,
			Conversations:    50,
			NewConversations: 10,
			ToUpdate:         5,
			UpToDate:         35,
			QueuedFiles:      []string{"a.jsonl"},
		}

		view := ansi.Strip(m.View())
		assertContainsAll(t, view, "Ready to Import", "Will import", "Press Enter to import")
	})

	t.Run("ready without sync", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(context.Background(), cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseReady
		m.analysis = arch.ImportAnalysis{
			ArchiveDir: cfg.ArchiveDir,
			UpToDate:   10,
		}

		view := ansi.Strip(m.View())
		assertContainsAll(t, view, "No import needed", "Press Enter to continue")
	})

	t.Run("syncing", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(context.Background(), cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseSyncing
		m.syncActivity = arch.SyncActivitySyncingFiles
		m.total = 5
		m.current = 2
		m.result = arch.SyncResult{Copied: 1, Failed: 1}
		m.currentFile = "test.jsonl"

		view := ansi.Strip(m.View())
		assertContainsAll(t, view, "Importing", "2/5", "Copied", "Failed", "test.jsonl")
	})

	t.Run("rebuilding store", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(context.Background(), cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseSyncing
		m.syncActivity = arch.SyncActivityRebuildingStore
		m.total = 5
		m.current = 5
		m.result = arch.SyncResult{Copied: 5}
		m.currentFile = "stale.jsonl"

		view := ansi.Strip(m.View())
		assertContainsAll(t, view, "Importing", "Rebuilding local store", "Copied")
		assert.NotContains(t, view, "5/5")
		assert.NotContains(t, view, "stale.jsonl")
	})

	t.Run("done", func(t *testing.T) {
		t.Parallel()
		m := newImportOverviewModel(context.Background(), cfg)
		m.width = 120
		m.height = 40
		m.phase = phaseDone
		m.total = 3
		m.result = arch.SyncResult{Copied: 3, Elapsed: time.Second}

		view := ansi.Strip(m.View())
		assertContainsAll(t, view, "Import Workspace", "Complete", "Elapsed", "Press Enter to continue")
	})
}

func TestRenderCenteredImportActivityBlockCentersNonEmptyLines(t *testing.T) {
	t.Parallel()

	width := 80
	got := ansi.Strip(renderCenteredImportActivityBlock(
		[]string{
			"No import needed. Archived files already match the source.",
			renderKeyHint("Press ", "Enter", " to continue"),
		},
		width,
	))

	lines := nonEmptyLines(got)
	require.Len(t, lines, 2)
	assert.Equal(t, width, ansi.StringWidth(lines[0]))
	assert.Equal(t, width, ansi.StringWidth(lines[1]))
	assertCenteredLineContains(t, lines, "No import needed. Archived files already match the source.")
	assertCenteredLineContains(t, lines, "Press Enter to continue")
}

func TestImportOverviewRenderActivityBlockCentersAllStates(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)

	tests := []struct {
		name       string
		model      func() importOverviewModel
		contains   []string
		blockWidth int
	}{
		{
			name: "analyzing",
			model: func() importOverviewModel {
				m := newImportOverviewModel(context.Background(), cfg)
				m.phase = phaseAnalyzing
				return m
			},
			contains: []string{
				"Scanning configured sources",
				"Import becomes available after analysis completes.",
			},
			blockWidth: 80,
		},
		{
			name: "ready with sync needed",
			model: func() importOverviewModel {
				m := newImportOverviewModel(context.Background(), cfg)
				m.phase = phaseReady
				m.analysis = arch.ImportAnalysis{
					ArchiveDir:  cfg.ArchiveDir,
					QueuedFiles: []string{"a.jsonl"},
				}
				return m
			},
			contains: []string{
				"Will import 1 archive files and refresh the local store after confirmation.",
				"Press Enter to import",
			},
			blockWidth: 80,
		},
		{
			name: "ready without sync",
			model: func() importOverviewModel {
				m := newImportOverviewModel(context.Background(), cfg)
				m.phase = phaseReady
				m.analysis = arch.ImportAnalysis{
					ArchiveDir: cfg.ArchiveDir,
					UpToDate:   1,
				}
				return m
			},
			contains: []string{
				"No import needed. Archived files already match the source.",
				"Press Enter to continue",
			},
			blockWidth: 80,
		},
		{
			name: "ready blocked",
			model: func() importOverviewModel {
				m := newImportOverviewModel(context.Background(), cfg)
				m.phase = phaseReady
				m.analysis = arch.ImportAnalysis{Err: errors.New("permission denied")}
				return m
			},
			contains: []string{
				"Import is blocked: permission denied",
				"Press q to quit",
			},
			blockWidth: 80,
		},
		{
			name: "syncing progress",
			model: func() importOverviewModel {
				m := newImportOverviewModel(context.Background(), cfg)
				m.phase = phaseSyncing
				m.syncActivity = arch.SyncActivitySyncingFiles
				m.current = 2
				m.total = 5
				m.currentFile = "test.jsonl"
				return m
			},
			contains: []string{
				"Importing archive files",
				"2/5",
				"Current file test.jsonl",
			},
			blockWidth: 80,
		},
		{
			name: "syncing rebuild",
			model: func() importOverviewModel {
				m := newImportOverviewModel(context.Background(), cfg)
				m.phase = phaseSyncing
				m.syncActivity = arch.SyncActivityRebuildingStore
				return m
			},
			contains: []string{
				"Rebuilding local store",
			},
			blockWidth: 80,
		},
		{
			name: "done",
			model: func() importOverviewModel {
				m := newImportOverviewModel(context.Background(), cfg)
				m.phase = phaseDone
				m.result = arch.SyncResult{Copied: 1, Elapsed: time.Second}
				return m
			},
			contains: []string{
				"Import complete.",
				"Press Enter to continue",
			},
			blockWidth: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lines := nonEmptyLines(ansi.Strip(tt.model().renderActivityBlock(tt.blockWidth)))
			require.NotEmpty(t, lines)

			for _, want := range tt.contains {
				assertCenteredLineContains(t, lines, want)
			}
		})
	}
}

func TestImportOverviewUsesPipelineMessages(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	pipeline := stubImportPipeline{
		analyzeFn: func(_ context.Context, onProgress func(arch.ImportProgress)) (arch.ImportAnalysis, error) {
			onProgress(arch.ImportProgress{
				ProjectsCompleted: 1,
				ProjectsTotal:     2,
				CurrentProject:    "proj-a",
			})
			return arch.ImportAnalysis{
				ArchiveDir:  cfg.ArchiveDir,
				Projects:    2,
				QueuedFiles: []string{"a.jsonl"},
			}, nil
		},
		runFn: func(_ context.Context, onProgress func(arch.SyncProgress)) (arch.SyncResult, error) {
			onProgress(arch.SyncProgress{
				Current:  1,
				Total:    1,
				Copied:   1,
				Activity: arch.SyncActivityRebuildingStore,
			})
			return arch.SyncResult{Copied: 1, StoreBuilt: true}, nil
		},
	}

	m := newImportOverviewModelWithPipeline(context.Background(), cfg, pipeline)

	analysisMsg := requireMsgType[importAnalysisStartedMsg](t, startImportAnalysisCmd(context.Background(), pipeline)())
	m, cmd := m.Update(analysisMsg)
	require.NotNil(t, cmd)
	m, cmd = m.Update(cmd())
	require.NotNil(t, cmd)
	assert.Equal(t, "proj-a", m.analysisProgress.CurrentProject)
	m, _ = m.Update(cmd())
	assert.Equal(t, phaseReady, m.phase)

	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	syncStarted := requireMsgType[importSyncStartedMsg](t, cmd())
	m, cmd = m.Update(syncStarted)
	require.NotNil(t, cmd)
	m, cmd = m.Update(cmd())
	require.NotNil(t, cmd)
	assert.Equal(t, arch.SyncActivityRebuildingStore, m.syncActivity)
	assert.Empty(t, m.currentFile)
	m, _ = m.Update(cmd())
	assert.Equal(t, phaseDone, m.phase)
	assert.True(t, m.result.StoreBuilt)
}

func TestImportOverviewStoreRebuildOnlyShowsSpinnerState(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	pipeline := stubImportPipeline{
		analyzeFn: func(_ context.Context, _ func(arch.ImportProgress)) (arch.ImportAnalysis, error) {
			return arch.ImportAnalysis{
				ArchiveDir:      cfg.ArchiveDir,
				StoreNeedsBuild: true,
			}, nil
		},
		runFn: func(_ context.Context, onProgress func(arch.SyncProgress)) (arch.SyncResult, error) {
			onProgress(arch.SyncProgress{Activity: arch.SyncActivityRebuildingStore})
			return arch.SyncResult{StoreBuilt: true}, nil
		},
	}

	m := newImportOverviewModelWithPipeline(context.Background(), cfg, pipeline)
	m.width = 120
	m.height = 40

	analysisMsg := requireMsgType[importAnalysisStartedMsg](t, startImportAnalysisCmd(context.Background(), pipeline)())
	m, cmd := m.Update(analysisMsg)
	require.NotNil(t, cmd)
	m, _ = m.Update(cmd())
	require.Equal(t, phaseReady, m.phase)

	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	syncStarted := requireMsgType[importSyncStartedMsg](t, cmd())
	m, cmd = m.Update(syncStarted)
	require.NotNil(t, cmd)
	m, _ = m.Update(cmd())

	view := ansi.Strip(m.View())
	assert.Contains(t, view, "Rebuilding local store")
	assert.NotContains(t, view, "0/0")
}

func nonEmptyLines(s string) []string {
	lines := strings.Split(s, "\n")
	nonEmpty := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		nonEmpty = append(nonEmpty, line)
	}
	return nonEmpty
}

func assertCenteredLineContains(t testing.TB, lines []string, want string) {
	t.Helper()

	for _, line := range lines {
		index := strings.Index(line, want)
		if index == -1 {
			continue
		}

		assert.Greater(t, index, 0, "expected %q to be centered in %q", want, line)
		return
	}

	t.Fatalf("expected to find centered line containing %q in %q", want, lines)
}
