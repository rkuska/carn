package browser

import (
	"context"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	el "github.com/rkuska/carn/internal/app/elements"
	arch "github.com/rkuska/carn/internal/archive"
	"github.com/rkuska/carn/internal/canonical"
	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

type Model = browserModel
type Store = browserStore
type SessionLauncher = sessionLauncher
type ResyncRequestedMsg = browserResyncRequestedMsg
type ResyncPhase = resyncPhase
type DeepSearchDebounceMsg = deepSearchDebounceMsg
type DeepSearchResultMsg = deepSearchResultMsg

const (
	ResyncPhaseIdle      = resyncPhaseIdle
	ResyncPhaseAnalyzing = resyncPhaseAnalyzing
	ResyncPhaseSyncing   = resyncPhaseSyncing
)

func NewModelWithStore(
	ctx context.Context,
	archiveDir string,
	logFilePath string,
	glamourStyle string,
	timestampFormat string,
	cacheSize int,
	debounceMs int,
	store Store,
	launchers ...SessionLauncher,
) Model {
	return newBrowserModelWithStore(
		ctx,
		archiveDir,
		logFilePath,
		glamourStyle,
		timestampFormat,
		cacheSize,
		debounceMs,
		store,
		launchers...,
	)
}

func NewStore(store *canonical.Store) Store {
	return newBrowserStore(store)
}

func NewSessionLauncher(backends ...src.Backend) SessionLauncher {
	return newSessionLauncher(backends...)
}

func NewFilterState() el.FilterState {
	return newBrowserFilterState()
}

func SingleSessionConversation(meta conv.SessionMeta) conv.Conversation {
	return singleSessionConversation(meta)
}

func NewDeepSearchDebounceMsg(revision int, query string) DeepSearchDebounceMsg {
	return deepSearchDebounceMsg{
		revision: revision,
		query:    query,
	}
}

func (m Model) AllConversations() []conv.Conversation {
	return m.allConversations
}

func (m Model) ArchiveDir() string {
	return m.archiveDir
}

func (m Model) FilterState() el.FilterState {
	return m.filter
}

func (m Model) TimestampFormat() string {
	return m.timestampFormat
}

func (m Model) BrowserCacheSize() int {
	return m.browserCacheSize
}

func (m Model) DeepSearchDebounceMs() int {
	return m.deepSearchDebounceMs
}

func (m Model) Notification() el.Notification {
	return m.notification
}

func (m Model) ViewerSession() conv.Session {
	return m.viewer.session
}

func (m Model) OpenConversationID() string {
	return m.openConversationID
}

func (m Model) SearchQuery() string {
	return m.search.query
}

func (m Model) SearchRevision() int {
	return m.search.revision
}

func (m Model) SearchSelectedConversationID() string {
	return m.search.selectedConversationID
}

func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	return m.updateLayout()
}

func (m Model) SetConversationLists(
	allConversations []conv.Conversation,
	mainConversations []conv.Conversation,
	filter el.FilterState,
) Model {
	m.allConversations = append([]conv.Conversation(nil), allConversations...)
	m.mainConversations = append([]conv.Conversation(nil), mainConversations...)
	m.filter = copyBrowserFilterState(filter)
	return m
}

func (m Model) SetSearchState(
	query string,
	baseConversations []conv.Conversation,
	visibleConversations []conv.Conversation,
	selectedConversationID string,
) Model {
	m.search.query = query
	m.search.baseConversations = append([]conv.Conversation(nil), baseConversations...)
	m.search.visibleConversations = append([]conv.Conversation(nil), visibleConversations...)
	m.search.selectedConversationID = selectedConversationID
	return m
}

func (m Model) SetListConversations(conversations []conv.Conversation, selected int) Model {
	items := buildPlainConversationItems(conversations)
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}
	m.list.SetItems(listItems)
	if selected >= 0 && selected < len(listItems) {
		m.list.Select(selected)
	}
	return m
}

func (m Model) SetNotification(n el.Notification) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	m = m.setNotification(n, &cmds)
	return m, tea.Batch(cmds...)
}

func (m Model) ResyncActive() bool {
	return m.resync.active
}

func (m Model) StartResync() Model {
	if m.resync.active {
		return m
	}
	m.resync = browserResyncState{
		active: true,
		phase:  resyncPhaseAnalyzing,
	}
	return m
}

func (m Model) SetResyncPhase(phase ResyncPhase) Model {
	m.resync.phase = resyncPhase(phase)
	return m
}

func (m Model) ApplyResyncProgress(progress arch.SyncProgress) (Model, bool) {
	startSpinner := !m.resyncSpinnerActive() &&
		progress.Activity == arch.SyncActivityRebuildingStore
	m.resync.phase = resyncPhaseSyncing
	m.resync.current = progress.Current
	m.resync.total = progress.Total
	m.resync.activity = progress.Activity
	return m, startSpinner
}

func (m Model) ResyncSpinnerTickCmd() tea.Cmd {
	return m.resyncSpinner.Tick
}

func (m Model) ClearResync() Model {
	m.resync = browserResyncState{}
	return m
}

func (m Model) PrepareForResyncReload() Model {
	return m.prepareForResyncReload()
}

func (m Model) LoadSessionsCmd() tea.Cmd {
	return loadSessionsCmdWithStore(m.ctx, m.archiveDir, m.store)
}

func (m Model) OpenLoadedSession(conversation conv.Conversation, session conv.Session) Model {
	m.transcriptMode = transcriptFullscreen
	m.focus = focusTranscript
	m = m.updateLayout()
	return m.installViewer(session, conversation)
}
