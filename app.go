package main

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type viewState int

const (
	viewBrowser viewState = iota
	viewViewer
)

type appModel struct {
	ctx           context.Context
	state         viewState
	browser       browserModel
	viewer        viewerModel
	width, height int
}

func newAppModel(ctx context.Context) appModel {
	return appModel{
		ctx:     ctx,
		state:   viewBrowser,
		browser: newBrowserModel(ctx),
	}
}

func (m appModel) Init() tea.Cmd {
	return m.browser.Init()
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
	}

	switch m.state {
	case viewBrowser:
		return m.updateBrowser(msg)
	case viewViewer:
		return m.updateViewer(msg)
	}

	return m, nil
}

func (m appModel) updateBrowser(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle enter to switch to viewer
		if key.Matches(msg, browserKeys.Enter) && m.browser.list.FilterState() != 1 {
			if meta, ok := m.browser.selectedMeta(); ok {
				if session, ok := m.browser.cachedSession(meta.id); ok {
					return m, func() tea.Msg { return openViewerMsg{session: session} }
				}
				return m, openSessionCmd(m.ctx, meta)
			}
		}

		// Handle quit
		if key.Matches(msg, browserKeys.Quit) && m.browser.list.FilterState() != 1 {
			return m, tea.Quit
		}

	case openViewerMsg:
		m.viewer = newViewerModel(msg.session, m.width, m.height)
		m.state = viewViewer
		return m, m.viewer.Init()
	}

	var cmd tea.Cmd
	m.browser, cmd = m.browser.Update(msg)
	return m, cmd
}

func (m appModel) updateViewer(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(msg, viewerKeys.Back) && !m.viewer.searching {
			m.state = viewBrowser
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewer, cmd = m.viewer.Update(msg)
	return m, cmd
}

func (m appModel) View() string {
	switch m.state {
	case viewBrowser:
		return m.browser.View()
	case viewViewer:
		return m.viewer.View()
	}
	return ""
}
