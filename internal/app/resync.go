package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	appbrowser "github.com/rkuska/carn/internal/app/browser"
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
	case appbrowser.ResyncRequestedMsg:
		m = m.startBrowserResync()
		model, cmd := m.withBrowserNotification(
			infoNotification("resync started").Notification,
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
	if m.browser.ResyncActive() {
		return m
	}
	m.browser = m.browser.StartResync()
	return m
}

func (m appModel) handleResyncAnalysisStarted(msg importAnalysisStartedMsg) (appModel, tea.Cmd, bool) {
	if !m.browser.ResyncActive() {
		return m, nil, false
	}
	m.resyncEvents = msg.events
	return m, waitForAsyncImportMsg(m.resyncEvents), true
}

func (m appModel) handleResyncAnalysisProgress() (appModel, tea.Cmd, bool) {
	if !m.browser.ResyncActive() {
		return m, nil, false
	}
	m.browser = m.browser.SetResyncPhase(appbrowser.ResyncPhaseAnalyzing)
	return m, waitForAsyncImportMsg(m.resyncEvents), true
}

func (m appModel) handleResyncAnalysisFinished(msg analysisFinishedMsg) (appModel, tea.Cmd, bool) {
	if !m.browser.ResyncActive() {
		return m, nil, false
	}
	if msg.analysis.Err != nil {
		return m.finishBrowserResyncError(fmt.Sprintf("resync failed: %v", msg.analysis.Err))
	}
	if !msg.analysis.NeedsSync() {
		return m.finishBrowserResyncSuccess("archive already current")
	}
	m.browser = m.browser.SetResyncPhase(appbrowser.ResyncPhaseSyncing)
	return m, startImportSyncCmd(m.ctx, m.pipeline), true
}

func (m appModel) handleResyncSyncStarted(msg importSyncStartedMsg) (appModel, tea.Cmd, bool) {
	if !m.browser.ResyncActive() {
		return m, nil, false
	}
	m.resyncEvents = msg.events
	return m, waitForAsyncImportMsg(m.resyncEvents), true
}

func (m appModel) handleResyncSyncProgress(msg importSyncProgressMsg) (appModel, tea.Cmd, bool) {
	if !m.browser.ResyncActive() {
		return m, nil, false
	}
	var startSpinner bool
	m.browser, startSpinner = m.browser.ApplyResyncProgress(msg.progress)
	waitCmd := waitForAsyncImportMsg(m.resyncEvents)
	if startSpinner {
		return m, tea.Batch(waitCmd, m.browser.ResyncSpinnerTickCmd()), true
	}
	return m, waitCmd, true
}

func (m appModel) handleResyncSyncFinished(msg importSyncFinishedMsg) (appModel, tea.Cmd, bool) {
	if !m.browser.ResyncActive() {
		return m, nil, false
	}
	if msg.err != nil {
		return m.finishBrowserResyncError(fmt.Sprintf("resync failed: %v", msg.err))
	}
	m.browser = m.browser.PrepareForResyncReload()
	n := successNotification("resync finished").Notification
	if malformed, ok := malformedDataNotification(msg.result.MalformedData); ok {
		n = malformed
	} else if drift, ok := driftNotification(msg.result.Drift); ok {
		n = drift
	}
	model, cmd := m.withBrowserNotification(
		n,
		m.browser.LoadSessionsCmd(),
	)
	return model, cmd, true
}

func (m appModel) finishBrowserResyncError(text string) (appModel, tea.Cmd, bool) {
	m.browser = m.browser.ClearResync()
	model, cmd := m.withBrowserNotification(errorNotification(text).Notification, nil)
	return model, cmd, true
}

func (m appModel) finishBrowserResyncSuccess(text string) (appModel, tea.Cmd, bool) {
	m.browser = m.browser.ClearResync()
	model, cmd := m.withBrowserNotification(successNotification(text).Notification, nil)
	return model, cmd, true
}

func (m appModel) withBrowserNotification(n notification, next tea.Cmd) (appModel, tea.Cmd) {
	var cmds []tea.Cmd
	appendCmd(&cmds, next)
	var notify tea.Cmd
	m.browser, notify = m.browser.SetNotification(n)
	appendCmd(&cmds, notify)
	return m, tea.Batch(cmds...)
}
