package app

import (
	"context"

	tea "charm.land/bubbletea/v2"

	appbrowser "github.com/rkuska/carn/internal/app/browser"
	el "github.com/rkuska/carn/internal/app/elements"
	appstats "github.com/rkuska/carn/internal/app/stats"
	arch "github.com/rkuska/carn/internal/archive"
)

type viewState int

const (
	viewImportOverview viewState = iota
	viewBrowser
	viewStats
)

type appModel struct {
	ctx             context.Context
	cfg             arch.Config
	glamourStyle    string
	logFilePath     string
	theme           *el.Theme
	pipeline        importPipeline
	pipelineFactory func(arch.Config) importPipeline
	store           appbrowser.Store
	launcher        appbrowser.SessionLauncher
	state           viewState
	importOverview  importOverviewModel
	browser         appbrowser.Model
	stats           appstats.Model
	width           int
	height          int
	resyncEvents    <-chan tea.Msg
}

func newAppModelWithDeps(
	ctx context.Context,
	cfg arch.Config,
	appCfg Config,
	store appbrowser.Store,
	pipeline importPipeline,
	launchers ...appbrowser.SessionLauncher,
) appModel {
	model := appModel{
		ctx:             ctx,
		cfg:             cfg,
		glamourStyle:    appCfg.GlamourStyle,
		logFilePath:     appCfg.LogFile,
		pipeline:        pipeline,
		pipelineFactory: func(nextCfg arch.Config) importPipeline { return pipeline },
		store:           store,
		state:           viewImportOverview,
	}
	if len(launchers) > 0 && launchers[0] != nil {
		model.launcher = launchers[0]
	}

	model = model.rebuildRuntime(resolveRuntimeConfig(appCfg))
	return model
}

func (m appModel) Init() tea.Cmd {
	return m.importOverview.Init()
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
	}

	switch m.state {
	case viewImportOverview:
		return m.updateImportOverview(msg)
	case viewBrowser:
		if _, ok := msg.(appbrowser.OpenStatsRequestedMsg); ok {
			return m.updateStats(msg)
		}
		return m.updateBrowser(msg)
	case viewStats:
		return m.updateStats(msg)
	}

	return m, nil
}

func (m appModel) updateImportOverview(msg tea.Msg) (tea.Model, tea.Cmd) {
	if next, cmd, handled := m.handleImportOverviewConfigMsg(msg); handled {
		return next, cmd
	}

	var cmd tea.Cmd
	m.importOverview, cmd = m.importOverview.Update(msg)

	if m.importOverview.done {
		m.state = viewBrowser
		var cmds []tea.Cmd
		appendCmd(&cmds, m.browser.Init())
		appendCmd(&cmds, func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		})
		if n, ok := malformedDataNotification(m.importOverview.result.MalformedData); ok {
			var notify tea.Cmd
			m.browser, notify = m.browser.SetNotification(n)
			appendCmd(&cmds, notify)
		} else if n, ok := driftNotification(m.importOverview.result.Drift); ok {
			var notify tea.Cmd
			m.browser, notify = m.browser.SetNotification(n)
			appendCmd(&cmds, notify)
		}
		return m, tea.Batch(cmds...)
	}

	return m, cmd
}

func (m appModel) View() tea.View {
	var content string
	switch m.state {
	case viewImportOverview:
		content = m.importOverview.View()
	case viewBrowser:
		content = m.browser.View()
	case viewStats:
		content = m.stats.View()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
