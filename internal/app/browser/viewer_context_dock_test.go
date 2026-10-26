package browser

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

func fullscreenContextViewer(width, height int) viewerModel {
	return newContextTestViewer(width, height).SetFullscreen(true)
}

func pressViewerKey(m viewerModel, text string) (viewerModel, string) {
	var cmds []tea.Cmd
	updated, _ := m.handleKey(tea.KeyPressMsg{Code: []rune(text)[0], Text: text}, &cmds)
	return updated, updated.notification.Text
}

func openDock(t *testing.T, m viewerModel) viewerModel {
	t.Helper()

	opened, _ := pressViewerKey(m, "c")
	require.True(t, opened.dockOpen)
	require.Equal(t, el.ContextDockHeight, opened.dockHeight())
	return opened
}

func dockLines(m viewerModel) []string {
	return strings.Split(ansi.Strip(m.dockView()), "\n")
}

func TestDockHeightArithmetic(t *testing.T) {
	t.Parallel()

	sizes := [][2]int{{120, 40}, {100, 30}, {80, 24}, {72 + 4, 22}, {200, 60}, {160, 18}}
	for _, size := range sizes {
		width, height := size[0], size[1]
		m := openDock(t, fullscreenContextViewer(width, height))

		// total = top border + viewport + dock + bottom border + footer
		assert.Equal(t, height, 1+m.viewport.Height()+el.ContextDockHeight+1+2, "%dx%d", width, height)
		assert.Equal(t, m.bodyHeight(), m.viewport.Height()+el.ContextDockHeight)
		view := ansi.Strip(m.View())
		assert.Len(t, strings.Split(view, "\n"), height, "%dx%d", width, height)
		for i, line := range strings.Split(view, "\n") {
			assert.LessOrEqual(t, ansi.StringWidth(line), width, "%dx%d line %d", width, height, i)
		}
	}
}

func TestDockPaneContentFillsTheBody(t *testing.T) {
	t.Parallel()

	m := openDock(t, fullscreenContextViewer(120, 40))

	lines := strings.Split(m.paneContent(), "\n")

	assert.Len(t, lines, m.bodyHeight())
	assert.Equal(t, trimEach(dockLines(m)), trimEach(lines[len(lines)-el.ContextDockHeight:]))
}

func trimEach(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = strings.TrimRight(ansi.Strip(line), " ")
	}
	return out
}

func TestDockClosedKeepsTheWholeBody(t *testing.T) {
	t.Parallel()

	m := fullscreenContextViewer(120, 40)

	assert.Equal(t, 0, m.dockHeight())
	assert.Equal(t, m.bodyHeight(), m.viewport.Height())
}

func TestDockNeedsFullscreen(t *testing.T) {
	t.Parallel()

	m := newContextTestViewer(120, 40)

	updated, notice := pressViewerKey(m, "c")

	assert.False(t, updated.dockOpen)
	assert.Equal(t, "context dock needs fullscreen (O) and 72+ columns", notice)
	assert.Equal(t, updated.bodyHeight(), updated.viewport.Height())
}

func TestDockNeedsEnoughColumns(t *testing.T) {
	t.Parallel()

	narrow := fullscreenContextViewer(75, 40)
	fits := fullscreenContextViewer(76, 40)

	updated, notice := pressViewerKey(narrow, "c")
	assert.False(t, updated.dockOpen)
	assert.Equal(t, "context dock needs fullscreen (O) and 72+ columns", notice)
	assert.True(t, openDock(t, fits).dockOpen)
}

func TestDockNeedsEnoughRows(t *testing.T) {
	t.Parallel()

	short := fullscreenContextViewer(120, 15)
	fits := fullscreenContextViewer(120, 16)

	updated, notice := pressViewerKey(short, "c")

	assert.False(t, updated.dockOpen)
	assert.Equal(t, "context dock needs 16+ rows", notice, "the width fits, so the notice names the rows")
	opened := openDock(t, fits)
	assert.Equal(t, dockMinViewportRows, opened.viewport.Height(),
		"total = chrome (4) + viewport + dock (9) = 16")
}

func TestDockNeedsContextData(t *testing.T) {
	t.Parallel()

	session := contextFixtureSession()
	for i := range session.Messages {
		session.Messages[i].Usage = conv.TokenUsage{}
	}
	m := newViewerModel(session, contextFixtureConversation(session), "dark", "2006-01-02 15:04", 120, 40).
		SetFullscreen(true)

	updated, notice := pressViewerKey(m, "c")

	assert.False(t, updated.dockOpen)
	assert.Equal(t, "no context usage recorded in this conversation", notice)
}

