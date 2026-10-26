package browser

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

// contextMatrixSession extends the context fixture with a Codex-style response:
// the usage sits on a text message and the tool call lives in the next message
// of the same request.
func contextMatrixSession() conv.Session {
	session := contextFixtureSession()
	bash := conv.ToolCall{Name: "Bash", Summary: "go vet ./..."}
	session.Messages = append(session.Messages,
		conv.Message{Role: conv.RoleUser, Text: "m18 one more check"},
		conv.Message{Role: conv.RoleAssistant, Text: "m19 running vet", Usage: contextUsage(70_000)},
		conv.Message{Role: conv.RoleAssistant, ToolCalls: []conv.ToolCall{bash}},
		conv.Message{Role: conv.RoleUser, ToolResults: []conv.ToolResult{{ToolName: "Bash", Content: "m21 vet ok"}}},
		conv.Message{Role: conv.RoleAssistant, Text: "m22 vet is clean", Usage: contextUsage(72_000)},
	)
	return session
}

func matrixViewer(session conv.Session) viewerModel {
	return newViewerModel(session, contextFixtureConversation(session), "dark", "2006-01-02 15:04", 120, 30).
		SetFullscreen(true)
}

type matrixCase struct {
	opts transcriptOptions
	dock bool
}

func matrixCases() []matrixCase {
	combos := toggleCombos()
	cases := make([]matrixCase, 0, 2*len(combos))
	for _, opts := range combos {
		for _, dock := range []bool{false, true} {
			cases = append(cases, matrixCase{opts: opts, dock: dock})
		}
	}
	return cases
}

func (c matrixCase) String() string {
	return fmt.Sprintf("%+v dock=%t", c.opts, c.dock)
}

// headerLines lists the rendered role header lines with the message that
// opened each group.
type renderedHeader struct {
	line     int
	msgIndex int
	text     string
}

func renderedHeaders(m viewerModel) []renderedHeader {
	lines := viewerLines(m)
	var headers []renderedHeader
	for _, anchor := range m.contextLines {
		text := strings.TrimSpace(lines[anchor.Line])
		for _, role := range []string{"User ─", "Assistant ─", "System ─"} {
			if strings.HasPrefix(text, role) {
				headers = append(headers, renderedHeader{line: anchor.Line, msgIndex: anchor.MsgIndex, text: text})
			}
		}
	}
	return headers
}

func TestContextToggleMatrix(t *testing.T) {
	t.Parallel()

	session := contextMatrixSession()
	want := matrixViewer(session).contextSeries
	require.Len(t, want.Steps, 8)

	for _, tc := range matrixCases() {
		t.Run(tc.String(), func(t *testing.T) {
			t.Parallel()

			m := matrixViewer(session)
			m.opts = tc.opts
			m.dockOpen = tc.dock
			m = m.applyViewportHeight().renderContent()

			assert.Equal(t, want, m.contextSeries, "the series never depends on toggles")
			assertMatrixHeaders(t, m, tc)
			assertMatrixGrowthTags(t, m, tc)
			assertMatrixLineMap(t, m)
			assertMatrixDock(t, m, tc)
		})
	}
}

func assertMatrixHeaders(t *testing.T, m viewerModel, tc matrixCase) {
	t.Helper()

	series := m.contextSeries
	headers := renderedHeaders(m)
	for i, header := range headers {
		annotated := strings.Contains(header.text, " ctx ")
		isAssistant := strings.HasPrefix(header.text, "Assistant")
		sidechain := m.session.Messages[header.msgIndex].IsSidechain
		if !tc.opts.showContextLedger || !isAssistant || sidechain {
			assert.False(t, annotated, "%v: %q", tc, header.text)
			continue
		}
		end := len(m.session.Messages)
		if i+1 < len(headers) {
			end = headers[i+1].msgIndex
		}
		last, ok := series.StepAt(max(end-1, header.msgIndex))
		if !ok {
			assert.False(t, annotated, "%v: no step in the group of %q", tc, header.text)
			continue
		}
		first, ok := series.StepAt(header.msgIndex)
		if !ok {
			first = 0
		}
		wantStart := "ctx " + el.FormatContextTokens(series.Steps[first].PromptTokens)
		assert.Contains(t, header.text, wantStart, "%v", tc)
		endTokens := series.Steps[last].PromptTokens
		if series.Steps[first].PromptTokens != endTokens {
			assert.Contains(t, header.text, " → "+el.FormatContextTokens(endTokens), "%v", tc)
		}
		percent := fmt.Sprintf("%d%%", int(math.Round(series.Share(endTokens)*100)))
		assert.True(t, strings.HasSuffix(header.text, percent), "%v: %q should end with %s", tc, header.text, percent)
	}
}

