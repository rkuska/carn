package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTextHello = "hello"

func TestScanContentFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []conv.Message
		want     contentFlags
	}{
		{
			name:     "empty session",
			messages: nil,
			want:     contentFlags{},
		},
		{
			name:     "has thinking only",
			messages: []conv.Message{{Role: conv.RoleAssistant, Thinking: "deep thought"}},
			want:     contentFlags{hasThinking: true},
		},
		{
			name:     "has hidden thinking only",
			messages: []conv.Message{{Role: conv.RoleAssistant, HasHiddenThinking: true}},
			want:     contentFlags{hasThinking: true},
		},
		{
			name:     "has tool calls only",
			messages: []conv.Message{{Role: conv.RoleAssistant, ToolCalls: []conv.ToolCall{{Name: "Read"}}}},
			want:     contentFlags{hasToolCalls: true},
		},
		{
			name:     "has tool results only",
			messages: []conv.Message{{Role: conv.RoleUser, ToolResults: []conv.ToolResult{{Content: "x"}}}},
			want:     contentFlags{hasToolResults: true},
		},
		{
			name:     "has plans only",
			messages: []conv.Message{{Role: conv.RoleAssistant, Plans: []conv.Plan{{Content: "plan"}}}},
			want:     contentFlags{hasPlans: true},
		},
		{
			name:     "has sidechain only",
			messages: []conv.Message{{Role: conv.RoleAssistant, Text: "side", IsSidechain: true}},
			want:     contentFlags{hasSidechain: true},
		},
		{
			name: "has all",
			messages: []conv.Message{
				{
					Role:      conv.RoleAssistant,
					Thinking:  "t",
					ToolCalls: []conv.ToolCall{{Name: "W"}},
					Plans:     []conv.Plan{{Content: "plan"}},
				},
				{Role: conv.RoleUser, ToolResults: []conv.ToolResult{{Content: "x"}}},
				{Role: conv.RoleAssistant, Text: "side", IsSidechain: true},
			},
			want: contentFlags{hasThinking: true, hasToolCalls: true, hasToolResults: true, hasPlans: true, hasSidechain: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := scanContentFlags(tt.messages)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHelpBarAlwaysVisibleInFooter(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("help-always"), 120, 40)
	footer := m.footerView()

	assert.Contains(t, footer, "thinking")
}

func TestHelpViewGlowsWhenHiddenDataExists(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "glow-test",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "hello"},
			{Role: conv.RoleAssistant, Text: "hi", Thinking: "deep thought"},
		},
	}

	m := newTestViewer(session, 120, 40)

	// Thinking is off by default and there IS thinking content — should glow.
	helpOff := renderHelpItems(m.footerItems())

	m.opts.showThinking = true
	helpOn := renderHelpItems(m.footerItems())

	assert.Contains(
		t,
		helpOff,
		lipgloss.NewStyle().Foreground(colorPrimary).Render("-t"),
	)
	assert.Contains(
		t,
		helpOn,
		lipgloss.NewStyle().Foreground(colorAccent).Render("+t"),
	)
}

func TestHelpViewGlowsWhenHiddenThinkingExistsWithoutVisibleThinking(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "glow-hidden-thinking",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "hello"},
			{Role: conv.RoleAssistant, Text: "hi", HasHiddenThinking: true},
		},
	}

	m := newTestViewer(session, 120, 40)

	helpOff := renderHelpItems(m.footerItems())

	m.opts.showThinking = true
	helpOn := renderHelpItems(m.footerItems())

	assert.Contains(
		t,
		helpOff,
		lipgloss.NewStyle().Foreground(colorPrimary).Render("-t"),
	)
	assert.Contains(
		t,
		helpOn,
		lipgloss.NewStyle().Foreground(colorAccent).Render("+t"),
	)
}

func TestHelpViewNoGlowWhenNoHiddenData(t *testing.T) {
	t.Parallel()

	noThinkSession := testSession("no-glow-plain")
	m := newTestViewer(noThinkSession, 120, 40)
	helpOff := renderHelpItems(m.footerItems())

	m.opts.showThinking = true
	helpOn := renderHelpItems(m.footerItems())

	assert.Contains(
		t,
		helpOff,
		lipgloss.NewStyle().Foreground(colorAccent).Render("-t"),
	)
	assert.Contains(
		t,
		helpOn,
		lipgloss.NewStyle().Foreground(colorAccent).Render("+t"),
	)
}

func newTestViewer(session conv.Session, width, height int) viewerModel {
	return newViewerModel(session, singleSessionConversation(session.Meta), "dark", "2006-01-02 15:04", width, height)
}

