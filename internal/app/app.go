package app

import (
	"context"

	tea "charm.land/bubbletea/v2"
	arch "github.com/rkuska/carn/internal/archive"
)

type viewState int

const (
	viewImportOverview viewState = iota
	viewBrowser
)

type appModel struct {
	ctx            context.Context
	cfg            arch.Config
	glamourStyle   string
	pipeline       importPipeline
	state          viewState
	importOverview importOverviewModel
	browser        browserModel
	width          int
	height         int
	resyncEvents   <-chan tea.Msg
}

func newAppModelWithDeps(
	ctx context.Context,
	cfg arch.Config,
	glamourStyle string,
	store browserStore,
	pipeline importPipeline,
) appModel {
	return appModel{
		ctx:            ctx,
		cfg:            cfg,
		glamourStyle:   glamourStyle,
		pipeline:       pipeline,
		state:          viewImportOverview,
		importOverview: newImportOverviewModelWithPipeline(ctx, cfg, pipeline),
		browser:        newBrowserModelWithStore(ctx, cfg.ArchiveDir, glamourStyle, store),
	}
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
		return m.updateBrowser(msg)
	}

	return m, nil
}

func (m appModel) updateImportOverview(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.importOverview, cmd = m.importOverview.Update(msg)

	if m.importOverview.done {
		m.state = viewBrowser
		return m, tea.Batch(
			m.browser.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)
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
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
