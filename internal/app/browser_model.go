package app

import (
	"context"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	conv "github.com/rkuska/carn/internal/conversation"
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
	ctx                       context.Context
	archiveDir                string
	store                     browserStore
	launcher                  sessionLauncher
	glamourStyle              string
	list                      list.Model
	delegate                  conversationDelegate
	focus                     focusArea
	transcriptMode            transcriptMode
	allConversations          []conv.Conversation
	width                     int
	height                    int
	mainConversationCount     int
	notification              notification
	searchInput               textinput.Model
	search                    browserSearchState
	deepSearchAvailable       bool
	sessionCache              map[string]conv.Session
	transcriptCache           map[string]conv.Session
	searchCancel              context.CancelFunc
	openConversationID        string
	loadingConversationID     string
	helpOpen                  bool
	pendingListGotoTopKey     bool
	cacheOrder                []string
	viewer                    viewerModel
	resync                    browserResyncState
	resyncSpinner             spinner.Model
	pendingResyncTranscriptID string
}

func newBrowserModelWithStore(
	ctx context.Context,
	archiveDir, glamourStyle string,
	store browserStore,
	launchers ...sessionLauncher,
) browserModel {
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

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	launcher := newDefaultSessionLauncher()
	if len(launchers) > 0 && launchers[0] != nil {
		launcher = launchers[0]
	}

	return browserModel{
		ctx:            ctx,
		archiveDir:     archiveDir,
		store:          store,
		launcher:       launcher,
		glamourStyle:   glamourStyle,
		list:           l,
		delegate:       delegate,
		focus:          focusList,
		transcriptMode: transcriptClosed,
		searchInput:    newBrowserSearchInput(),
		search: browserSearchState{
			mode:   searchModeMetadata,
			status: searchStatusIdle,
		},
		sessionCache:        make(map[string]conv.Session, browserCacheSize),
		transcriptCache:     make(map[string]conv.Session, browserCacheSize),
		deepSearchAvailable: true,
		resyncSpinner:       s,
	}
}

func newBrowserModel(ctx context.Context, archiveDir, glamourStyle string) browserModel {
	return newBrowserModelWithStore(
		ctx,
		archiveDir,
		glamourStyle,
		newDefaultBrowserStore(),
	)
}

func (m browserModel) Init() tea.Cmd {
	return loadSessionsCmdWithStore(m.ctx, m.archiveDir, m.store)
}

func (m browserModel) Update(msg tea.Msg) (browserModel, tea.Cmd) {
	var cmds []tea.Cmd
	m = m.handleMsg(msg, &cmds)

	_, isKey := msg.(tea.KeyPressMsg)
	m = m.updateChildModels(msg, isKey, &cmds)

	return m, tea.Batch(cmds...)
}

func (m browserModel) applyOpenViewer(msg openViewerMsg) browserModel {
	if msg.conversationID == m.loadingConversationID && msg.conversationID != "" {
		return m.installViewer(msg.session, msg.conversation)
	}
	return m
}

func (m browserModel) clearNotifications() browserModel {
	m.notification = notification{}
	if m.viewer.notification.text != "" {
		m.viewer.notification = notification{}
	}
	return m
}

func (m browserModel) updateChildModels(msg tea.Msg, isKey bool, cmds *[]tea.Cmd) browserModel {
	if m.shouldUpdateList(isKey) {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		appendCmd(cmds, cmd)
		m = m.updateSelectedConversationID()
		m = m.syncTranscriptSelection(cmds)
	}

	if m.shouldUpdateViewer(isKey) {
		var cmd tea.Cmd
		previousNotification := m.viewer.notification
		m.viewer, cmd = m.viewer.Update(msg)
		appendCmd(cmds, cmd)
		if m.viewer.notification != previousNotification {
			m.notification = m.viewer.notification
		}
	}
	return m
}

func appendCmd(cmds *[]tea.Cmd, cmd tea.Cmd) {
	if cmd != nil {
		*cmds = append(*cmds, cmd)
	}
}