func TestNewViewerModelStartsWithSearchInactive(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-init"), 120, 40)

	assert.False(t, m.searching)
	assert.False(t, m.searchInput.Focused())
}

func TestViewerSearchBindingUsesSlash(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-keys"), 120, 40)

	// Slash should enter search mode.
	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	assert.True(t, m.searching)
	assert.True(t, m.searchInput.Focused())
}

// testSessionLong creates a session with many messages so content exceeds a small viewport.
func testSessionLong(id string, keyword string) conv.Session {
	msgs := make([]conv.Message, 0, 40)
	for i := range 20 {
		msgs = append(msgs, conv.Message{Role: conv.RoleUser, Text: fmt.Sprintf("user message %d", i)})
		text := fmt.Sprintf("assistant response %d", i)
		if i == 15 {
			text = fmt.Sprintf("this contains %s in the middle", keyword)
		}
		msgs = append(msgs, conv.Message{Role: conv.RoleAssistant, Text: text})
	}
	return conv.Session{
		Meta: conv.SessionMeta{
			ID:        id,
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: msgs,
	}
}

func TestPerformSearchFindsMatchesBeyondViewport(t *testing.T) {
	t.Parallel()

	// Use a small viewport height so most content is off-screen.
	m := newTestViewer(testSessionLong("search-full", "UNIQUEWORD"), 120, 10)

	m.searchQuery = "UNIQUEWORD"
	m = m.performSearch()

	assert.NotEmpty(t, m.matches)
}

func TestPerformSearchStripsAnsiBeforeMatching(t *testing.T) {
	t.Parallel()

	// Create a session where the rendered output will contain ANSI codes around the keyword.
	// Glamour wraps text in ANSI escapes; searching raw ANSI would miss a plain-text query.
	m := newTestViewer(testSession("search-ansi"), 120, 40)

	// The rendered content will have ANSI escape codes around "hello" (glamour renders markdown).
	m.searchQuery = testTextHello
	m = m.performSearch()

	assert.NotEmpty(t, m.matches)
}

func TestPerformSearchRefreshesOnContentRerender(t *testing.T) {
	t.Parallel()

	session := testSessionLong("search-rerender", "TARGETWORD")
	session.Messages = append(session.Messages, conv.Message{
		Role:     conv.RoleAssistant,
		Text:     "thinking about TARGETWORD",
		Thinking: "deep thought about TARGETWORD",
	})

	m := newTestViewer(session, 120, 10)

	m.searchQuery = "TARGETWORD"
	m = m.performSearch()
	matchesBefore := len(m.matches)

	// Toggle thinking on — adds the thinking block which contains TARGETWORD.
	m.opts.showThinking = true
	m = m.renderContent()

	assert.Greater(t, len(m.matches), matchesBefore)
}

func TestFooterShowsNoMatchesWhenSearchHasZeroResults(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("footer-zero"), 120, 40)

	m.searchQuery = "XYZNONEXISTENT"
	m = m.performSearch()

	footer := m.footerView()

	assert.NotContains(t, footer, "1/0")
	assert.Contains(t, footer, "no matches")
}

func TestFooterShowsMatchCountWhenSearchHasResults(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("footer-match"), 120, 40)

	m.searchQuery = testTextHello
	m = m.performSearch()

	require.NotEmpty(t, m.matches)

	footer := m.footerView()

	expected := fmt.Sprintf("1/%d", len(m.matches))
	assert.Contains(t, footer, expected)
}

func TestViewerFooterUsesSeparateStatusRow(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-footer-status"), 120, 40)
	m.notification = errorNotification("resume failed: directory not found: /tmp/project").notification

	lines := strings.Split(m.footerView(), "\n")
	require.Len(t, lines, 2)

	helpLine := ansi.Strip(lines[0])
	statusLine := ansi.Strip(lines[1])

	assert.Contains(t, helpLine, "thinking")
	assert.NotContains(t, helpLine, "resume failed")
	assert.Contains(t, statusLine, "resume failed: directory not found")
}

func TestViewerFooterSearchKeepsStatusRow(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-search-footer"), 120, 40)
	m.searching = true
	m.searchInput.Focus()
	m.searchInput.SetValue(testTextHello)
	m.notification = infoNotification("search ready").notification

	lines := strings.Split(m.footerView(), "\n")
	require.Len(t, lines, 2)

	searchLine := ansi.Strip(lines[0])
	statusLine := ansi.Strip(lines[1])

	assert.Contains(t, searchLine, "/hello")
	assert.Contains(t, statusLine, "search ready")
}

