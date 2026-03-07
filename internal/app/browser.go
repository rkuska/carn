package app

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const browserCacheSize = 20

type focusArea int

const (
	focusList focusArea = iota
	focusTranscript
)

type transcriptMode int

const (
	transcriptClosed transcriptMode = iota
	transcriptSplit
	transcriptFullscreen
)

type browserModel struct {
	ctx                   context.Context
	archiveDir            string
	glamourStyle          string
	list                  list.Model
	focus                 focusArea
	transcriptMode        transcriptMode
	allConversations      []conversation
	width                 int
	height                int
	mainConversationCount int
	notification          notification
	deepSearch            bool
	sessionCache          map[string]sessionFull
	transcriptCache       map[string]sessionFull
	searchIndex           map[string]string
	openConversationID    string
	loadingConversationID string
	helpOpen              bool
	pendingListGotoTopKey bool
	cacheOrder            []string
	viewer                viewerModel
}

func newBrowserModel(ctx context.Context, archiveDir, glamourStyle string) browserModel {
	delegate := newDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.DefaultFilterCharacterMatch = lipgloss.NewStyle().
		Background(colorHighlight).
		Bold(true)
	l.DisableQuitKeybindings()

	keyMap := l.KeyMap
	keyMap.GoToStart = key.NewBinding(
		key.WithKeys("home"),
		key.WithHelp("home", "go to start"),
	)
	keyMap.GoToEnd = key.NewBinding(
		key.WithKeys("end", "G"),
		key.WithHelp("G/end", "go to end"),
	)
	keyMap.NextPage = key.NewBinding(
		key.WithKeys("pgdown", "ctrl+f"),
		key.WithHelp("ctrl+f/pgdn", "next page"),
	)
	keyMap.PrevPage = key.NewBinding(
		key.WithKeys("pgup", "ctrl+b"),
		key.WithHelp("ctrl+b/pgup", "prev page"),
	)
	keyMap.ShowFullHelp.SetEnabled(false)
	keyMap.CloseFullHelp.SetEnabled(false)
	l.KeyMap = keyMap

	return browserModel{
		ctx:             ctx,
		archiveDir:      archiveDir,
		glamourStyle:    glamourStyle,
		list:            l,
		focus:           focusList,
		transcriptMode:  transcriptClosed,
		sessionCache:    make(map[string]sessionFull, browserCacheSize),
		transcriptCache: make(map[string]sessionFull, browserCacheSize),
		searchIndex:     make(map[string]string, browserCacheSize),
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

	case conversationsLoadedMsg:
		m.allConversations = msg.conversations
		mainConvs := filterMainConversations(msg.conversations)
		m.mainConversationCount = len(mainConvs)
		m.searchIndex = make(map[string]string, browserCacheSize)
		cmds = append(cmds, m.list.SetItems(conversationItems(mainConvs)))
		m.syncTranscriptSelection(&cmds)

	case sessionsLoadErrorMsg:
		m.setNotification(
			errorNotification(fmt.Sprintf("load sessions failed: %v", msg.err)).notification,
			&cmds,
		)

	case openViewerMsg:
		if msg.conversationID == m.loadingConversationID && msg.conversationID != "" {
			m.installViewer(msg.session, msg.conversation)
		}

	case deepSearchResultMsg:
		maps.Copy(m.searchIndex, msg.indexed)
		cmds = append(cmds, m.list.SetItems(conversationItems(msg.conversations)))
		m.setNotification(
			infoNotification(fmt.Sprintf("deep search: %d results", len(msg.conversations))).notification,
			&cmds,
		)
		m.syncTranscriptSelection(&cmds)

	case notificationMsg:
		m.setNotification(msg.notification, &cmds)

	case clearNotificationMsg:
		m.notification = notification{}
		if m.viewer.notification.text != "" {
			m.viewer.notification = notification{}
		}
	}

	_, isKey := msg.(tea.KeyPressMsg)

	if m.shouldUpdateList(isKey) {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.syncTranscriptSelection(&cmds)
	}

	if m.shouldUpdateViewer(isKey) {
		var cmd tea.Cmd
		previousNotification := m.viewer.notification
		m.viewer, cmd = m.viewer.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if m.viewer.notification != previousNotification {
			m.notification = m.viewer.notification
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *browserModel) handleKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) tea.Cmd {
	if m.helpOpen {
		switch {
		case key.Matches(msg, browserKeys.Help), key.Matches(msg, browserKeys.Close):
			m.helpOpen = false
		}
		return nil
	}

	if m.transcriptVisible() {
		switch {
		case key.Matches(msg, browserKeys.Help):
			if !m.isFiltering() && !m.viewer.searching {
				m.helpOpen = true
			}
			return nil

		case key.Matches(msg, browserKeys.ToggleFullscreen):
			if !m.isFiltering() && !m.viewer.searching {
				m.toggleTranscriptLayout()
			}
			return nil

		case key.Matches(msg, browserKeys.Close):
			if !m.isFiltering() && !m.viewer.searching {
				m.closeTranscript()
			}
			return nil

		case m.transcriptMode == transcriptSplit && key.Matches(msg, browserKeys.FocusPane):
			if !m.isFiltering() && !m.viewer.searching {
				if m.focus == focusList {
					m.focus = focusTranscript
				} else {
					m.focus = focusList
				}
			}
			return nil
		}
	}

	if m.transcriptFocused() {
		return nil
	}

	if key.Matches(msg, browserKeys.Help) && !m.isFiltering() {
		m.helpOpen = true
		return nil
	}

	if m.isFiltering() {
		m.pendingListGotoTopKey = false
		return nil
	}

	if msg.Text == "g" {
		if m.pendingListGotoTopKey {
			m.list.GoToStart()
			m.pendingListGotoTopKey = false
			m.syncTranscriptSelection(cmds)
			return nil
		}
		m.pendingListGotoTopKey = true
		return nil
	}
	m.pendingListGotoTopKey = false

	switch {
	case key.Matches(msg, browserKeys.Enter):
		if conv, ok := m.selectedConversation(); ok {
			m.transcriptMode = transcriptSplit
			m.focus = focusList
			m.updateLayout()
			return m.openTranscript(conv)
		}

	case key.Matches(msg, browserKeys.DeepSearch):
		m.deepSearch = !m.deepSearch
		if m.deepSearch {
			m.notification = infoNotification("deep search: loading...").notification
			return deepSearchCmd(
				m.ctx,
				m.list.FilterValue(),
				filterMainConversations(m.allConversations),
				m.cloneSearchIndex(),
				m.cloneSessionCache(),
			)
		}
		*cmds = append(*cmds, m.list.SetItems(conversationItems(filterMainConversations(m.allConversations))))
		m.setNotification(infoNotification("deep search disabled").notification, cmds)
		m.syncTranscriptSelection(cmds)
		return nil

	case key.Matches(msg, browserKeys.Editor):
		if conv, ok := m.selectedConversation(); ok {
			return openInEditorCmd(conv.latestFilePath())
		}

	case key.Matches(msg, browserKeys.Resume):
		if conv, ok := m.selectedConversation(); ok {
			return resumeSessionCmd(conv.resumeID(), conv.resumeCWD())
		}

	case key.Matches(msg, browserKeys.Quit):
		return tea.Quit
	}

	return nil
}

func (m *browserModel) selectedConversation() (conversation, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return conversation{}, false
	}

	conv, ok := item.(conversation)
	return conv, ok
}