func TestDockKeyTogglesTheDock(t *testing.T) {
	t.Parallel()

	m := openDock(t, fullscreenContextViewer(120, 40))

	closed, notice := pressViewerKey(m, "c")

	assert.False(t, closed.dockOpen)
	assert.Equal(t, "context: off", notice)
	assert.Equal(t, closed.bodyHeight(), closed.viewport.Height())
}

func TestDockKeyClosesADockThatNoLongerFits(t *testing.T) {
	t.Parallel()

	m := openDock(t, fullscreenContextViewer(120, 40)).SetSize(75, 40)
	require.Equal(t, 0, m.dockHeight(), "the narrowed window hides the dock")

	closed, notice := pressViewerKey(m, "c")

	assert.False(t, closed.dockOpen)
	assert.Equal(t, "context: off", notice)
}

func TestDockIsRestoredAfterLeavingFullscreen(t *testing.T) {
	t.Parallel()

	m := openDock(t, fullscreenContextViewer(120, 40))

	split := m.SetFullscreen(false)
	assert.Equal(t, 0, split.dockHeight())
	assert.Equal(t, split.bodyHeight(), split.viewport.Height())
	assert.Empty(t, split.dockView())
	assert.True(t, split.dockOpen, "the dock stays on while hidden")

	back := split.SetFullscreen(true)
	assert.Equal(t, el.ContextDockHeight, back.dockHeight())
	assert.Equal(t, back.bodyHeight()-el.ContextDockHeight, back.viewport.Height())
}

func TestDockIsHiddenInSelectionModeAndRestoredAfterwards(t *testing.T) {
	t.Parallel()

	m := openDock(t, fullscreenContextViewer(120, 40))

	selecting := m.toggleSelectionMode()
	assert.Equal(t, 0, selecting.dockHeight())
	assert.Equal(t, selecting.bodyHeight(), selecting.viewport.Height())
	assert.NotContains(t, selecting.paneContent(), "ctx per model step")

	back := selecting.toggleSelectionMode()
	assert.Equal(t, el.ContextDockHeight, back.dockHeight())
	assert.Equal(t, back.bodyHeight()-el.ContextDockHeight, back.viewport.Height())
}

func TestDockResizeKeepsTheTopLine(t *testing.T) {
	t.Parallel()

	m := openDock(t, fullscreenContextViewer(120, 40))
	m.viewport.SetYOffset(7)

	resized := m.SetSize(140, 50)

	assert.Equal(t, 7, resized.viewport.YOffset())
	assert.Equal(t, 50, 1+resized.viewport.Height()+el.ContextDockHeight+1+2)
}

func TestDockOpeningAndClosingKeepsTheTopLine(t *testing.T) {
	t.Parallel()

	m := fullscreenContextViewer(120, 30)
	m.viewport.SetYOffset(9)

	opened := openDock(t, m)
	closed, _ := pressViewerKey(opened, "c")

	assert.Equal(t, 9, opened.viewport.YOffset())
	assert.Equal(t, 9, closed.viewport.YOffset())
}

func TestDockCursorFollowsTheTopLine(t *testing.T) {
	t.Parallel()

	m := openDock(t, fullscreenContextViewer(120, 20))
	total := m.viewport.TotalLineCount()
	require.Greater(t, total, 30)

	for offset := range total {
		m.viewport.SetYOffset(offset)
		want, _ := m.contextSeries.StepAt(m.messageAtLine(m.viewport.YOffset()))

		assert.Equal(t, want, m.cursorStep(), "offset %d", offset)
		wantStep := fmt.Sprintf("step %d of %d", want+1, len(m.contextSeries.Steps))
		assert.Contains(t, strings.Join(dockLines(m), "\n"), wantStep, "offset %d", offset)
	}
}

func TestDockCursorAboveTheFirstStepIsTheFirstStep(t *testing.T) {
	t.Parallel()

	m := openDock(t, fullscreenContextViewer(120, 20))
	m.viewport.GotoTop()

	assert.Equal(t, 0, m.cursorStep())
	assert.Contains(t, strings.Join(dockLines(m), "\n"), "system prompt")
}

func TestDockReadoutMatchesTheLedgerForTheSameStep(t *testing.T) {
	t.Parallel()

	m := fullscreenContextViewer(120, 20)
	m.opts = allTranscriptOptions()
	m = openDock(t, m.renderContent())
	line, ok := m.contextLines.LineOf(4)
	require.True(t, ok)
	m.viewport.SetYOffset(line)

	require.Equal(t, 4, m.messageAtLine(m.viewport.YOffset()))
	header := ansi.Strip(strings.Split(m.baseContent, "\n")[m.viewport.YOffset()])
	dock := strings.Join(dockLines(m), "\n")

	assert.Contains(t, header, "ctx 45.0k")
	assert.Contains(t, header, "23%")
	assert.Contains(t, dock, "ctx 45.0k  23%")
	assert.Contains(t, dock, "step 2 of 6")
	assert.Contains(t, dock, "grew +15.0k")
	assert.Contains(t, dock, "from Read: internal/parser.…")
}