func TestViewerFooterOrdersItemsByWorkflow(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-footer-order"), 120, 40)

	assert.Equal(
		t,
		[]string{"/", "n/N", "t", "T", "R", "s", "m", "y", "e", "?", "q/esc"},
		helpItemKeys(m.footerItems()),
	)
}

func TestViewerFooterShowsPlanToggleWhenPlansExist(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "conv.Plan-toggle",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "hello", Plans: []conv.Plan{{FilePath: "a.md", Content: "conv.Plan"}}},
			{Role: conv.RoleAssistant, Text: "hi"},
		},
	}
	m := newTestViewer(session, 120, 40)

	keys := helpItemKeys(m.footerItems())
	assert.Contains(t, keys, "p")
	assert.Contains(
		t,
		renderHelpItems(m.footerItems()),
		lipgloss.NewStyle().Foreground(colorPrimary).Render("-p"),
	)

	// p should appear after the last transcript toggle and before action items
	pIdx := -1
	sIdx := -1
	yIdx := -1
	for i, k := range keys {
		switch k {
		case "p":
			pIdx = i
		case "s":
			sIdx = i
		case "y":
			yIdx = i
		}
	}
	assert.Greater(t, pIdx, sIdx, "conv.Plan toggle should come after sidechain toggle")
	assert.Less(t, pIdx, yIdx, "conv.Plan toggle should come before action items")

	m.planExpanded = true

	assert.Contains(
		t,
		renderHelpItems(m.footerItems()),
		lipgloss.NewStyle().Foreground(colorAccent).Render("+p"),
	)
}

func TestViewerEscapeCancelsActiveSearch(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-search-cancel"), 120, 40)
	m.searchQuery = testTextHello
	m = m.performSearch()
	require.NotEmpty(t, m.matches)

	m.searching = true
	m.searchInput.Focus()
	m.searchInput.SetValue(testTextHello)

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	assert.False(t, m.searching)
	assert.Empty(t, m.searchQuery)
	assert.Empty(t, m.matches)
	assert.Equal(t, 0, m.currentMatch)
	assert.Empty(t, m.searchInput.Value())
}

func TestViewerViewKeepsWindowHeightWithTwoLineFooter(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-height"), 120, 40)

	assert.Equal(t, m.height, lipgloss.Height(m.View()))
}

func TestViewerUpdateShowsAndClearsNotifications(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-notification"), 120, 40)

	m, _ = m.Update(errorNotification("resume failed: directory not found: /tmp/missing"))
	assert.Equal(t, "resume failed: directory not found: /tmp/missing", m.notification.text)
	assert.Equal(t, notificationError, m.notification.kind)

	m, _ = m.Update(clearNotificationMsg{})
	assert.Empty(t, m.notification.text)
}

func TestRenderRoleHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		role      conv.Role
		wantLabel string
	}{
		{
			name:      "user role shows User badge",
			role:      conv.RoleUser,
			wantLabel: "User",
		},
		{
			name:      "assistant role shows Assistant badge",
			role:      conv.RoleAssistant,
			wantLabel: "Assistant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderRoleHeader(tt.role, 80)
			stripped := ansi.Strip(got)

			assert.Contains(t, stripped, tt.wantLabel)
			assert.Contains(t, stripped, "─")
		})
	}
}

func TestViewerSearchHighlightsMatchedText(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("search-highlight"), 120, 40)

	contentBefore := m.paneView(colorPrimary)
	baseBefore := m.viewport.GetContent()

	m.searchQuery = testTextHello
	m = m.performSearch()
	require.NotEmpty(t, m.matches)

	contentAfter := m.paneView(colorPrimary)

	// The viewport content should differ because matches are highlighted.
	assert.NotEqual(t, contentBefore, contentAfter)
	// Stripped content should be the same (only styling changed).
	assert.Equal(t, ansi.Strip(contentBefore), ansi.Strip(contentAfter))
	assert.Equal(t, baseBefore, m.viewport.GetContent())
}

