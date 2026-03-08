package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanContentFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []message
		want     contentFlags
	}{
		{
			name:     "empty session",
			messages: nil,
			want:     contentFlags{},
		},
		{
			name:     "has thinking only",
			messages: []message{{role: roleAssistant, thinking: "deep thought"}},
			want:     contentFlags{hasThinking: true},
		},
		{
			name:     "has tool calls only",
			messages: []message{{role: roleAssistant, toolCalls: []toolCall{{name: "Read"}}}},
			want:     contentFlags{hasToolCalls: true},
		},
		{
			name:     "has tool results only",
			messages: []message{{role: roleUser, toolResults: []toolResult{{content: "x"}}}},
			want:     contentFlags{hasToolResults: true},
		},
		{
			name:     "has sidechain only",
			messages: []message{{role: roleAssistant, text: "side", isSidechain: true}},
			want:     contentFlags{hasSidechain: true},
		},
		{
			name: "has all",
			messages: []message{
				{role: roleAssistant, thinking: "t", toolCalls: []toolCall{{name: "W"}}},
				{role: roleUser, toolResults: []toolResult{{content: "x"}}},
				{role: roleAssistant, text: "side", isSidechain: true},
			},
			want: contentFlags{hasThinking: true, hasToolCalls: true, hasToolResults: true, hasSidechain: true},
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

	session := sessionFull{
		meta: sessionMeta{
			id:        "glow-test",
			timestamp: time.Now(),
			project:   project{displayName: "test"},
		},
		messages: []message{
			{role: roleUser, text: "hello"},
			{role: roleAssistant, text: "hi", thinking: "deep thought"},
		},
	}

	m := newTestViewer(session, 120, 40)

	// Thinking is off by default and there IS thinking content — should glow.
	helpOff := m.footerView()

	m.opts.showThinking = true
	helpOn := m.footerView()

	assert.NotEqual(t, helpOff, helpOn)
}

func TestHelpViewNoGlowWhenNoHiddenData(t *testing.T) {
	t.Parallel()

	// Session with no thinking data — glow should not activate,
	// but the +/- prefix still changes with toggle state.
	session := sessionFull{
		meta: sessionMeta{
			id:        "no-glow",
			timestamp: time.Now(),
			project:   project{displayName: "test"},
		},
		messages: []message{
			{role: roleUser, text: "hello"},
			{role: roleAssistant, text: "hi", thinking: "deep thought"},
		},
	}
	m := newTestViewer(session, 120, 40)

	helpOff := m.footerView()

	m.opts.showThinking = true
	helpOn := m.footerView()

	// With thinking data, toggling changes both glow and prefix.
	assert.NotEqual(t, helpOff, helpOn)

	// Session with NO thinking data — glow should not activate.
	noThinkSession := testSession("no-glow-plain")
	m2 := newTestViewer(noThinkSession, 120, 40)
	footer := m2.footerView()

	// The key should show -t (off) but not glow since there's no thinking content.
	assert.Contains(t, footer, "-t")
}

func newTestViewer(session sessionFull, width, height int) viewerModel {
	return newViewerModel(session, singleSessionConversation(session.meta), "dark", width, height)
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
func testSessionLong(id string, keyword string) sessionFull {
	msgs := make([]message, 0, 40)
	for i := range 20 {
		msgs = append(msgs, message{role: roleUser, text: fmt.Sprintf("user message %d", i)})
		text := fmt.Sprintf("assistant response %d", i)
		if i == 15 {
			text = fmt.Sprintf("this contains %s in the middle", keyword)
		}
		msgs = append(msgs, message{role: roleAssistant, text: text})
	}
	return sessionFull{
		meta: sessionMeta{
			id:        id,
			timestamp: time.Now(),
			project:   project{displayName: "test"},
		},
		messages: msgs,
	}
}

func TestPerformSearchFindsMatchesBeyondViewport(t *testing.T) {
	t.Parallel()

	// Use a small viewport height so most content is off-screen.
	m := newTestViewer(testSessionLong("search-full", "UNIQUEWORD"), 120, 10)

	m.searchQuery = "UNIQUEWORD"
	m.performSearch()

	assert.NotEmpty(t, m.matchIndices)
}

func TestPerformSearchStripsAnsiBeforeMatching(t *testing.T) {
	t.Parallel()

	// Create a session where the rendered output will contain ANSI codes around the keyword.
	// Glamour wraps text in ANSI escapes; searching raw ANSI would miss a plain-text query.
	m := newTestViewer(testSession("search-ansi"), 120, 40)

	// The rendered content will have ANSI escape codes around "hello" (glamour renders markdown).
	m.searchQuery = "hello"
	m.performSearch()

	assert.NotEmpty(t, m.matchIndices)
}

func TestPerformSearchRefreshesOnContentRerender(t *testing.T) {
	t.Parallel()

	session := testSessionLong("search-rerender", "TARGETWORD")
	session.messages = append(session.messages, message{
		role:     roleAssistant,
		text:     "thinking about TARGETWORD",
		thinking: "deep thought about TARGETWORD",
	})

	m := newTestViewer(session, 120, 10)

	m.searchQuery = "TARGETWORD"
	m.performSearch()
	matchesBefore := len(m.matchIndices)

	// Toggle thinking on — adds the thinking block which contains TARGETWORD.
	m.opts.showThinking = true
	m.renderContent()

	assert.Greater(t, len(m.matchIndices), matchesBefore)
}

func TestFooterShowsNoMatchesWhenSearchHasZeroResults(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("footer-zero"), 120, 40)

	m.searchQuery = "XYZNONEXISTENT"
	m.performSearch()

	footer := m.footerView()

	assert.NotContains(t, footer, "1/0")
	assert.Contains(t, footer, "no matches")
}

func TestFooterShowsMatchCountWhenSearchHasResults(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("footer-match"), 120, 40)

	m.searchQuery = "hello"
	m.performSearch()

	require.NotEmpty(t, m.matchIndices)

	footer := m.footerView()

	expected := fmt.Sprintf("1/%d", len(m.matchIndices))
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
	m.searchInput.SetValue("hello")
	m.notification = infoNotification("search ready").notification

	lines := strings.Split(m.footerView(), "\n")
	require.Len(t, lines, 2)

	searchLine := ansi.Strip(lines[0])
	statusLine := ansi.Strip(lines[1])

	assert.Contains(t, searchLine, "/hello")
	assert.Contains(t, statusLine, "search ready")
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
		role      role
		wantLabel string
	}{
		{
			name:      "user role shows User badge",
			role:      roleUser,
			wantLabel: "User",
		},
		{
			name:      "assistant role shows Assistant badge",
			role:      roleAssistant,
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
