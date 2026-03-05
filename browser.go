package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	previewCacheSize = 20
	previewMessages  = 6
)

type focusArea int

const (
	focusList focusArea = iota
	focusPreview
)

type browserModel struct {
	ctx           context.Context
	archiveDir    string
	list          list.Model
	preview       viewport.Model
	focus         focusArea
	allSessions   []sessionMeta
	width, height int
	statusText    string
	deepSearch    bool
	previewCache  map[string]string      // session ID -> rendered preview
	sessionCache  map[string]sessionFull // session ID -> parsed session
	lastPreviewID string

	// for LRU eviction
	cacheOrder []string
}

func newBrowserModel(ctx context.Context, archiveDir string) browserModel {
	delegate := newDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Claude Sessions"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	vp := viewport.New(0, 0)

	return browserModel{
		ctx:          ctx,
		archiveDir:   archiveDir,
		list:         l,
		preview:      vp,
		focus:        focusList,
		previewCache: make(map[string]string, previewCacheSize),
		sessionCache: make(map[string]sessionFull, previewCacheSize),
	}
}

func (m browserModel) Init() tea.Cmd {
	return loadSessionsCmd(m.ctx, m.archiveDir)
}

func (m browserModel) Update(msg tea.Msg) (browserModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		cmd := m.handleKey(msg, &cmds)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		m.previewCache = make(map[string]string, previewCacheSize)
		m.lastPreviewID = ""

	case sessionsLoadedMsg:
		m.allSessions = msg.sessions
		items := filterMainSessionItems(msg.sessions)
		cmd := m.list.SetItems(items)
		cmds = append(cmds, cmd)
		m.updatePreview()

	case sessionsLoadErrorMsg:
		m.statusText = fmt.Sprintf("Error: %v", msg.err)
		cmds = append(cmds, clearStatusAfter(5*time.Second))

	case sessionParsedMsg:
		preview := renderPreview(msg.session, previewMessages, m.preview.Width)
		m.previewCache[msg.session.meta.id] = preview
		m.sessionCache[msg.session.meta.id] = msg.session
		m.addToCache(msg.session.meta.id)
		// Only update preview if this is still the selected session
		if selected, ok := m.selectedMeta(); ok && selected.id == msg.session.meta.id {
			m.preview.SetContent(preview)
		}

	case deepSearchResultMsg:
		items := filterMainSessionItems(msg.sessions)
		cmd := m.list.SetItems(items)
		cmds = append(cmds, cmd)
		m.statusText = fmt.Sprintf("Deep search: %d results", len(items))
		cmds = append(cmds, clearStatusAfter(3*time.Second))

	case statusMsg:
		m.statusText = msg.text
		cmds = append(cmds, clearStatusAfter(3*time.Second))

	case clearStatusMsg:
		m.statusText = ""
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Update preview on cursor change
	m.checkPreviewUpdate(&cmds)

	// Update viewport if focused
	if m.focus == focusPreview {
		m.preview, cmd = m.preview.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *browserModel) handleKey(msg tea.KeyMsg, cmds *[]tea.Cmd) tea.Cmd {
	// Don't handle keys when list is filtering
	if m.list.FilterState() == list.Filtering {
		return nil
	}

	switch {
	case key.Matches(msg, browserKeys.Tab):
		if m.focus == focusList {
			m.focus = focusPreview
		} else {
			m.focus = focusList
		}
		return nil

	case key.Matches(msg, browserKeys.DeepSearch):
		m.deepSearch = !m.deepSearch
		if m.deepSearch {
			m.statusText = "Deep search: loading..."
			filterVal := m.list.FilterValue()
			return deepSearchCmd(m.ctx, filterVal, m.allSessions)
		}
		// Reset to all sessions
		items := filterMainSessionItems(m.allSessions)
		*cmds = append(*cmds, m.list.SetItems(items))
		m.statusText = "Deep search disabled"
		*cmds = append(*cmds, clearStatusAfter(2*time.Second))
		return nil

	case key.Matches(msg, browserKeys.Copy):
		if meta, ok := m.selectedMeta(); ok {
			return copyFromMetaCmd(m.ctx, meta)
		}

	case key.Matches(msg, browserKeys.Export):
		if meta, ok := m.selectedMeta(); ok {
			return exportFromMetaCmd(m.ctx, meta)
		}

	case key.Matches(msg, browserKeys.Editor):
		if meta, ok := m.selectedMeta(); ok {
			return openInEditorCmd(meta.filePath)
		}

	case key.Matches(msg, browserKeys.Resume):
		if meta, ok := m.selectedMeta(); ok {
			return resumeSessionCmd(meta.id)
		}
	}

	return nil
}

func (m browserModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	listWidth := m.width * 6 / 10
	previewWidth := m.width - listWidth - 3 // 3 for borders/gap

	// Status bar
	status := m.statusBar()

	// Render list pane
	m.list.SetSize(listWidth, m.height-2)
	listView := m.list.View()

	// Render preview pane
	m.preview.Width = previewWidth - 2
	m.preview.Height = m.height - 4
	previewBorder := stylePreviewBorder
	if m.focus == focusPreview {
		previewBorder = previewBorder.BorderForeground(colorAccent)
	}
	previewView := previewBorder.
		Width(previewWidth).
		Height(m.height - 2).
		Render(m.preview.View())

	// Join horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, listView, " ", previewView)

	return lipgloss.JoinVertical(lipgloss.Left, content, status)
}

func (m *browserModel) updateLayout() {
	listWidth := m.width * 6 / 10
	previewWidth := m.width - listWidth - 3

	m.list.SetSize(listWidth, m.height-2)
	m.preview.Width = previewWidth - 2
	m.preview.Height = m.height - 4
}

func (m *browserModel) selectedMeta() (sessionMeta, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return sessionMeta{}, false
	}
	meta, ok := item.(sessionMeta)
	return meta, ok
}

