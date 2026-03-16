package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	arch "github.com/rkuska/carn/internal/archive"
	conv "github.com/rkuska/carn/internal/conversation"
)

type browserResyncRequestedMsg struct{}

type resyncPhase int

const (
	resyncPhaseIdle resyncPhase = iota
	resyncPhaseAnalyzing
	resyncPhaseSyncing
)

type browserResyncState struct {
	active   bool
	phase    resyncPhase
	current  int
	total    int
	activity arch.SyncActivity
}

func (m browserModel) requestResyncCmd() tea.Cmd {
	return func() tea.Msg {
		return browserResyncRequestedMsg{}
	}
}

func (m browserModel) prepareForResyncReload() browserModel {
	m.resync = browserResyncState{}
	m.pendingResyncTranscriptID = m.visibleTranscriptConversationID()
	m.openConversationID = ""
	m.loadingConversationID = ""
	m.sessionCache = make(map[string]conv.Session, m.browserCacheSize)
	return m
}

func (m browserModel) visibleTranscriptConversationID() string {
	if !m.transcriptVisible() {
		return ""
	}
	if m.openConversationID != "" {
		return m.openConversationID
	}
	if key := m.viewer.conversation.CacheKey(); key != "" {
		return key
	}
	return m.viewer.session.Meta.ID
}

func (m browserModel) resyncHelpItem() helpItem {
	return helpItem{
		key:      "R",
		desc:     "resync",
		detail:   "refresh source sessions and rebuild the local store",
		priority: helpPriorityHigh,
	}
}

func (m browserModel) resyncStatusParts() []string {
	if !m.resync.active {
		return nil
	}

	parts := []string{styleToolCall.Render("[resync]")}
	switch m.resync.phase {
	case resyncPhaseIdle:
		return nil
	case resyncPhaseAnalyzing:
		parts = append(parts, "analyzing")
	case resyncPhaseSyncing:
		if m.resync.activity == arch.SyncActivityRebuildingStore {
			parts = append(parts, m.resyncSpinner.View(), resyncSyncActivityLabel(m.resync.activity))
			return parts
		}
		if m.resync.total > 0 {
			parts = append(parts, fmt.Sprintf("%d/%d", m.resync.current, m.resync.total))
		}
	}
	return parts
}

func (m browserModel) resyncSpinnerActive() bool {
	return m.resync.active &&
		m.resync.phase == resyncPhaseSyncing &&
		m.resync.activity == arch.SyncActivityRebuildingStore
}
