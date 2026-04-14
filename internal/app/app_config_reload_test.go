package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	arch "github.com/rkuska/carn/internal/archive"
	"github.com/rkuska/carn/internal/config"
	conv "github.com/rkuska/carn/internal/conversation"
)

func TestNewModelInvalidConfigBlocksImportOverview(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	model, err := NewModel(context.Background(), Config{
		SourceDirs:           cfg.SourceDirs,
		ArchiveDir:           cfg.ArchiveDir,
		GlamourStyle:         "dark",
		TimestampFormat:      "2006-01-02 15:04",
		BrowserCacheSize:     20,
		DeepSearchDebounceMs: 200,
		ConfigFilePath:       "/tmp/carn/config.toml",
		ConfigStatus:         config.StatusInvalid,
		ConfigErr:            errors.New("invalid config"),
	})
	require.NoError(t, err)

	app := requireAs[appModel](t, model)
	assert.Nil(t, app.Init())

	nextModel, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = requireAs[appModel](t, nextModel)

	view := ansi.Strip(app.View().Content)
	assert.Contains(t, view, "Config is invalid: invalid config")
	assert.Contains(t, view, "Press c to fix")

	nextModel, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = requireAs[appModel](t, nextModel)
	assert.False(t, app.importOverview.done)
	assert.Equal(t, viewImportOverview, app.state)
}

func TestAppConfigReloadRebuildsRuntimeAndRestartsAnalysis(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	store := &fakeBrowserStore{}
	m := newAppModelWithDeps(context.Background(), cfg, testAppConfig(), store, stubImportPipeline{})

	var analyzedConfigs []arch.Config
	m.pipelineFactory = func(nextCfg arch.Config) importPipeline {
		return stubImportPipeline{
			analyzeFn: func(
				_ context.Context,
				_ func(arch.ImportProgress),
			) (arch.ImportAnalysis, error) {
				analyzedConfigs = append(analyzedConfigs, nextCfg)
				return arch.ImportAnalysis{ArchiveDir: nextCfg.ArchiveDir}, nil
			},
		}
	}

	nextModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = requireAs[appModel](t, nextModel)

	reloaded := config.Config{
		Paths: config.PathsConfig{
			ArchiveDir:      filepath.Join(t.TempDir(), "archive"),
			ClaudeSourceDir: filepath.Join(t.TempDir(), "claude"),
			CodexSourceDir:  filepath.Join(t.TempDir(), "codex"),
			LogFile:         filepath.Join(t.TempDir(), "carn.log"),
		},
		Display: config.DisplayConfig{
			TimestampFormat:  "15:04",
			BrowserCacheSize: 99,
		},
		Search: config.SearchConfig{
			DeepSearchDebounceMs: 750,
		},
	}

	nextModel, cmd := m.Update(configEditedMsg{
		state: config.State{
			Path:   "/tmp/carn/config.toml",
			Status: config.StatusLoaded,
			Config: reloaded,
		},
	})
	m = requireAs[appModel](t, nextModel)

	require.NotNil(t, cmd)
	assert.Equal(t, reloaded.Paths.ArchiveDir, m.cfg.ArchiveDir)
	assert.Equal(t, reloaded.Paths.ClaudeSourceDir, m.cfg.SourceDirs[conv.ProviderClaude])
	assert.Equal(t, reloaded.Paths.CodexSourceDir, m.cfg.SourceDirs[conv.ProviderCodex])
	assert.Equal(t, reloaded.Paths.ArchiveDir, m.browser.ArchiveDir())
	assert.Equal(t, reloaded.Display.TimestampFormat, m.browser.TimestampFormat())
	assert.Equal(t, reloaded.Display.BrowserCacheSize, m.browser.BrowserCacheSize())
	assert.Equal(t, reloaded.Search.DeepSearchDebounceMs, m.browser.DeepSearchDebounceMs())
	assert.Equal(t, phaseAnalyzing, m.importOverview.phase)

	started := requireBatchMsgType[importAnalysisStartedMsg](t, cmd())
	nextModel, cmd = m.Update(started)
	m = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, _ = m.Update(cmd())
	m = requireAs[appModel](t, nextModel)

	assert.Len(t, analyzedConfigs, 1)
	assert.Equal(t, reloaded.Paths.ArchiveDir, analyzedConfigs[0].ArchiveDir)
	assert.Equal(t, phaseReady, m.importOverview.phase)
	assert.Equal(t, reloaded.Paths.ArchiveDir, m.importOverview.analysis.ArchiveDir)
}

