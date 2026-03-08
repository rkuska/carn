package app

import (
	"context"
	"fmt"
	"slices"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
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
	repo                  conversationRepository
	glamourStyle          string
	list                  list.Model
	focus                 focusArea
	transcriptMode        transcriptMode
	allConversations      []conversation
	width                 int
	height                int
	mainConversationCount int
	notification          notification
	searchInput           textinput.Model
	search                browserSearchState
	searchCorpus          searchCorpus
	deepSearchAvailable   bool
	sessionCache          map[string]sessionFull
	transcriptCache       map[string]sessionFull
	searchCancel          context.CancelFunc
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
	l.SetFilteringEnabled(false)
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
		ctx:            ctx,
		archiveDir:     archiveDir,
		repo:           newDefaultConversationRepository(),
		glamourStyle:   glamourStyle,
		list:           l,
		focus:          focusList,
		transcriptMode: transcriptClosed,
		searchInput:    newBrowserSearchInput(),
		search: browserSearchState{
			mode:   searchModeMetadata,
			status: searchStatusIdle,
		},
		sessionCache:        make(map[string]sessionFull, browserCacheSize),
		transcriptCache:     make(map[string]sessionFull, browserCacheSize),
		deepSearchAvailable: true,
	}
}

func (m browserModel) Init() tea.Cmd {
	return loadSessionsCmdWithRepository(m.ctx, m.archiveDir, m.repo)
}

func (m browserModel) Update(msg tea.Msg) (browserModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.searchEditing() && !m.transcriptFocused() {
			var cmd tea.Cmd
			m, cmd = m.handleSearchKey(msg, &cmds)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		} else {
			cmd := m.handleKey(msg, &cmds)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()

	case conversationsLoadedMsg:
		m.allConversations = msg.conversations
		mainConvs := filterMainConversations(msg.conversations)
		m.mainConversationCount = len(mainConvs)
		m.searchCorpus = msg.searchCorpus
		m.deepSearchAvailable = msg.deepSearchAvailable
		m.search.baseConversations = mainConvs
		m.search.visibleConversations = mainConvs
		if !m.deepSearchAvailable && m.search.mode == searchModeDeep {
			m.search.mode = searchModeMetadata
			m.search.status = searchStatusIdle
		}
		if m.search.query == "" {
			m.applyFullConversationList(&cmds)
		} else {
			m.refreshSearchResults(&cmds)
		}
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

	case deepSearchDebounceMsg:
		if m.search.mode == searchModeDeep &&
			msg.revision == m.search.revision &&
			msg.query == m.search.query {
			m.startDeepSearch(&cmds)
		}

	case deepSearchResultMsg:
		if m.search.mode == searchModeDeep &&
			msg.revision == m.search.revision &&
			msg.query == m.search.query {
			m.search.appliedRevision = msg.revision
			m.search.status = searchStatusIdle
			m.searchCancel = nil
			m.setSearchItems(buildDeepSearchItems(msg.conversations), &cmds)
			m.syncTranscriptSelection(&cmds)
		}

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
		m.updateSelectedConversationID()
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

	if key.Matches(msg, browserKeys.Help) && !m.searchEditing() {
		m.helpOpen = true
		return nil
	}

	if m.searchEditing() {
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
	case key.Matches(msg, browserKeys.Search):
		return m.beginSearchEditing()

	case key.Matches(msg, browserKeys.Enter):
		if conv, ok := m.selectedConversation(); ok {
			m.transcriptMode = transcriptSplit
			m.focus = focusList
			m.updateLayout()
			return m.openTranscript(conv)
		}

	case key.Matches(msg, browserKeys.DeepSearch):
		m.toggleSearchMode(cmds)
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

	return conversationFromItem(item)
}

func (m *browserModel) setNotification(n notification, cmds *[]tea.Cmd) {
	m.notification = n
	*cmds = append(*cmds, clearNotificationAfter(n.kind))
}

func (m *browserModel) openTranscript(conv conversation) tea.Cmd {
	if session, ok := m.transcriptCache[conv.cacheKey()]; ok {
		m.installViewer(session, conv)
		return nil
	}

	m.openConversationID = ""
	m.loadingConversationID = conv.cacheKey()
	if session, ok := m.sessionCache[conv.cacheKey()]; ok {
		return openConversationCmdCachedWithRepository(m.ctx, conv, session, m.repo)
	}
	return openConversationCmdWithRepository(m.ctx, m.archiveDir, conv, m.repo)
}

func (m *browserModel) installViewer(session sessionFull, conv conversation) {
	key := conv.cacheKey()
	if key == "" {
		key = session.meta.id
	}
	m.openConversationID = key
	m.loadingConversationID = ""
	m.transcriptCache[key] = session
	m.sessionCache[key] = session
	m.addToCache(key)

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
	if !ok || conv.cacheKey() == m.openConversationID || conv.cacheKey() == m.loadingConversationID {
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
	return m.search.editing
}

func (m browserModel) shouldUpdateList(isKey bool) bool {
	if !isKey {
		return true
	}
	if m.helpOpen || m.transcriptMode == transcriptFullscreen || m.searchEditing() {
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
