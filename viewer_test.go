package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestNewViewerModelStartsWithSearchInactive(t *testing.T) {
	t.Parallel()

	m := newViewerModel(testSession("viewer-init"), 120, 40)

	if m.searching {
		t.Fatal("expected searching to be false on init")
	}
	if m.searchInput.Focused() {
		t.Fatal("expected search input to be blurred on init")
	}
}

func TestViewerSearchBindingUsesSlash(t *testing.T) {
	t.Parallel()

	m := newViewerModel(testSession("viewer-keys"), 120, 40)

	// Slash should enter search mode.
	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	if !m.searching {
		t.Fatal("expected slash key to activate search mode")
	}
	if !m.searchInput.Focused() {
		t.Fatal("expected search input to be focused after slash")
	}
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
	m := newViewerModel(testSessionLong("search-full", "UNIQUEWORD"), 120, 10)

	m.searchQuery = "UNIQUEWORD"
	m.performSearch()

	if len(m.matchIndices) == 0 {
		t.Fatal("expected at least one match for UNIQUEWORD in full content")
	}
}

func TestPerformSearchStripsAnsiBeforeMatching(t *testing.T) {
	t.Parallel()

	// Create a session where the rendered output will contain ANSI codes around the keyword.
	// Glamour wraps text in ANSI escapes; searching raw ANSI would miss a plain-text query.
	m := newViewerModel(testSession("search-ansi"), 120, 40)

	// The rendered content will have ANSI escape codes around "hello" (glamour renders markdown).
	m.searchQuery = "hello"
	m.performSearch()

	if len(m.matchIndices) == 0 {
		t.Fatal("expected search to find 'hello' even when content has ANSI codes")
	}
}

func TestPerformSearchRefreshesOnContentRerender(t *testing.T) {
	t.Parallel()

	session := testSessionLong("search-rerender", "TARGETWORD")
	session.messages = append(session.messages, message{
		role:     roleAssistant,
		text:     "thinking about TARGETWORD",
		thinking: "deep thought about TARGETWORD",
	})

	m := newViewerModel(session, 120, 10)

	m.searchQuery = "TARGETWORD"
	m.performSearch()
	matchesBefore := len(m.matchIndices)

	// Toggle thinking on — adds the thinking block which contains TARGETWORD.
	m.opts.showThinking = true
	m.renderContent()

	if len(m.matchIndices) <= matchesBefore {
		t.Fatalf("expected more matches after toggling thinking on, got %d (was %d)",
			len(m.matchIndices), matchesBefore)
	}
}

func TestFooterShowsNoMatchesWhenSearchHasZeroResults(t *testing.T) {
	t.Parallel()

	m := newViewerModel(testSession("footer-zero"), 120, 40)

	m.searchQuery = "XYZNONEXISTENT"
	m.performSearch()

	footer := m.footerView()

	if strings.Contains(footer, "1/0") {
		t.Fatal("footer should not show '1/0' when there are no matches")
	}
	if !strings.Contains(footer, "no matches") {
		t.Fatalf("footer should show 'no matches', got: %s", footer)
	}
}

func TestFooterShowsMatchCountWhenSearchHasResults(t *testing.T) {
	t.Parallel()

	m := newViewerModel(testSession("footer-match"), 120, 40)

	m.searchQuery = "hello"
	m.performSearch()

	if len(m.matchIndices) == 0 {
		t.Fatal("expected at least one match for 'hello'")
	}

	footer := m.footerView()

	expected := fmt.Sprintf("1/%d", len(m.matchIndices))
	if !strings.Contains(footer, expected) {
		t.Fatalf("footer should contain '%s', got: %s", expected, footer)
	}
}
