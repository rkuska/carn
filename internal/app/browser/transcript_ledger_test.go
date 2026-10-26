package browser

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

func ledgerViewer(opts transcriptOptions, width, height int) viewerModel {
	m := newContextTestViewer(width, height)
	m.opts = opts
	return m.renderContent()
}

// headerLine returns the plain rendered header line that opens message index.
func headerLine(t *testing.T, m viewerModel, msgIndex int) string {
	t.Helper()

	for _, anchor := range m.contextLines {
		if anchor.MsgIndex != msgIndex {
			continue
		}
		lines := viewerLines(m)
		require.Less(t, anchor.Line, len(lines))
		// A reset divider shares the anchor of the message it precedes.
		for _, line := range lines[anchor.Line:min(anchor.Line+3, len(lines))] {
			if strings.Contains(line, "User") || strings.Contains(line, "Assistant") || strings.Contains(line, "System") {
				return line
			}
		}
		return lines[anchor.Line]
	}
	require.FailNowf(t, "no segment for message", "message %d", msgIndex)
	return ""
}

func TestLedgerAnnotatesAssistantHeadersWithTheStepOfTheirGroup(t *testing.T) {
	t.Parallel()

	m := ledgerViewer(allTranscriptOptions(), 120, 60)

	tests := []struct {
		name     string
		msgIndex int
		want     string
	}{
		{name: "first assistant group", msgIndex: 1, want: "ctx 30.0k  █▌         15%"},
		{name: "second assistant group", msgIndex: 4, want: "ctx 45.0k  ██▎        23%"},
		{name: "group after the reset", msgIndex: 12, want: "ctx 20.0k  █          10%"},
		{name: "group spanning two steps", msgIndex: 14, want: "ctx 60.0k → 63.0k  +3.0k  ███▏       32%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			line := headerLine(t, m, tt.msgIndex)

			assert.Contains(t, line, "Assistant")
			assert.True(t, strings.HasSuffix(strings.TrimRight(line, " "), tt.want), "%q should end with %q", line, tt.want)
			assert.Equal(t, m.contentWidth(), ansi.StringWidth(line))
		})
	}
}

func TestLedgerSkipsUserSystemAndSidechainHeaders(t *testing.T) {
	t.Parallel()

	m := ledgerViewer(allTranscriptOptions(), 120, 60)

	for _, msgIndex := range []int{0, 3, 6, 9, 11} {
		assert.NotContains(t, headerLine(t, m, msgIndex), "ctx ", "message %d", msgIndex)
	}
	assert.NotContains(t, headerLine(t, m, 7), "ctx ", "sidechain assistant header")
	assert.NotContains(t, headerLine(t, m, 10), "ctx ", "sidechain assistant header")
}

// contentToggleCombos is every combination of the content toggles with the
// Turn Ledger set to ledger.
func contentToggleCombos(ledger bool) []transcriptOptions {
	combos := toggleCombos()[:32] // the first half leaves showContextLedger off
	for i := range combos {
		combos[i].showContextLedger = ledger
	}
	return combos
}

// countLedgerAnnotations counts the annotated Assistant headers of m and checks
// each one shows the prompt of the step its message belongs to.
func countLedgerAnnotations(t *testing.T, m viewerModel, opts transcriptOptions) int {
	t.Helper()

	lines := viewerLines(m)
	annotated := 0
	for _, anchor := range m.contextLines {
		line := lines[anchor.Line]
		if !strings.Contains(line, "Assistant") || !strings.Contains(line, "ctx ") {
			continue
		}
		annotated++
		step, ok := m.contextSeries.StepAt(anchor.MsgIndex)
		require.True(t, ok, "%+v", opts)
		assert.Contains(t, line, "ctx "+el.FormatContextTokens(m.contextSeries.Steps[step].PromptTokens), "%+v", opts)
	}
	return annotated
}

func TestLedgerAnnotationFollowsTheSeriesWhateverIsHidden(t *testing.T) {
	t.Parallel()

	for _, opts := range contentToggleCombos(true) {
		assert.Positive(t, countLedgerAnnotations(t, ledgerViewer(opts, 120, 60), opts), "%+v", opts)
	}
}

func TestLedgerOffAnnotatesNoHeaderWhateverIsHidden(t *testing.T) {
	t.Parallel()

	for _, opts := range contentToggleCombos(false) {
		assert.Zero(t, countLedgerAnnotations(t, ledgerViewer(opts, 120, 60), opts), "%+v", opts)
	}
}