func (m *browserModel) updatePreview() {
	meta, ok := m.selectedMeta()
	if !ok {
		m.preview.SetContent("No session selected")
		return
	}

	if cached, ok := m.previewCache[meta.id]; ok {
		m.preview.SetContent(cached)
		return
	}

	// Re-render from cached session if available
	if session, ok := m.sessionCache[meta.id]; ok {
		preview := renderPreview(session, previewMessages, m.preview.Width)
		m.previewCache[meta.id] = preview
		m.preview.SetContent(preview)
		return
	}

	m.preview.SetContent("Loading preview...")
}

func (m *browserModel) checkPreviewUpdate(cmds *[]tea.Cmd) {
	meta, ok := m.selectedMeta()
	if !ok {
		return
	}

	if meta.id == m.lastPreviewID {
		return
	}
	m.lastPreviewID = meta.id

	if cached, ok := m.previewCache[meta.id]; ok {
		m.preview.SetContent(cached)
		return
	}

	// Re-render from cached session if available (e.g. after resize)
	if session, ok := m.sessionCache[meta.id]; ok {
		preview := renderPreview(session, previewMessages, m.preview.Width)
		m.previewCache[meta.id] = preview
		m.preview.SetContent(preview)
		return
	}

	m.preview.SetContent("Loading preview...")
	*cmds = append(*cmds, parseSessionCmd(m.ctx, meta))
}

func (m *browserModel) addToCache(id string) {
	if slices.Contains(m.cacheOrder, id) {
		return
	}

	m.cacheOrder = append(m.cacheOrder, id)

	// Evict oldest if over capacity
	for len(m.cacheOrder) > previewCacheSize {
		evictID := m.cacheOrder[0]
		m.cacheOrder = m.cacheOrder[1:]
		delete(m.previewCache, evictID)
		delete(m.sessionCache, evictID)
	}
}

func (m browserModel) cachedSession(id string) (sessionFull, bool) {
	s, ok := m.sessionCache[id]
	return s, ok
}

func filterMainSessionItems(sessions []sessionMeta) []list.Item {
	items := make([]list.Item, 0, len(sessions))
	for _, s := range sessions {
		if !s.isSubagent {
			items = append(items, s)
		}
	}
	return items
}

func (m browserModel) statusBar() string {
	var parts []string
	if m.deepSearch {
		parts = append(parts, styleToolCall.Render("[DEEP SEARCH]"))
	}
	if m.statusText != "" {
		parts = append(parts, m.statusText)
	}

	mainCount := 0
	for _, s := range m.allSessions {
		if !s.isSubagent {
			mainCount++
		}
	}
	info := fmt.Sprintf("%d sessions", mainCount)
	if meta, ok := m.selectedMeta(); ok {
		info = fmt.Sprintf("%s | %s", info, meta.project.displayName)
	}
	parts = append(parts, info)

	return styleStatusBar.Width(m.width).Render(strings.Join(parts, "  "))
}
