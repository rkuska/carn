package app

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	appbrowser "github.com/rkuska/carn/internal/app/browser"
	arch "github.com/rkuska/carn/internal/archive"
	"github.com/rkuska/carn/internal/config"
	conv "github.com/rkuska/carn/internal/conversation"
)

type configEditRequestedMsg struct{}

type configEditedMsg struct {
	state config.State
	err   error
}

func requestConfigEditCmd() tea.Cmd {
	return func() tea.Msg {
		return configEditRequestedMsg{}
	}
}

func createAndEditConfigCmd(path, template string) tea.Cmd {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return func() tea.Msg {
			return configEditedMsg{err: fmt.Errorf("os.MkdirAll: %w", err)}
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		_, writeErr := f.Write([]byte(template))
		closeErr := f.Close()
		if writeErr != nil {
			return func() tea.Msg {
				return configEditedMsg{err: fmt.Errorf("writeTemplate: %w", writeErr)}
			}
		}
		if closeErr != nil {
			return func() tea.Msg {
				return configEditedMsg{err: fmt.Errorf("f.Close: %w", closeErr)}
			}
		}
	} else if !os.IsExist(err) {
		return func() tea.Msg {
			return configEditedMsg{err: fmt.Errorf("os.OpenFile: %w", err)}
		}
	}

	editorCmd := newEditorCmd(path)
	return tea.ExecProcess(editorCmd, func(err error) tea.Msg {
		if err != nil {
			return configEditedMsg{err: fmt.Errorf("editor: %w", err)}
		}

		state, loadErr := config.LoadState()
		if loadErr != nil {
			return configEditedMsg{err: fmt.Errorf("config.LoadState: %w", loadErr)}
		}
		return configEditedMsg{state: state}
	})
}

func (m appModel) handleImportOverviewConfigMsg(msg tea.Msg) (appModel, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case configEditRequestedMsg:
		return m, createAndEditConfigCmd(
			m.importOverview.configFilePath,
			config.DefaultTemplate(),
		), true
	case configEditedMsg:
		state := msg.state
		if msg.err != nil {
			state = config.State{
				Path:   m.importOverview.configFilePath,
				Status: config.StatusInvalid,
				Config: m.currentConfig(),
				Err:    msg.err,
			}
		}
		next, cmd := m.applyConfigState(state)
		return next, cmd, true
	}

	return m, nil, false
}

func (m appModel) applyConfigState(state config.State) (appModel, tea.Cmd) {
	appCfg := configStateToAppConfig(state, m.glamourStyle)
	m = m.rebuildRuntime(appCfg)
	if state.Status == config.StatusInvalid {
		return m, nil
	}
	return m, m.importOverview.Init()
}

func (m appModel) rebuildRuntime(appCfg Config) appModel {
	m.cfg = arch.Config{
		SourceDirs: appCfg.SourceDirs,
		ArchiveDir: appCfg.ArchiveDir,
	}
	m.logFilePath = appCfg.LogFile
	m.pipeline = m.pipelineFactory(m.cfg)
	m.importOverview = newImportOverviewModelWithPipelineConfig(
		m.ctx,
		m.cfg,
		m.pipeline,
		appCfg.ConfigFilePath,
		appCfg.ConfigStatus,
		appCfg.ConfigErr,
		appCfg.LogFile,
	)
	m.importOverview.width = m.width
	m.importOverview.height = m.height
	m.importOverview.progress.SetWidth(m.width / 3)
	m.browser = appbrowser.NewModelWithStore(
		m.ctx,
		appCfg.ArchiveDir,
		appCfg.LogFile,
		appCfg.GlamourStyle,
		appCfg.TimestampFormat,
		appCfg.BrowserCacheSize,
		appCfg.DeepSearchDebounceMs,
		m.store,
		m.launcher,
	)
	m.browser = m.browser.SetSize(m.width, m.height)
	return m
}

func (m appModel) currentConfig() config.Config {
	return config.Config{
		Paths: config.PathsConfig{
			ArchiveDir:      m.cfg.ArchiveDir,
			ClaudeSourceDir: m.cfg.SourceDirFor(conv.ProviderClaude),
			CodexSourceDir:  m.cfg.SourceDirFor(conv.ProviderCodex),
			LogFile:         m.logFilePath,
		},
		Display: config.DisplayConfig{
			TimestampFormat:  m.browser.TimestampFormat(),
			BrowserCacheSize: m.browser.BrowserCacheSize(),
		},
		Search: config.SearchConfig{
			DeepSearchDebounceMs: m.browser.DeepSearchDebounceMs(),
		},
	}
}

func configStateToAppConfig(state config.State, glamourStyle string) Config {
	archiveCfg := state.Config.ArchiveConfig()
	return Config{
		SourceDirs:           archiveCfg.SourceDirs,
		ArchiveDir:           archiveCfg.ArchiveDir,
		LogFile:              state.Config.Paths.LogFile,
		GlamourStyle:         glamourStyle,
		TimestampFormat:      state.Config.Display.TimestampFormat,
		BrowserCacheSize:     state.Config.Display.BrowserCacheSize,
		DeepSearchDebounceMs: state.Config.Search.DeepSearchDebounceMs,
		ConfigFilePath:       state.Path,
		ConfigStatus:         state.Status,
		ConfigErr:            state.Err,
	}
}
