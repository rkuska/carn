package main

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

type viewState int

const (
	viewSync viewState = iota
	viewBrowser
	viewViewer
)

type appModel struct {
	ctx           context.Context
	cfg           archiveConfig
	glamourStyle  string
	state         viewState
	sync          syncModel
	browser       browserModel
	viewer        viewerModel
	width, height int
}

func newAppModel(ctx context.Context, cfg archiveConfig, glamourStyle string) appModel {
	return appModel{
		ctx:          ctx,
		cfg:          cfg,
		glamourStyle: glamourStyle,
		state:        viewSync,
		sync:         newSyncModel(cfg),
		browser:      newBrowserModel(ctx, cfg.archiveDir),
	}
}

func (m appModel) Init() tea.Cmd {
	return m.sync.Init()
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
	}

	switch m.state {
	case viewSync:
		return m.updateSync(msg)
	case viewBrowser:
		return m.updateBrowser(msg)
	case viewViewer:
		return m.updateViewer(msg)
	}

	return m, nil
}

func (m appModel) updateSync(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.sync, cmd = m.sync.Update(msg)

	if m.sync.done {
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

func (m appModel) updateBrowser(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Handle enter to switch to viewer
		if key.Matches(msg, browserKeys.Enter) && m.browser.list.FilterState() != 1 {
			if meta, ok := m.browser.selectedMeta(); ok {
				return m, openSessionCmd(m.ctx, meta)
			}
		}

		// Handle quit
		if key.Matches(msg, browserKeys.Quit) && m.browser.list.FilterState() != 1 {
			return m, tea.Quit
		}

	case openViewerMsg:
		m.viewer = newViewerModel(msg.session, m.glamourStyle, m.width, m.height)
		m.state = viewViewer
		return m, tea.Batch(
			m.viewer.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)
	}

	var cmd tea.Cmd
	m.browser, cmd = m.browser.Update(msg)
	return m, cmd
}

func (m appModel) updateViewer(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		if key.Matches(msg, viewerKeys.Back) && !m.viewer.searching {
			m.state = viewBrowser
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewer, cmd = m.viewer.Update(msg)
	return m, cmd
}

func (m appModel) View() tea.View {
	var content string
	switch m.state {
	case viewSync:
		content = m.sync.View()
	case viewBrowser:
		content = m.browser.View()
	case viewViewer:
		content = m.viewer.View()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