func (m *browserModel) setNotification(n notification, cmds *[]tea.Cmd) {
	m.notification = n
	*cmds = append(*cmds, clearNotificationAfter(n.kind))
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

func (m *browserModel) openTranscript(conv conversation) tea.Cmd {
	if session, ok := m.transcriptCache[conv.id()]; ok {
		m.installViewer(session, conv)
		return nil
	}

	m.openConversationID = ""
	m.loadingConversationID = conv.id()
	if session, ok := m.sessionCache[conv.id()]; ok {
		return openConversationCmdCached(m.ctx, conv, session)
	}
	return openConversationCmd(m.ctx, conv)
}

func (m *browserModel) installViewer(session sessionFull, conv conversation) {
	m.openConversationID = session.meta.id
	m.loadingConversationID = ""
	m.transcriptCache[session.meta.id] = session
	m.sessionCache[session.meta.id] = session
	m.searchIndex[session.meta.id] = buildSessionSearchBlob(session)
	m.addToCache(session.meta.id)

	m.viewer = newViewerModel(session, conv, m.glamourStyle, m.viewerWidth(), m.height)
	if m.transcriptMode == transcriptClosed {
		m.transcriptMode = transcriptSplit
	}
	if m.transcriptMode == transcriptFullscreen {
		m.focus = focusTranscript
	}
}

func (m *browserModel) syncTranscriptSelection(cmds *[]tea.Cmd) {
	if m.transcriptMode != transcriptSplit || m.helpOpen {
		return
	}

	conv, ok := m.selectedConversation()
	if !ok || conv.id() == m.openConversationID || conv.id() == m.loadingConversationID {
		return
	}

	cmd := m.openTranscript(conv)
	if cmd != nil {
		*cmds = append(*cmds, cmd)
	}
}

func (m *browserModel) closeTranscript() {
	m.transcriptMode = transcriptClosed
	m.focus = focusList
	m.helpOpen = false
	m.loadingConversationID = ""
	m.openConversationID = ""
	m.updateLayout()
}

func (m *browserModel) toggleTranscriptLayout() {
	switch m.transcriptMode {
	case transcriptClosed:
		return
	case transcriptSplit:
		m.transcriptMode = transcriptFullscreen
		m.focus = focusTranscript
	case transcriptFullscreen:
		m.transcriptMode = transcriptSplit
		m.focus = focusList
	}
	m.updateLayout()
}

func (m browserModel) transcriptVisible() bool {
	return m.transcriptMode != transcriptClosed
}

func (m browserModel) transcriptFocused() bool {
	return m.transcriptVisible() &&
		(m.transcriptMode == transcriptFullscreen || m.focus == focusTranscript)
}

func (m browserModel) isFiltering() bool {
	return m.list.FilterState() == list.Filtering
}

func (m browserModel) shouldUpdateList(isKey bool) bool {
	if !isKey {
		return true
	}
	if m.helpOpen || m.transcriptMode == transcriptFullscreen {
		return false
	}
	return m.focus == focusList
}

func (m browserModel) shouldUpdateViewer(isKey bool) bool {
	if !m.transcriptVisible() || m.viewer.session.meta.id == "" || !isKey || m.helpOpen {
		return false
	}
	return m.transcriptFocused()
}

func (m *browserModel) addToCache(id string) {
	if slices.Contains(m.cacheOrder, id) {
		return
	}

	m.cacheOrder = append(m.cacheOrder, id)
	for len(m.cacheOrder) > browserCacheSize {
		evictID := m.cacheOrder[0]
		m.cacheOrder = m.cacheOrder[1:]
		delete(m.sessionCache, evictID)
		delete(m.transcriptCache, evictID)
	}
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
