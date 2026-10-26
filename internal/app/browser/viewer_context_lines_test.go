package browser

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func viewerLines(m viewerModel) []string {
	return strings.Split(ansi.Strip(m.baseContent), "\n")
}

// toggleCombos returns every combination of the viewer toggles that change
// transcript content (thinking, tools, results, system, sidechain, ledger).
func toggleCombos() []transcriptOptions {
	combos := make([]transcriptOptions, 0, 64)
	for bits := range 64 {
		combos = append(combos, transcriptOptions{
			showThinking:      bits&1 != 0,
			showTools:         bits&2 != 0,
			showToolResults:   bits&4 != 0,
			showSystem:        bits&8 != 0,
			hideSidechain:     bits&16 != 0,
			showContextLedger: bits&32 != 0,
		})
	}
	return combos
}

func TestMessageAtLineBeforeFirstMessageIsMinusOne(t *testing.T) {
	t.Parallel()

	m := newContextTestViewer(100, 40)

	require.NotEmpty(t, m.contextLines)
	assert.Equal(t, -1, m.messageAtLine(0), "conversation header lines have no message")
	assert.Equal(t, -1, m.messageAtLine(-5))
	assert.Equal(t, 0, m.messageAtLine(m.contextLines[0].Line))
}

func TestContextLineMapIsMonotonic(t *testing.T) {
	t.Parallel()

	for _, opts := range toggleCombos() {
		m := newContextTestViewer(100, 40)
		m.opts = opts
		m = m.renderContent()

		require.NotEmpty(t, m.contextLines)
		previousLine, previousMsg := -1, -1
		for _, anchor := range m.contextLines {
			require.Greater(t, anchor.Line, previousLine, "%+v", opts)
			require.Greater(t, anchor.MsgIndex, previousMsg-1, "%+v", opts)
			previousLine, previousMsg = anchor.Line, anchor.MsgIndex
		}
	}
}

func TestMessageAtLineReturnsTheMessageWhoseContentIsOnTheLine(t *testing.T) {
	t.Parallel()

	for _, opts := range toggleCombos() {
		m := newContextTestViewer(100, 40)
		m.opts = opts
		m = m.renderContent()

		for y, line := range viewerLines(m) {
			for index := range m.session.Messages {
				if !strings.Contains(line, contextToken(index)) {
					continue
				}
				assert.Equalf(t, index, m.messageAtLine(y), "line %d %q with %+v", y, line, opts)
			}
		}
	}
}

func TestContextLineMapSurvivesRenderCacheHits(t *testing.T) {
	t.Parallel()

	m := newContextTestViewer(100, 40)
	want := m.contextLines

	m.opts.showTools = true
	m = m.renderContent()
	m.opts.showTools = false
	m = m.renderContent()

	assert.Equal(t, want, m.contextLines)
}

func TestContextLineMapFollowsResize(t *testing.T) {
	t.Parallel()

	m := newContextTestViewer(120, 40)
	narrow := m.SetSize(50, 40)

	require.NotEmpty(t, narrow.contextLines)
	for y, line := range viewerLines(narrow) {
		if strings.Contains(line, contextToken(12)) {
			assert.Equal(t, 12, narrow.messageAtLine(y))
		}
	}
}

func TestSelectionModeProvidesItsOwnLineMap(t *testing.T) {
	t.Parallel()

	m := newContextTestViewer(100, 40)
	m = m.toggleSelectionMode()

	require.True(t, m.selectionMode)
	for y, line := range strings.Split(m.baseContent, "\n") {
		for index := range m.session.Messages {
			if strings.Contains(line, contextToken(index)) {
				assert.Equalf(t, index, m.messageAtLine(y), "selection line %d %q", y, line)
			}
		}
	}
}

func TestViewportOffsetMapsToMessage(t *testing.T) {
	t.Parallel()

	m := newContextTestViewer(100, 12)
	require.Greater(t, m.viewport.TotalLineCount(), 12)

	m.viewport.GotoBottom()

	assert.Equal(t, 17, m.messageAtLine(m.viewport.TotalLineCount()-1))
}

func TestConsecutiveAssistantTextKeepsOneBlankLineBetweenBlocks(t *testing.T) {
	t.Parallel()

	session := contextFixtureSession()
	session.Messages = session.Messages[:0]
	session.Messages = append(session.Messages,
		conv.Message{Role: conv.RoleUser, Text: "ask"},
		conv.Message{Role: conv.RoleAssistant, Text: "intro paragraph", Usage: contextUsage(30_000)},
		conv.Message{Role: conv.RoleAssistant, Text: "> quoted advice"},
		conv.Message{Role: conv.RoleAssistant, Text: "closing paragraph"},
	)
	m := newViewerModel(session, contextFixtureConversation(session), "dark", "2006-01-02 15:04", 100, 40)

	lines := viewerLines(m)
	var between []string
	capture := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.Contains(trimmed, "intro paragraph"):
			capture = true
		case strings.Contains(trimmed, "quoted advice"):
			capture = false
		case capture:
			between = append(between, trimmed)
		}
	}

	assert.Equal(t, []string{""}, between, "exactly one blank line separates the paragraph and the quote")
}
