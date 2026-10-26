package browser

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// noContextNotice explains why the context keys do nothing in a conversation
// without recorded usage.
const noContextNotice = "no context usage recorded in this conversation"

// messageAtLine returns the index of the message whose content is on rendered
// line y, or -1 when y lies before the first message (conversation header).
func (m viewerModel) messageAtLine(y int) int {
	return m.contextLines.MessageAt(y)
}

// handleContextKey handles the keys of the context views: C toggles the Turn
// Ledger, c the Context Dock, [ and ] jump between Notable Steps.
func (m viewerModel) handleContextKey(msg tea.KeyPressMsg, cmds *[]tea.Cmd) (viewerModel, bool) {
	switch {
	case key.Matches(msg, viewerKeys.ToggleDock):
		return m.toggleDock(cmds), true
	case key.Matches(msg, viewerKeys.NextNotable):
		return m.jumpToNotable(1, cmds), true
	case key.Matches(msg, viewerKeys.PrevNotable):
		return m.jumpToNotable(-1, cmds), true
	case key.Matches(msg, viewerKeys.ToggleLedger):
		return m.toggleLedger(cmds), true
	}
	return m, false
}

// toggleLedger shows or hides the Turn Ledger. Like the dock, it is refused
// when the conversation recorded no context usage: help and footer hide the
// key then, and the ledger would have nothing to annotate.
func (m viewerModel) toggleLedger(cmds *[]tea.Cmd) viewerModel {
	if !m.content.hasContext {
		return m.setNotification(infoNotification(noContextNotice).Notification, cmds)
	}
	m.opts.showContextLedger = !m.opts.showContextLedger
	m = m.renderContentPreservingScroll()
	return m.setNotification(infoNotification("ledger: "+toggleLabel(m.opts.showContextLedger)).Notification, cmds)
}