func TestNotableJumpsVisitEveryReachableNotableStepOnce(t *testing.T) {
	t.Parallel()

	m := fullscreenContextViewer(120, 12)
	m.viewport.GotoTop()
	maxOffset := m.viewport.TotalLineCount() - m.viewport.Height()
	// Notable steps: 1 (+15k), 2 (reset), 3 (+40k). A step too close to the end
	// cannot reach the top line; the jump stops at the last scrollable line.
	var want []int
	for _, line := range el.NotableLines(m.contextSeries, m.contextLines) {
		reached := min(line, maxOffset)
		if len(want) == 0 || want[len(want)-1] != reached {
			want = append(want, reached)
		}
	}
	require.GreaterOrEqual(t, len(want), 2)

	forward := []int{}
	for range 20 {
		next, _ := pressViewerKey(m, "]")
		if next.viewport.YOffset() == m.viewport.YOffset() {
			break
		}
		m = next
		forward = append(forward, m.viewport.YOffset())
	}
	assert.Equal(t, want, forward)

	backward := []int{}
	for range 20 {
		prev, _ := pressViewerKey(m, "[")
		if prev.viewport.YOffset() == m.viewport.YOffset() {
			break
		}
		m = prev
		backward = append(backward, m.viewport.YOffset())
	}
	assert.IsDecreasing(t, backward)
	assert.Equal(t, el.NotableLines(m.contextSeries, m.contextLines)[0], backward[len(backward)-1])
}

func TestNotableJumpLandsOnTheStepMessage(t *testing.T) {
	t.Parallel()

	m := fullscreenContextViewer(120, 12)
	m.viewport.GotoTop()

	first, _ := pressViewerKey(m, "]")

	assert.Equal(t, 4, first.messageAtLine(first.viewport.YOffset()), "step 1 starts at message 4")
	second, _ := pressViewerKey(first, "]")
	assert.Equal(t, 12, second.messageAtLine(second.viewport.YOffset()), "the reset step starts at message 12")
	assert.Contains(t, ansi.Strip(strings.Split(second.baseContent, "\n")[second.viewport.YOffset()]), "context compacted",
		"the jump shows the reset divider")
}

func TestNotableJumpAtTheEndsNotifies(t *testing.T) {
	t.Parallel()

	m := fullscreenContextViewer(120, 40)
	m.viewport.GotoTop()

	_, atTop := pressViewerKey(m, "[")
	assert.Equal(t, "no earlier notable step", atTop)

	m.viewport.GotoBottom()
	_, atBottom := pressViewerKey(m, "]")
	assert.Equal(t, "no later notable step", atBottom)
}

func TestNotableJumpWorksWhateverIsHidden(t *testing.T) {
	t.Parallel()

	for _, opts := range toggleCombos() {
		m := fullscreenContextViewer(120, 12)
		m.opts = opts
		m = m.renderContent()
		m.viewport.GotoTop()

		visited := 0
		for range 30 {
			next, _ := pressViewerKey(m, "]")
			if next.viewport.YOffset() == m.viewport.YOffset() {
				break
			}
			require.Greater(t, next.viewport.YOffset(), m.viewport.YOffset(), "%+v", opts)
			m = next
			visited++
		}
		assert.LessOrEqual(t, visited, 3, "%+v", opts)
	}
}

func TestNotableJumpsAreAvailableWithoutTheDock(t *testing.T) {
	t.Parallel()

	m := newContextTestViewer(120, 40)
	m.viewport.GotoTop()

	next, _ := pressViewerKey(m, "]")

	assert.Positive(t, next.viewport.YOffset())
}

func TestDockHelpAndFooterOnlyOfferTheDockInFullscreen(t *testing.T) {
	t.Parallel()

	split := newContextTestViewer(120, 40)
	full := fullscreenContextViewer(120, 40)

	assert.NotContains(t, helpItemKeys(split.footerItems()), "c")
	assert.Contains(t, helpItemKeys(full.footerItems()), "c")
	assert.Contains(t, helpItemKeys(full.footerItems()), "C")
	assert.NotContains(t, sectionKeys(split.helpSections(nil)), "c")
	assert.Contains(t, sectionKeys(full.helpSections(nil)), "c")
	assert.Contains(t, sectionKeys(split.helpSections(nil)), "[ / ]")
}

func sectionKeys(sections []helpSection) []string {
	var keys []string
	for _, section := range sections {
		keys = append(keys, helpItemKeys(section.Items)...)
	}
	return keys
}
