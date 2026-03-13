package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	arch "github.com/rkuska/carn/internal/archive"
)

func (m appModel) updateBrowser(msg tea.Msg) (tea.Model, tea.Cmd) {
	if next, cmd, handled := m.handleBrowserResyncMsg(msg); handled {
		return next, cmd
	}

	var cmd tea.Cmd
	m.browser, cmd = m.browser.Update(msg)
	return m, cmd
}

func (m appModel) handleBrowserResyncMsg(msg tea.Msg) (appModel, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case browserResyncRequestedMsg:
		m = m.startBrowserResync()
		model, cmd := m.withBrowserNotification(
			infoNotification("resync started").notification,
			startImportAnalysisCmd(m.ctx, m.pipeline),
		)
		return model, cmd, true
	case importAnalysisStartedMsg:
		return m.handleResyncAnalysisStarted(msg)
	case analysisProgressMsg:
		return m.handleResyncAnalysisProgress()
	case analysisFinishedMsg:
		return m.handleResyncAnalysisFinished(msg)
	case importSyncStartedMsg:
		return m.handleResyncSyncStarted(msg)
	case importSyncProgressMsg:
		return m.handleResyncSyncProgress(msg)
	case importSyncFinishedMsg:
		return m.handleResyncSyncFinished(msg)
	}
	return m, nil, false
}

func (m appModel) startBrowserResync() appModel {
	if m.browser.resync.active {
		return m
	}
	m.browser.resync = browserResyncState{
		active: true,
		phase:  resyncPhaseAnalyzing,
	}
	return m
}

func (m appModel) handleResyncAnalysisStarted(msg importAnalysisStartedMsg) (appModel, tea.Cmd, bool) {
	if !m.browser.resync.active {
		return m, nil, false
	}
	m.resyncEvents = msg.events
	return m, waitForAsyncImportMsg(m.resyncEvents), true
}

func (m appModel) handleResyncAnalysisProgress() (appModel, tea.Cmd, bool) {
	if !m.browser.resync.active {
		return m, nil, false
	}
	m.browser.resync.phase = resyncPhaseAnalyzing
	return m, waitForAsyncImportMsg(m.resyncEvents), true
}

func (m appModel) handleResyncAnalysisFinished(msg analysisFinishedMsg) (appModel, tea.Cmd, bool) {
	if !m.browser.resync.active {
		return m, nil, false
	}
	if msg.analysis.Err != nil {
		return m.finishBrowserResyncError(fmt.Sprintf("resync failed: %v", msg.analysis.Err))
	}
	if !msg.analysis.NeedsSync() {
		return m.finishBrowserResyncSuccess("archive already current")
	}
	m.browser.resync.phase = resyncPhaseSyncing
	return m, startImportSyncCmd(m.ctx, m.pipeline), true
}

func (m appModel) handleResyncSyncStarted(msg importSyncStartedMsg) (appModel, tea.Cmd, bool) {
	if !m.browser.resync.active {
		return m, nil, false
	}
	m.resyncEvents = msg.events
	return m, waitForAsyncImportMsg(m.resyncEvents), true
}

func (m appModel) handleResyncSyncProgress(msg importSyncProgressMsg) (appModel, tea.Cmd, bool) {
	if !m.browser.resync.active {
		return m, nil, false
	}
	startSpinner := !m.browser.resyncSpinnerActive() &&
		msg.progress.Activity == arch.SyncActivityRebuildingStore
	m.browser.resync.phase = resyncPhaseSyncing
	m.browser.resync.current = msg.progress.Current
	m.browser.resync.total = msg.progress.Total
	m.browser.resync.activity = msg.progress.Activity
	waitCmd := waitForAsyncImportMsg(m.resyncEvents)
	if startSpinner {
		return m, tea.Batch(waitCmd, m.browser.resyncSpinner.Tick), true
	}
	return m, waitCmd, true
}

func (m appModel) handleResyncSyncFinished(msg importSyncFinishedMsg) (appModel, tea.Cmd, bool) {
	if !m.browser.resync.active {
		return m, nil, false
	}
	if msg.err != nil {
		return m.finishBrowserResyncError(fmt.Sprintf("resync failed: %v", msg.err))
	}
	m.browser = m.browser.prepareForResyncReload()
	model, cmd := m.withBrowserNotification(
		successNotification("resync finished").notification,
		loadSessionsCmdWithStore(m.ctx, m.cfg.ArchiveDir, m.browser.store),
	)
	return model, cmd, true
}

func (m appModel) finishBrowserResyncError(text string) (appModel, tea.Cmd, bool) {
	m.browser.resync = browserResyncState{}
	model, cmd := m.withBrowserNotification(errorNotification(text).notification, nil)
	return model, cmd, true
}

func (m appModel) finishBrowserResyncSuccess(text string) (appModel, tea.Cmd, bool) {
	m.browser.resync = browserResyncState{}
	model, cmd := m.withBrowserNotification(successNotification(text).notification, nil)
	return model, cmd, true
}

func (m appModel) withBrowserNotification(n notification, next tea.Cmd) (appModel, tea.Cmd) {
	var cmds []tea.Cmd
	appendCmd(&cmds, next)
	m.browser = m.browser.setNotification(n, &cmds)
	return m, tea.Batch(cmds...)
}