func TestLedgerNarrowWidthKeepsOnlyTheEndValue(t *testing.T) {
	t.Parallel()

	m := ledgerViewer(allTranscriptOptions(), 60, 60)

	line := headerLine(t, m, 14)

	assert.True(t, strings.HasSuffix(strings.TrimRight(line, " "), "ctx 63.0k"), line)
	assert.Equal(t, m.contentWidth(), ansi.StringWidth(line))
}

func TestLedgerGrowthTagSitsOnTheLastToolCallOfTheStep(t *testing.T) {
	t.Parallel()

	m := ledgerViewer(transcriptOptions{showTools: true, showContextLedger: true}, 120, 60)
	lines := viewerLines(m)

	var tagged []string
	for _, line := range lines {
		if isTagLine(line) {
			tagged = append(tagged, strings.TrimRight(line, " "))
		}
	}

	require.Len(t, tagged, 2)
	assert.True(t, strings.HasPrefix(tagged[0], "[Read: internal/parser.go] ·"), tagged[0])
	assert.True(t, strings.HasSuffix(tagged[0], " +15.0k"), tagged[0])
	assert.True(t, strings.HasPrefix(tagged[1], "[Edit: internal/parser.go] ·"), tagged[1])
	assert.True(t, strings.HasSuffix(tagged[1], " ▲ +40.0k"), tagged[1])
}

func TestLedgerGrowthTagFallsBackToTheToolResult(t *testing.T) {
	t.Parallel()

	m := ledgerViewer(transcriptOptions{showToolResults: true, showContextLedger: true}, 120, 60)

	var tagged []string
	for _, line := range viewerLines(m) {
		if isTagLine(line) {
			tagged = append(tagged, strings.TrimRight(line, " "))
		}
	}

	require.Len(t, tagged, 2)
	assert.Contains(t, tagged[0], "Read")
	assert.Contains(t, tagged[1], "Edit")
}

func TestLedgerGrowthTagAbsentWhenToolsAndResultsAreHidden(t *testing.T) {
	t.Parallel()

	m := ledgerViewer(transcriptOptions{showContextLedger: true}, 120, 60)

	for _, line := range viewerLines(m) {
		assert.False(t, isTagLine(line), line)
	}
}

func TestLedgerResetDividerSitsBeforeTheResetStep(t *testing.T) {
	t.Parallel()

	m := ledgerViewer(allTranscriptOptions(), 120, 60)
	lines := viewerLines(m)

	dividerLine := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "━━━ ↓ context compacted  45.0k → 20.0k  −25.0k ━") {
			dividerLine = i
		}
	}
	require.NotEqual(t, -1, dividerLine, "reset divider")
	assert.Equal(t, m.contentWidth(), ansi.StringWidth(lines[dividerLine]))

	// The divider starts the segments of the reset step message, so the line map
	// (and the Notable Step jump) lands on it.
	assert.Equal(t, 12, m.messageAtLine(dividerLine))
	assert.Equal(t, 11, m.messageAtLine(dividerLine-1))
	assert.Contains(t, lines[dividerLine+2], "Assistant")
}

func TestLedgerResetDividerNamesAResetWhenNothingWasCompacted(t *testing.T) {
	t.Parallel()

	session := contextFixtureSession()
	session.Meta.Performance.CompactionCount = 0
	m := newViewerModel(session, contextFixtureConversation(session), "dark", "2006-01-02 15:04", 120, 60)

	assert.Contains(t, ansi.Strip(m.baseContent), "↓ context reset  45.0k → 20.0k  −25.0k")
}

func TestLedgerResetDividerBreaksAGroupInTheMiddle(t *testing.T) {
	t.Parallel()

	session := contextFixtureSession()
	session.Messages = []conv.Message{
		{Role: conv.RoleUser, Text: "m00 go"},
		{Role: conv.RoleAssistant, Text: "m01 one", Usage: contextUsage(150_000)},
		{Role: conv.RoleAssistant, Text: "m02 two", Usage: contextUsage(30_000)},
		{Role: conv.RoleAssistant, Text: "m03 three"},
	}
	m := newViewerModel(session, contextFixtureConversation(session), "dark", "2006-01-02 15:04", 120, 60)

	var rendered []string
	for _, line := range viewerLines(m) {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "Assistant"):
			rendered = append(rendered, "header")
		case strings.HasPrefix(trimmed, "━━━"):
			rendered = append(rendered, "divider")
		case strings.Contains(trimmed, "m01"), strings.Contains(trimmed, "m02"), strings.Contains(trimmed, "m03"):
			rendered = append(rendered, trimmed)
		}
	}

	assert.Equal(t, []string{"header", "m01 one", "divider", "header", "m02 two", "m03 three"}, rendered)
}

