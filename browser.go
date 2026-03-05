package main

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	ctx                   context.Context
	archiveDir            string
	list                  list.Model
	preview               viewport.Model
	focus                 focusArea
	allConversations      []conversation
	width, height         int
	mainConversationCount int
	statusText            string
	deepSearch            bool
	previewCache          map[string]string      // conv ID -> rendered preview
	sessionCache          map[string]sessionFull // conv ID -> parsed session
	searchIndex           map[string]string      // conv ID -> lower-cased searchable blob
	lastPreviewID         string

	// for LRU eviction
	cacheOrder []string
}

func newBrowserModel(ctx context.Context, archiveDir string) browserModel {
	delegate := newDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			browserKeys.Enter,
			browserKeys.Tab,
			browserKeys.DeepSearch,
			browserKeys.Copy,
		}
	}
	l.Styles.DefaultFilterCharacterMatch = lipgloss.NewStyle().
		Background(colorHighlight).
		Bold(true)

	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			browserKeys.Enter,
			browserKeys.Tab,
			browserKeys.DeepSearch,
			browserKeys.Resume,
			browserKeys.Copy,
			browserKeys.Export,
			browserKeys.Editor,
		}
	}

	vp := viewport.New()

	return browserModel{
		ctx:          ctx,
		archiveDir:   archiveDir,
		list:         l,
		preview:      vp,
		focus:        focusList,
		previewCache: make(map[string]string, previewCacheSize),
		sessionCache: make(map[string]sessionFull, previewCacheSize),
		searchIndex:  make(map[string]string, previewCacheSize),
	}
}

func (m browserModel) Init() tea.Cmd {
	return loadSessionsCmd(m.ctx, m.archiveDir)
}

func (m browserModel) Update(msg tea.Msg) (browserModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
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

	case conversationsLoadedMsg:
		m.allConversations = msg.conversations
		mainConvs := filterMainConversations(msg.conversations)
		m.mainConversationCount = len(mainConvs)
		m.searchIndex = make(map[string]string, previewCacheSize)
		cmd := m.list.SetItems(conversationItems(mainConvs))
		cmds = append(cmds, cmd)
		m.updatePreview()

	case sessionsLoadErrorMsg:
		m.statusText = fmt.Sprintf("Error: %v", msg.err)
		cmds = append(cmds, clearStatusAfter(5*time.Second))

	case sessionParsedMsg:
		preview := renderPreview(msg.session, previewMessages, m.preview.Width())
		m.previewCache[msg.session.meta.id] = preview
		m.sessionCache[msg.session.meta.id] = msg.session
		m.searchIndex[msg.session.meta.id] = buildSessionSearchBlob(msg.session)
		m.addToCache(msg.session.meta.id)
		// Only update preview if this is still the selected conversation
		if selected, ok := m.selectedConversation(); ok && selected.id() == msg.session.meta.id {
			m.preview.SetContent(preview)
		}

	case deepSearchResultMsg:
		maps.Copy(m.searchIndex, msg.indexed)
		items := conversationItems(msg.conversations)
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

func (m *browserModel) handleKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) tea.Cmd {
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
			mainConvs := filterMainConversations(m.allConversations)
			return deepSearchCmd(
				m.ctx,
				filterVal,
				mainConvs,
				m.cloneSearchIndex(),
				m.cloneSessionCache(),
			)
		}
		// Reset to all conversations
		items := conversationItems(filterMainConversations(m.allConversations))
		*cmds = append(*cmds, m.list.SetItems(items))
		m.statusText = "Deep search disabled"
		*cmds = append(*cmds, clearStatusAfter(2*time.Second))
		return nil

	case key.Matches(msg, browserKeys.Copy):
		if conv, ok := m.selectedConversation(); ok {
			if session, cached := m.cachedSession(conv.id()); cached {
				return copyTranscriptCmd(session, transcriptOptions{})
			}
			return copyFromConversationCmd(m.ctx, conv)
		}

	case key.Matches(msg, browserKeys.Export):
		if conv, ok := m.selectedConversation(); ok {
			if session, cached := m.cachedSession(conv.id()); cached {
				return exportTranscriptCmd(session, transcriptOptions{})
			}
			return exportFromConversationCmd(m.ctx, conv)
		}

	case key.Matches(msg, browserKeys.Editor):
		if conv, ok := m.selectedConversation(); ok {
			return openInEditorCmd(conv.latestFilePath())
		}

	case key.Matches(msg, browserKeys.Resume):
		if conv, ok := m.selectedConversation(); ok {
			return resumeSessionCmd(conv.resumeID())
		}
	}

	return nil
}

