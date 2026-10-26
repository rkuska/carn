package browser

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

// dockMinViewportRows is the fewest transcript rows the viewport keeps when
// the Context Dock is open.
const dockMinViewportRows = 3

// dockMinWindowRows is the fewest window rows that hold the dock and
// dockMinViewportRows transcript rows inside the frame:
//
//	total = chrome (4) + viewport (3) + dock (9) = 16
const dockMinWindowRows = el.FramedChromeHeight + dockMinViewportRows + el.ContextDockHeight

// SetFullscreen tells the viewer whether it owns the whole window; the Context
// Dock is only drawn in fullscreen.
func (m viewerModel) SetFullscreen(fullscreen bool) viewerModel {
	if m.fullscreen == fullscreen {
		return m
	}
	m.fullscreen = fullscreen
	return m.applyViewportHeight()
}

// setContextSeries installs series and the dock data derived from it. The
// context keys are offered exactly when the series has steps.
func (m viewerModel) setContextSeries(series conv.ContextSeries) viewerModel {
	m.contextSeries = series
	m.content.hasContext = len(series.Steps) > 0
	m.dockData = el.NewContextDockData(series)
	return m.refreshDockColumns()
}

// dockContentWidth is the width the dock draws in: the transcript content width
// inside the frame.
func (m viewerModel) dockContentWidth() int {
	return max(m.width-viewerBorderH-viewerPaddingH, 1)
}

// refreshDockColumns recomputes the chart column layout; it only depends on the
// steps and the width, not on the cursor.
func (m viewerModel) refreshDockColumns() viewerModel {
	m.dockColumns = el.NewContextDockColumns(m.dockData.Steps, el.ContextDockChartWidth(m.dockContentWidth()))
	return m
}

// dockFits reports whether the window can hold the dock: fullscreen and wide
// enough, and tall enough to keep a few transcript rows next to it.
func (m viewerModel) dockFits() bool {
	return m.dockFitsWidth() && m.height >= dockMinWindowRows
}

// dockFitsWidth reports whether the viewer is fullscreen and wide enough for
// the dock.
func (m viewerModel) dockFitsWidth() bool {
	return m.fullscreen && m.dockContentWidth() >= el.ContextDockMinWidth
}

// dockHeight is the number of lines the dock takes below the viewport:
//
//	total = top border (1) + viewport + dock + bottom border (1) + footer (2)
//
// It is zero unless the dock is on, fits, has data and selection mode is off.
func (m viewerModel) dockHeight() int {
	if m.dockOpen && !m.selectionMode && len(m.dockData.Steps) > 0 && m.dockFits() {
		return el.ContextDockHeight
	}
	return 0
}

// viewportHeight is the body height left for the transcript viewport.
func (m viewerModel) viewportHeight() int {
	return max(m.bodyHeight()-m.dockHeight(), 1)
}

// applyViewportHeight resizes the viewport for the current dock height and
// keeps the top line in place where possible.
func (m viewerModel) applyViewportHeight() viewerModel {
	m.viewport.SetHeight(m.viewportHeight())
	m.viewport.SetYOffset(m.viewport.YOffset())
	return m
}

// toggleDock opens or closes the Context Dock. Only opening is refused when the
// dock cannot be drawn; an open dock can always be closed, even after the
// window stopped fitting it.
func (m viewerModel) toggleDock(cmds *[]tea.Cmd) viewerModel {
	if notice := m.dockOpenNotice(); !m.dockOpen && notice != "" {
		return m.setNotification(infoNotification(notice).Notification, cmds)
	}
	m.dockOpen = !m.dockOpen
	m = m.applyViewportHeight()
	return m.setNotification(infoNotification("context: "+toggleLabel(m.dockOpen)).Notification, cmds)
}

// dockOpenNotice explains why the dock cannot be opened, or is "" when it can.
func (m viewerModel) dockOpenNotice() string {
	switch {
	case len(m.dockData.Steps) == 0:
		return noContextNotice
	case !m.dockFitsWidth():
		return fmt.Sprintf("context dock needs fullscreen (O) and %d+ columns", el.ContextDockMinWidth)
	case !m.dockFits():
		return fmt.Sprintf("context dock needs %d+ rows", dockMinWindowRows)
	}
	return ""
}

// cursorStep is the Context Step under the dock cursor: the step of the message
// on the viewport's top line, or the first step above any message.
func (m viewerModel) cursorStep() int {
	step, _ := m.contextSeries.StepAt(m.messageAtLine(m.viewport.YOffset()))
	return step
}

// dockView renders the dock padded to the viewport's width, or "" when it is
// not shown.
func (m viewerModel) dockView() string {
	if m.dockHeight() == 0 {
		return ""
	}
	cursor := m.cursorStep()
	cause := m.contextSeries.StepCause(m.session.Messages, cursor)
	input := m.dockData.Input(m.dockColumns, cursor, cause)
	lines := strings.Split(m.theme.RenderContextDock(input, m.dockContentWidth()), "\n")
	for i, line := range lines {
		lines[i] = " " + line + " "
	}
	return strings.Join(lines, "\n")
}

// jumpToNotable scrolls so the next (direction > 0) or previous Notable Step
// starts on the top line.
func (m viewerModel) jumpToNotable(direction int, cmds *[]tea.Cmd) viewerModel {
	lines, top := el.NotableLines(m.contextSeries, m.contextLines), m.viewport.YOffset()
	target, found := el.PrevLine(lines, top)
	label := "earlier"
	if direction > 0 {
		target, found = el.NextLine(lines, top)
		label = "later"
	}
	if found {
		m.viewport.SetYOffset(target)
	}
	if !found || m.viewport.YOffset() == top {
		return m.setNotification(infoNotification("no "+label+" notable step").Notification, cmds)
	}
	return m
}