func TestViewerSearchCurrentMatchMovesOnJump(t *testing.T) {
	t.Parallel()

	session := testSessionLong("jump-highlight", "JUMPWORD")
	// Add more messages with the keyword to get multiple matches.
	session.Messages = append(session.Messages,
		conv.Message{Role: conv.RoleUser, Text: "JUMPWORD again"},
		conv.Message{Role: conv.RoleAssistant, Text: "reply with JUMPWORD"},
	)

	m := newTestViewer(session, 120, 10)

	m.searchQuery = "JUMPWORD"
	m = m.performSearch()
	require.Greater(t, len(m.matches), 1)

	contentAt0 := m.paneView(colorPrimary)

	m = m.jumpToMatch(1)
	contentAt1 := m.paneView(colorPrimary)

	// Different current match should produce different highlighted content.
	assert.NotEqual(t, contentAt0, contentAt1)
	assert.Equal(t, m.baseContent, m.viewport.GetContent())
}

func TestViewerSearchClearRemovesHighlights(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("clear-highlight"), 120, 40)

	// Capture the un-highlighted content.
	contentClean := m.paneView(colorPrimary)

	m.searchQuery = testTextHello
	m = m.performSearch()
	require.NotEmpty(t, m.matches)

	// After clearing, content should return to the original un-highlighted state.
	m = m.clearSearch()
	contentAfterClear := m.paneView(colorPrimary)

	assert.Equal(t, contentClean, contentAfterClear)
	assert.Equal(t, m.baseContent, m.viewport.GetContent())
}

func TestViewerSearchKeepsViewportContentStableAcrossRerender(t *testing.T) {
	t.Parallel()

	session := testSessionLong("search-stable-rerender", "TARGETWORD")
	session.Messages = append(session.Messages, conv.Message{
		Role:     conv.RoleAssistant,
		Text:     "thinking about TARGETWORD",
		Thinking: "deep thought about TARGETWORD",
	})

	m := newTestViewer(session, 120, 10)
	m.searchQuery = "TARGETWORD"
	m = m.performSearch()
	require.NotEmpty(t, m.matches)

	contentBefore := m.viewport.GetContent()

	m.opts.showThinking = true
	m = m.renderContent()

	assert.Equal(t, m.baseContent, m.viewport.GetContent())
	assert.NotEqual(t, contentBefore, m.viewport.GetContent())
	assert.NotEmpty(t, m.matches)
	assert.Equal(t, 0, m.currentMatch)
}

func TestViewerSearchSurvivesResize(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSessionLong("search-resize", "RESIZEWORD"), 120, 10)
	m.searchQuery = "RESIZEWORD"
	m = m.performSearch()
	require.NotEmpty(t, m.matches)

	m = m.SetSize(100, 12)

	assert.Equal(t, "RESIZEWORD", m.searchQuery)
	require.NotEmpty(t, m.matches)
	assert.Equal(t, m.baseContent, m.viewport.GetContent())
	assert.Equal(t, 0, m.currentMatch)
	assert.Contains(t, m.footerView(), fmt.Sprintf("1/%d", len(m.matches)))
}

func TestPerformSearchCountsOccurrencesNotLines(t *testing.T) {
	t.Parallel()

	// A single line with "foo" appearing 3 times.
	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "occ-count",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "foo foo foo"},
		},
	}
	m := newTestViewer(session, 120, 40)

	m.searchQuery = "foo"
	m = m.performSearch()

	assert.GreaterOrEqual(t, len(m.matches), 3)
}

func TestJumpToMatchCyclesThroughOccurrencesOnSameLine(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "jump-occ",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "aaa bbb aaa ccc aaa"},
		},
	}
	m := newTestViewer(session, 120, 40)

	m.searchQuery = "aaa"
	m = m.performSearch()
	require.GreaterOrEqual(t, len(m.matches), 3)

	// Find the first 3 matches that are on the same line as matches[0].
	targetLine := m.matches[0].line
	sameLineCount := 0
	for _, occ := range m.matches {
		if occ.line == targetLine {
			sameLineCount++
		}
	}
	require.GreaterOrEqual(t, sameLineCount, 3)

	assert.Equal(t, 0, m.currentMatch)
	m = m.jumpToMatch(1)
	assert.Equal(t, 1, m.currentMatch)
	m = m.jumpToMatch(1)
	assert.Equal(t, 2, m.currentMatch)
}

func TestFooterShowsOccurrenceCount(t *testing.T) {
	t.Parallel()

	// Two lines: "xxx xxx" and "xxx" → at least 3 occurrences.
	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "footer-occ",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "xxx xxx"},
			{Role: conv.RoleAssistant, Text: "xxx"},
		},
	}
	m := newTestViewer(session, 120, 40)

	m.searchQuery = "xxx"
	m = m.performSearch()

	require.GreaterOrEqual(t, len(m.matches), 3)

	footer := m.footerView()
	expected := fmt.Sprintf("1/%d", len(m.matches))
	assert.Contains(t, footer, expected)
}