func TestAppConfigReloadWithInvalidStateClearsAnalysisAndBlocksImport(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	m := newAppModelWithDeps(context.Background(), cfg, testAppConfig(), &fakeBrowserStore{}, stubImportPipeline{})
	m.importOverview.phase = phaseDone
	m.importOverview.analysis = arch.ImportAnalysis{
		ArchiveDir:       cfg.ArchiveDir,
		QueuedFiles:      []string{"a.jsonl"},
		NewConversations: 1,
	}
	m.importOverview.analysisProgress = arch.ImportProgress{
		FilesInspected: 10,
	}
	m.importOverview.total = 3
	m.importOverview.current = 2
	m.importOverview.currentFile = "a.jsonl"
	m.importOverview.result = arch.SyncResult{Copied: 2}

	nextModel, cmd := m.Update(configEditedMsg{
		state: config.State{
			Path:   "/tmp/carn/config.toml",
			Status: config.StatusInvalid,
			Config: defaultConfig(t),
			Err:    errors.New("bad config"),
		},
	})
	m = requireAs[appModel](t, nextModel)

	assert.Nil(t, cmd)
	assert.Equal(t, phaseReady, m.importOverview.phase)
	assert.Equal(t, config.StatusInvalid, m.importOverview.configStatus)
	require.Error(t, m.importOverview.configErr)
	assert.Empty(t, m.importOverview.analysis.QueuedFiles)
	assert.Zero(t, m.importOverview.analysisProgress.FilesInspected)
	assert.Zero(t, m.importOverview.total)
	assert.Zero(t, m.importOverview.current)
	assert.Empty(t, m.importOverview.currentFile)
	assert.Equal(t, arch.SyncResult{}, m.importOverview.result)

	nextModel, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = requireAs[appModel](t, nextModel)
	view := ansi.Strip(m.View().Content)
	assert.Contains(t, view, "Config is invalid: bad config")
	assert.Contains(t, view, "Press c to fix")

	nextModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = requireAs[appModel](t, nextModel)
	assert.False(t, m.importOverview.done)
}

func TestAppConfigReloadWithUnchangedPathsStillRerunsAnalysis(t *testing.T) {
	t.Parallel()

	cfg := testImportOverviewConfig(t)
	m := newAppModelWithDeps(context.Background(), cfg, testAppConfig(), &fakeBrowserStore{}, stubImportPipeline{})

	analyzeCalls := 0
	m.pipelineFactory = func(nextCfg arch.Config) importPipeline {
		return stubImportPipeline{
			analyzeFn: func(
				_ context.Context,
				_ func(arch.ImportProgress),
			) (arch.ImportAnalysis, error) {
				analyzeCalls++
				return arch.ImportAnalysis{ArchiveDir: nextCfg.ArchiveDir}, nil
			},
		}
	}

	reloaded := config.Config{
		Paths: config.PathsConfig{
			ArchiveDir:      cfg.ArchiveDir,
			ClaudeSourceDir: cfg.SourceDirFor(conv.ProviderClaude),
			CodexSourceDir:  cfg.SourceDirFor(conv.ProviderCodex),
			LogFile:         filepath.Join(t.TempDir(), "carn.log"),
		},
		Display: config.DisplayConfig{
			TimestampFormat:  "3:04PM",
			BrowserCacheSize: 7,
		},
		Search: config.SearchConfig{
			DeepSearchDebounceMs: 10,
		},
	}

	nextModel, cmd := m.Update(configEditedMsg{
		state: config.State{
			Path:   "/tmp/carn/config.toml",
			Status: config.StatusLoaded,
			Config: reloaded,
		},
	})
	m = requireAs[appModel](t, nextModel)

	require.NotNil(t, cmd)
	assert.Equal(t, "3:04PM", m.browser.TimestampFormat())
	assert.Equal(t, 7, m.browser.BrowserCacheSize())
	assert.Equal(t, 10, m.browser.DeepSearchDebounceMs())

	started := requireBatchMsgType[importAnalysisStartedMsg](t, cmd())
	nextModel, cmd = m.Update(started)
	m = requireAs[appModel](t, nextModel)
	require.NotNil(t, cmd)

	nextModel, _ = m.Update(cmd())
	m = requireAs[appModel](t, nextModel)

	assert.Equal(t, 1, analyzeCalls)
	assert.Equal(t, phaseReady, m.importOverview.phase)
}

func requireBatchMsgType[T any](t testing.TB, msg tea.Msg) T {
	t.Helper()

	batch := requireMsgType[tea.BatchMsg](t, msg)
	for _, cmd := range batch {
		candidate := cmd()
		if typed, ok := candidate.(T); ok {
			return typed
		}
	}

	var zero T
	t.Fatalf("expected batch to contain %T", zero)
	return zero
}

func defaultConfig(t *testing.T) config.Config {
	t.Helper()

	home := t.TempDir()

	return config.Config{
		Paths: config.PathsConfig{
			ArchiveDir:      filepath.Join(home, config.DefaultArchiveDir),
			ClaudeSourceDir: filepath.Join(home, config.DefaultClaudeSourceDir),
			CodexSourceDir:  filepath.Join(home, config.DefaultCodexSourceDir),
			LogFile:         filepath.Join(home, config.DefaultLogDir, config.DefaultLogFileName),
		},
		Display: config.DisplayConfig{
			TimestampFormat:  config.DefaultTimestampFormat,
			BrowserCacheSize: config.DefaultBrowserCacheSize,
		},
		Search: config.SearchConfig{
			DeepSearchDebounceMs: config.DefaultDeepSearchDebounceMs,
		},
		Logging: config.LoggingConfig{
			Level:      config.DefaultLogLevel,
			MaxSizeMB:  config.DefaultMaxSizeMB,
			MaxBackups: config.DefaultMaxBackups,
		},
	}
}
