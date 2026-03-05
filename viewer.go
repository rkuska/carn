package main

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"
)

type viewerModel struct {
	viewport      viewport.Model
	session       sessionFull
	opts          transcriptOptions
	width, height int
	searchInput   textinput.Model
	searching     bool
	searchQuery   string
	matchIndices  []int // line indices of matches
	currentMatch  int
	statusText    string
	rawContent    string // unrendered transcript
}

func newViewerModel(session sessionFull, width, height int) viewerModel {
	vp := viewport.New(viewport.WithWidth(width), viewport.WithHeight(height-3))
	vp.Style = lipgloss.NewStyle().Padding(0, 1)

	ti := textinput.New()
	ti.Prompt = "/"
	ti.CharLimit = 100

	m := viewerModel{
		viewport:    vp,
		session:     session,
		opts:        transcriptOptions{},
		width:       width,
		height:      height,
		searchInput: ti,
	}
	m.renderContent()
	return m
}

func (m viewerModel) Init() tea.Cmd {
	return nil
}

func (m viewerModel) Update(msg tea.Msg) (viewerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.searching {
			return m.handleSearchKey(msg)
		}
		cmd := m.handleKey(msg, &cmds)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetWidth(msg.Width)
		m.viewport.SetHeight(msg.Height - 3)
		m.renderContent()

	case statusMsg:
		m.statusText = msg.text
		cmds = append(cmds, clearStatusAfter(3*time.Second))

	case clearStatusMsg:
		m.statusText = ""
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func toggleLabel(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func (m *viewerModel) handleKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) tea.Cmd {
	switch {
	case key.Matches(msg, viewerKeys.ToggleThinking):
		m.opts.showThinking = !m.opts.showThinking
		m.renderContent()
		m.statusText = fmt.Sprintf("Thinking: %s", toggleLabel(m.opts.showThinking))
		*cmds = append(*cmds, clearStatusAfter(2*time.Second))

	case key.Matches(msg, viewerKeys.ToggleTools):
		m.opts.showTools = !m.opts.showTools
		m.renderContent()
		m.statusText = fmt.Sprintf("Tools: %s", toggleLabel(m.opts.showTools))
		*cmds = append(*cmds, clearStatusAfter(2*time.Second))

	case key.Matches(msg, viewerKeys.ToggleToolResults):
		m.opts.showToolResults = !m.opts.showToolResults
		m.renderContent()
		m.statusText = fmt.Sprintf("Tool results: %s", toggleLabel(m.opts.showToolResults))
		*cmds = append(*cmds, clearStatusAfter(2*time.Second))

	case key.Matches(msg, viewerKeys.ToggleSidechain):
		m.opts.hideSidechain = !m.opts.hideSidechain
		m.renderContent()
		label := "shown"
		if m.opts.hideSidechain {
			label = "hidden"
		}
		m.statusText = fmt.Sprintf("Sidechain: %s", label)
		*cmds = append(*cmds, clearStatusAfter(2*time.Second))

	case key.Matches(msg, viewerKeys.Search):
		m.searching = true
		m.searchInput.Focus()
		return textinput.Blink

	case key.Matches(msg, viewerKeys.NextMatch):
		m.jumpToMatch(1)

	case key.Matches(msg, viewerKeys.PrevMatch):
		m.jumpToMatch(-1)

	case key.Matches(msg, viewerKeys.Copy):
		return copyTranscriptCmd(m.session, m.opts)

	case key.Matches(msg, viewerKeys.Export):
		return exportTranscriptCmd(m.session, m.opts)

	case key.Matches(msg, viewerKeys.Editor):
		return openInEditorCmd(m.session.meta.filePath)

	case key.Matches(msg, viewerKeys.Resume):
		return resumeSessionCmd(m.session.meta.id)
	}

	return nil
}

func (m viewerModel) handleSearchKey(msg tea.KeyPressMsg) (viewerModel, tea.Cmd) {
	if msg.Code == tea.KeyEnter {
		m.searching = false
		m.searchQuery = m.searchInput.Value()
		m.searchInput.Blur()
		m.performSearch()
		return m, nil
	}

	if msg.Code == tea.KeyEscape {
		m.searching = false
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m viewerModel) View() string {
	header := m.headerView()
	content := m.viewport.View()
	footer := m.footerView()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

func (m viewerModel) headerView() string {
	title := styleTitle.Render(fmt.Sprintf(
		"%s / %s",
		m.session.meta.project.displayName,
		m.session.meta.slug,
	))
	date := styleSubtitle.Render(m.session.meta.timestamp.Format("2006-01-02 15:04"))
	return lipgloss.JoinHorizontal(lipgloss.Bottom, title, "  ", date)
}

func (m viewerModel) footerView() string {
	if m.searching {
		return m.searchInput.View()
	}

	var parts []string

	// Scroll position
	parts = append(parts, fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))

	// Toggle status
	if m.opts.showThinking {
		parts = append(parts, styleToolCall.Render("[thinking]"))
	}
	if m.opts.showTools {
		parts = append(parts, styleToolCall.Render("[tools]"))
	}
	if m.opts.showToolResults {
		parts = append(parts, styleToolCall.Render("[results]"))
	}
	if m.opts.hideSidechain {
		parts = append(parts, styleToolCall.Render("[no-sidechain]"))
	}

	// Search matches
	if m.searchQuery != "" {
		parts = append(parts, fmt.Sprintf("/%s (%d/%d)",
			m.searchQuery, m.currentMatch+1, len(m.matchIndices)))
	}

	if m.statusText != "" {
		parts = append(parts, m.statusText)
	}

	return styleStatusBar.Width(m.width).Render(strings.Join(parts, "  "))
}

func (m *viewerModel) renderContent() {
	m.rawContent = renderTranscript(m.session, m.opts)

	// Render markdown with glamour
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(m.width-4),
	)
	if err != nil {
		m.viewport.SetContent(m.rawContent)
		return
	}

	rendered, err := renderer.Render(m.rawContent)
	if err != nil {
		m.viewport.SetContent(m.rawContent)
		return
	}

	m.viewport.SetContent(rendered)
}

func (m *viewerModel) performSearch() {
	m.matchIndices = nil
	m.currentMatch = 0

	if m.searchQuery == "" {
		return
	}

	lines := strings.Split(m.viewport.View(), "\n")
	queryLower := strings.ToLower(m.searchQuery)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), queryLower) {
			m.matchIndices = append(m.matchIndices, i)
		}
	}

	if len(m.matchIndices) > 0 {
		m.viewport.SetYOffset(m.matchIndices[0])
	}
}

func (m *viewerModel) jumpToMatch(delta int) {
	if len(m.matchIndices) == 0 {
		return
	}

	m.currentMatch = (m.currentMatch + delta + len(m.matchIndices)) % len(m.matchIndices)
	m.viewport.SetYOffset(m.matchIndices[m.currentMatch])
}