func assertMatrixGrowthTags(t *testing.T, m viewerModel, tc matrixCase) {
	t.Helper()

	series := m.contextSeries
	for y, line := range viewerLines(m) {
		if !isTagLine(line) {
			continue
		}
		require.True(t, tc.opts.showContextLedger, "%v: tag on line %d without the ledger", tc, y)
		msg := m.messageAtLine(y)
		step, ok := series.StepAt(msg)
		require.True(t, ok, "%v: tag on line %d before the first step", tc, y)
		assert.Less(t, msg, series.Steps[step].NextMessageIndex, "%v: tag line %d outside its step", tc, y)
		assert.GreaterOrEqual(t, series.Steps[step].Growth, series.MinGrowth())
		wantTag := el.FormatSignedContextTokens(series.Steps[step].Growth)
		assert.True(t, strings.HasSuffix(strings.TrimRight(line, " "), wantTag), "%v: %q", tc, line)
		assert.False(t, m.session.Messages[msg].IsSidechain, "%v: tag on a sidechain message", tc)
	}
}

func assertMatrixLineMap(t *testing.T, m viewerModel) {
	t.Helper()

	require.NotEmpty(t, m.contextLines)
	previous := -1
	for _, anchor := range m.contextLines {
		require.Greater(t, anchor.Line, previous)
		previous = anchor.Line
	}
	for y := m.contextLines[0].Line; y < m.viewport.TotalLineCount(); y++ {
		require.GreaterOrEqual(t, m.messageAtLine(y), 0, "line %d is covered", y)
	}
}

func assertMatrixDock(t *testing.T, m viewerModel, tc matrixCase) {
	t.Helper()

	assert.Equal(t, tc.dock, m.dockHeight() == el.ContextDockHeight, "%v", tc)
	for offset := range m.viewport.TotalLineCount() {
		m.viewport.SetYOffset(offset)
		want, _ := m.contextSeries.StepAt(m.messageAtLine(m.viewport.YOffset()))
		require.Equal(t, want, m.cursorStep(), "%v offset %d", tc, offset)
		// A closed dock renders "", so the readout shows the cursor step
		// exactly when the dock is open.
		wantStep := fmt.Sprintf("step %d of %d", want+1, len(m.contextSeries.Steps))
		require.Equal(t, tc.dock, strings.Contains(ansi.Strip(m.dockView()), wantStep), "%v offset %d", tc, offset)
	}
}

// TestContextToggleSequenceKeepsTheTopMessage walks every combination of the
// content toggles one key press at a time and checks that the message on the
// top line survives, or that the nearest following message takes its place.
func TestContextToggleSequenceKeepsTheTopMessage(t *testing.T) {
	t.Parallel()

	session := contextMatrixSession()
	m := matrixViewer(session)
	m.viewport.SetHeight(8)
	toggles := []string{"t", "T", "R", "m", "s", "C"}

	// Gray code: every step flips exactly one toggle.
	for state := 1; state < 1<<len(toggles); state++ {
		flipped := 0
		for bit := range toggles {
			if (state^(state>>1))^((state-1)^((state-1)>>1)) == 1<<bit {
				flipped = bit
			}
		}
		target := []int{4, 12, 14, 19, 22}[state%5]
		if line, ok := m.contextLines.LineOf(target); ok {
			m.viewport.SetYOffset(line)
		}
		before := m.messageAtLine(m.viewport.YOffset())
		next, _ := pressViewerKey(m, toggles[flipped])
		next.viewport.SetHeight(8)

		after := next.messageAtLine(next.viewport.YOffset())
		maxOffset := next.viewport.TotalLineCount() - next.viewport.Height()
		if next.viewport.YOffset() < maxOffset {
			visible, ok := next.contextLines.LineOf(before)
			firstOfBefore := ok && next.messageAtLine(visible) == before
			if firstOfBefore {
				require.Equal(t, before, after, "state %d after %q", state, toggles[flipped])
			} else {
				require.Greater(t, after, before, "state %d after %q", state, toggles[flipped])
			}
		}
		m = next
	}
}