func (m browserModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	listBoxWidth := m.width * 6 / 10
	previewBoxWidth := m.width - listBoxWidth - 1 // 1 for gap

	// Status bar
	status := m.statusBar()

	// Render list pane with embedded title in frame border
	listTopBorder := renderBorderTop("Claude Sessions", listBoxWidth, colorAccent, colorAccent)
	listBody := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		BorderForeground(colorAccent).
		Width(listBoxWidth).
		Height(m.height - 3).
		Render(m.list.View())
	listView := listTopBorder + "\n" + listBody

	// Render preview pane with embedded title in frame border
	previewTopBorder := renderBorderTop("Preview", previewBoxWidth, colorPrimary, colorPrimary)
	previewBody := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		BorderForeground(colorPrimary).
		Width(previewBoxWidth).
		Height(m.height - 3).
		Render(m.preview.View())
	previewView := previewTopBorder + "\n" + previewBody

	// Join horizontally with gap
	content := lipgloss.JoinHorizontal(lipgloss.Top, listView, " ", previewView)

	return lipgloss.JoinVertical(lipgloss.Left, content, status)
}

func (m *browserModel) updateLayout() {
	listBoxWidth := m.width * 6 / 10
	previewBoxWidth := m.width - listBoxWidth - 1

	// List inner dimensions (inside border: box - 2 border chars, no built-in title)
	m.list.SetSize(listBoxWidth-2, m.height-3)
	// Preview inner dimensions (inside border: box - 2 border chars)
	m.preview.SetWidth(previewBoxWidth - 2)
	m.preview.SetHeight(m.height - 3)
}

func (m *browserModel) selectedConversation() (conversation, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return conversation{}, false
	}
	conv, ok := item.(conversation)
	return conv, ok
}

func (m *browserModel) updatePreview() {
	conv, ok := m.selectedConversation()
	if !ok {
		m.preview.SetContent("No session selected")
		return
	}

	if cached, ok := m.previewCache[conv.id()]; ok {
		m.preview.SetContent(cached)
		return
	}

	// Re-render from cached session if available
	if session, ok := m.sessionCache[conv.id()]; ok {
		preview := renderPreview(session, previewMessages, m.preview.Width())
		m.previewCache[conv.id()] = preview
		m.preview.SetContent(preview)
		return
	}

	m.preview.SetContent("Loading preview...")
}

func (m *browserModel) checkPreviewUpdate(cmds *[]tea.Cmd) {
	conv, ok := m.selectedConversation()
	if !ok {
		return
	}

	if conv.id() == m.lastPreviewID {
		return
	}
	m.lastPreviewID = conv.id()

	if cached, ok := m.previewCache[conv.id()]; ok {
		m.preview.SetContent(cached)
		return
	}

	// Re-render from cached session if available (e.g. after resize)
	if session, ok := m.sessionCache[conv.id()]; ok {
		preview := renderPreview(session, previewMessages, m.preview.Width())
		m.previewCache[conv.id()] = preview
		m.preview.SetContent(preview)
		return
	}

	m.preview.SetContent("Loading preview...")
	*cmds = append(*cmds, parseConversationCmd(m.ctx, conv))
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

func (m browserModel) cloneSearchIndex() map[string]string {
	out := make(map[string]string, len(m.searchIndex))
	maps.Copy(out, m.searchIndex)
	return out
}

func (m browserModel) cloneSessionCache() map[string]sessionFull {
	out := make(map[string]sessionFull, len(m.sessionCache))
	maps.Copy(out, m.sessionCache)
	return out
}

func filterMainConversations(convs []conversation) []conversation {
	items := make([]conversation, 0, len(convs))
	for _, c := range convs {
		if !c.isSubagent() {
			items = append(items, c)
		}
	}
	return items
}

func conversationItems(convs []conversation) []list.Item {
	items := make([]list.Item, 0, len(convs))
	for _, c := range convs {
		items = append(items, c)
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

	info := fmt.Sprintf("%d sessions", m.mainConversationCount)
	if conv, ok := m.selectedConversation(); ok {
		info = fmt.Sprintf("%s | %s", info, conv.project.displayName)
	}
	parts = append(parts, info)

	return styleStatusBar.Width(m.width).Render(strings.Join(parts, "  "))
}