func TestLedgerNeverChangesRawContent(t *testing.T) {
	t.Parallel()

	on := ledgerViewer(transcriptOptions{showTools: true, showToolResults: true, showContextLedger: true}, 120, 60)
	off := ledgerViewer(transcriptOptions{showTools: true, showToolResults: true}, 120, 60)

	assert.Equal(t, off.rawContent, on.rawContent)
	assert.NotEqual(t, off.baseContent, on.baseContent)
	for _, forbidden := range []string{"ctx ", "━━━", "▲", "+15.0k", "context compacted"} {
		assert.NotContains(t, on.rawContent, forbidden)
	}
}

func TestLedgerKeyTogglesAnnotationsAndKeepsScroll(t *testing.T) {
	t.Parallel()

	m := newContextTestViewer(120, 14)
	require.True(t, m.opts.showContextLedger, "the ledger is on by default")
	require.Contains(t, ansi.Strip(m.baseContent), "ctx ")

	// Message 9 opens the fourth group, above the first Context Reset.
	m = scrollToMessage(t, m, 9, 1)
	msg, delta := scrollPosition(t, m)

	updated, _ := pressViewerKey(m, "C")

	assert.False(t, updated.opts.showContextLedger)
	assert.NotContains(t, ansi.Strip(updated.baseContent), "ctx ")
	msgAfter, deltaAfter := scrollPosition(t, updated)
	assert.Equal(t, msg, msgAfter)
	assert.Equal(t, delta, deltaAfter)

	again, _ := pressViewerKey(updated, "C")
	assert.True(t, again.opts.showContextLedger)
	assert.Contains(t, ansi.Strip(again.baseContent), "ctx ")
}

func TestLedgerKeyIsRefusedWithoutContextData(t *testing.T) {
	t.Parallel()

	session := contextFixtureSession()
	for i := range session.Messages {
		session.Messages[i].Usage = conv.TokenUsage{}
	}
	m := newViewerModel(session, contextFixtureConversation(session), "dark", "2006-01-02 15:04", 120, 40)

	updated, notice := pressViewerKey(m, "C")

	assert.Equal(t, m.opts, updated.opts, "the hidden toggle does not flip")
	assert.Equal(t, m.baseContent, updated.baseContent)
	assert.Equal(t, "no context usage recorded in this conversation", notice)
}

func TestLedgerResetsToDefaultsWhenAViewerIsReopened(t *testing.T) {
	t.Parallel()

	m := newContextTestViewer(120, 40)
	m.opts.showContextLedger = false
	session := contextFixtureSession()

	reopened := m.resetForOpen(session, contextFixtureConversation(session), 40, m.theme, nil)

	assert.True(t, reopened.opts.showContextLedger)
}

func TestLedgerSelectionContentCarriesNoLedgerText(t *testing.T) {
	t.Parallel()

	m := newContextTestViewer(120, 40)
	m = m.toggleSelectionMode()

	for _, forbidden := range []string{"ctx ", "━━━", "▲", "context compacted"} {
		assert.NotContains(t, m.baseContent, forbidden)
	}
}

func TestLedgerTranscriptGolden(t *testing.T) {
	t.Parallel()

	opts := allTranscriptOptions()
	opts.hideSidechain = true
	m := ledgerViewer(opts, 120, 44)
	m.viewport.GotoTop()

	view := ansi.Strip(m.paneView(m.theme.ColorPrimary))

	assert.NotContains(t, view, "<no value>")
	golden.RequireEqual(t, []byte(view))
}

// isTagLine reports whether a rendered line carries a Growth Tag: a dotted
// leader followed by a signed token count.
func isTagLine(line string) bool {
	trimmed := strings.TrimRight(line, " ")
	return strings.Contains(trimmed, " ··· ") || strings.Contains(trimmed, "·· +") || strings.Contains(trimmed, "·· ▲")
}
