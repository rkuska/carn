package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	testConversationIDPrimary   = "id-1"
	testConversationIDSecondary = "id-2"
)

func testBrowser(t *testing.T) browserModel {
	t.Helper()
	initPalette(true)

	b := newBrowserModel(context.Background(), t.TempDir(), "dark")
	b.width = 120
	b.height = 40
	b.updateLayout()
	return b
}

func testSession(id string) sessionFull {
	return sessionFull{
		meta: sessionMeta{
			id:        id,
			timestamp: time.Now(),
			project:   project{displayName: "test"},
		},
		messages: []message{
			{role: roleUser, text: "hello"},
			{role: roleAssistant, text: "hi there"},
		},
	}
}

func testBrowserSessionLong(id string, keyword string) sessionFull {
	msgs := make([]message, 0, 40)
	for i := range 20 {
		msgs = append(msgs, message{role: roleUser, text: "user message"})
		text := "assistant response"
		if i == 15 {
			text = "assistant response with " + keyword
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

func testConv(id string) conversation {
	return conversation{
		name:    "test-slug",
		project: project{dirName: "test", displayName: "test"},
		sessions: []sessionMeta{
			{id: id, slug: "test-slug", timestamp: time.Now(), project: project{displayName: "test"}},
		},
	}
}

func testLongConv(id string) conversation {
	return conversation{
		name: "very-long-conversation-name-that-should-wrap-in-the-split-pane-because-the-list-is-narrow",
		project: project{
			dirName:     "test",
			displayName: "Projects/claude-search/with-a-very-long-project-name",
		},
		sessions: []sessionMeta{
			{
				id:           id,
				slug:         "very-long-conversation-name-that-should-wrap-in-the-split-pane-because-the-list-is-narrow",
				timestamp:    time.Now(),
				project:      project{displayName: "Projects/claude-search/with-a-very-long-project-name"},
				firstMessage: strings.Repeat("This is a long first message that wraps. ", 4),
				model:        "claude-opus-4-1",
				version:      "1.2.3",
				messageCount: 20,
			},
		},
	}
}

func TestBrowserEnterOpensTranscriptSplit(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conv := testConv(testConversationIDPrimary)
	b.list.SetItems([]list.Item{conv})
	b.list.Select(0)

	b, cmd := b.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if b.transcriptMode != transcriptSplit {
		t.Fatalf("transcriptMode = %v, want %v", b.transcriptMode, transcriptSplit)
	}
	if b.loadingConversationID != conv.id() {
		t.Fatalf("loadingConversationID = %q, want %q", b.loadingConversationID, conv.id())
	}
	if cmd == nil {
		t.Fatal("expected open transcript command")
	}
	if got, want := b.list.Width(), b.listPaneWidth()-2; got != want {
		t.Fatalf("list width = %d, want %d after entering split mode", got, want)
	}
}

func TestBrowserSplitModeUsesHalfWidthPanes(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.transcriptMode = transcriptSplit

	listWidth := b.listPaneWidth()
	viewerWidth := b.viewerWidth()
	diff := listWidth - viewerWidth
	if diff < 0 {
		diff = -diff
	}

	if got, want := listWidth+viewerWidth+1, b.width; got != want {
		t.Fatalf("combined split width = %d, want %d", got, want)
	}
	if diff > 1 {
		t.Fatalf("pane width difference = %d, want <= 1 (list=%d viewer=%d)", diff, listWidth, viewerWidth)
	}
}

func TestBrowserSplitModeKeepsMinimumListWidthOnNarrowTerminal(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.width = 50
	b.transcriptMode = transcriptSplit
	b.updateLayout()

	if got, want := b.listPaneWidth(), 32; got != want {
		t.Fatalf("listPaneWidth = %d, want %d", got, want)
	}
	if got, want := b.viewerWidth(), 17; got != want {
		t.Fatalf("viewerWidth = %d, want %d", got, want)
	}
	if got, want := b.list.Width(), 30; got != want {
		t.Fatalf("list width = %d, want %d after applying split guardrail", got, want)
	}
}

func TestBrowserOpenViewerMsgSetsViewerState(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	session := testSession(testConversationIDPrimary)
	b.transcriptMode = transcriptSplit
	b.loadingConversationID = session.meta.id

	b, _ = b.Update(openViewerMsg{
		conversationID: session.meta.id,
		conversation:   singleSessionConversation(session.meta),
		session:        session,
	})

	if b.openConversationID != session.meta.id {
		t.Fatalf("openConversationID = %q, want %q", b.openConversationID, session.meta.id)
	}
	if b.loadingConversationID != "" {
		t.Fatalf("loadingConversationID = %q, want empty", b.loadingConversationID)
	}
	if b.viewer.session.meta.id != session.meta.id {
		t.Fatalf("viewer session id = %q, want %q", b.viewer.session.meta.id, session.meta.id)
	}
	if _, ok := b.transcriptCache[session.meta.id]; !ok {
		t.Fatal("expected transcript cache to contain opened session")
	}
}

func TestBrowserOpenViewerMsgIgnoresStaleLoad(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conv1 := testConv(testConversationIDPrimary)
	conv2 := testConv(testConversationIDSecondary)
	b.list.SetItems([]list.Item{conv1, conv2})
	b.list.Select(1)
	b.transcriptMode = transcriptSplit
	b.loadingConversationID = conv2.id()

	b, _ = b.Update(openViewerMsg{
		conversationID: conv1.id(),
		conversation:   conv1,
		session:        testSession(conv1.id()),
	})

	if b.openConversationID != "" {
		t.Fatalf("openConversationID = %q, want empty", b.openConversationID)
	}
	if b.loadingConversationID != conv2.id() {
		t.Fatalf("loadingConversationID = %q, want %q", b.loadingConversationID, conv2.id())
	}
	if b.viewer.session.meta.id != "" {
		t.Fatalf("viewer session id = %q, want empty", b.viewer.session.meta.id)
	}
}

func TestBrowserOKeyTogglesTranscriptFullscreen(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	session := testSession(testConversationIDPrimary)
	b.transcriptMode = transcriptSplit
	b.focus = focusList
	b.loadingConversationID = session.meta.id
	b, _ = b.Update(openViewerMsg{
		conversationID: session.meta.id,
		conversation:   singleSessionConversation(session.meta),
		session:        session,
	})

	b, _ = b.Update(tea.KeyPressMsg{Text: "O"})
	if b.transcriptMode != transcriptFullscreen {
		t.Fatalf("transcriptMode = %v, want %v", b.transcriptMode, transcriptFullscreen)
	}
	if b.openConversationID != session.meta.id {
		t.Fatalf("openConversationID = %q, want %q", b.openConversationID, session.meta.id)
	}

	b, _ = b.Update(tea.KeyPressMsg{Text: "O"})
	if b.transcriptMode != transcriptSplit {
		t.Fatalf("transcriptMode = %v, want %v", b.transcriptMode, transcriptSplit)
	}
	if b.openConversationID != session.meta.id {
		t.Fatalf("openConversationID = %q, want %q", b.openConversationID, session.meta.id)
	}
}

func TestBrowserQClosesTranscriptBeforeQuit(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.transcriptMode = transcriptSplit
	b.loadingConversationID = testConversationIDPrimary
	b, _ = b.Update(openViewerMsg{
		conversationID: testConversationIDPrimary,
		conversation:   testConv(testConversationIDPrimary),
		session:        testSession(testConversationIDPrimary),
	})

	b, cmd := b.Update(tea.KeyPressMsg{Text: "q"})
	if b.transcriptMode != transcriptClosed {
		t.Fatalf("transcriptMode = %v, want %v", b.transcriptMode, transcriptClosed)
	}
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("expected close transcript, not quit")
		}
	}

	_, cmd = b.Update(tea.KeyPressMsg{Text: "q"})
	if cmd == nil {
		t.Fatal("expected quit command from list-only view")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("cmd() = %T, want tea.QuitMsg", cmd())
	}
}

func TestBrowserSplitListFocusUpdatesTranscriptSelection(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conv1 := testConv(testConversationIDPrimary)
	conv2 := testConv(testConversationIDSecondary)
	b.list.SetItems([]list.Item{conv1, conv2})
	b.list.Select(0)
	b.transcriptMode = transcriptSplit
	b.focus = focusList
	b.loadingConversationID = testConversationIDPrimary
	b, _ = b.Update(openViewerMsg{
		conversationID: testConversationIDPrimary,
		conversation:   conv1,
		session:        testSession(testConversationIDPrimary),
	})

	b, cmd := b.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	if b.list.Index() != 1 {
		t.Fatalf("list index = %d, want 1", b.list.Index())
	}
	if b.loadingConversationID != conv2.id() {
		t.Fatalf("loadingConversationID = %q, want %q", b.loadingConversationID, conv2.id())
	}
	if cmd == nil {
		t.Fatal("expected transcript reload command after selection change")
	}
}

func TestBrowserTranscriptFocusDoesNotMoveList(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conv1 := testConv(testConversationIDPrimary)
	conv2 := testConv(testConversationIDSecondary)
	b.list.SetItems([]list.Item{conv1, conv2})
	b.transcriptMode = transcriptSplit
	b.focus = focusTranscript
	b.loadingConversationID = testConversationIDPrimary
	b, _ = b.Update(openViewerMsg{
		conversationID: testConversationIDPrimary,
		conversation:   testConv(testConversationIDPrimary),
		session:        testBrowserSessionLong(testConversationIDPrimary, "KEYWORD"),
	})

	indexBefore := b.list.Index()
	yBefore := b.viewer.viewport.YOffset()

	b, _ = b.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	if b.list.Index() != indexBefore {
		t.Fatalf("list index = %d, want %d", b.list.Index(), indexBefore)
	}
	if b.viewer.viewport.YOffset() <= yBefore {
		t.Fatalf("viewer Y offset = %d, want > %d", b.viewer.viewport.YOffset(), yBefore)
	}
}

func TestBrowserFooterShowsTranscriptTogglePrefixesConsistently(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.transcriptMode = transcriptSplit
	b.focus = focusTranscript
	b.loadingConversationID = testConversationIDPrimary
	b, _ = b.Update(openViewerMsg{
		conversationID: testConversationIDPrimary,
		conversation:   testConv(testConversationIDPrimary),
		session:        testSession(testConversationIDPrimary),
	})

	lines := strings.Split(b.footerView(), "\n")
	if len(lines) != 2 {
		t.Fatalf("footer line count = %d, want 2", len(lines))
	}

	helpLine := ansi.Strip(lines[0])
	if !strings.Contains(helpLine, "-t") {
		t.Fatalf("help line = %q, want -t", helpLine)
	}
	if !strings.Contains(helpLine, "-T") {
		t.Fatalf("help line = %q, want -T", helpLine)
	}
	if !strings.Contains(helpLine, "-R") {
		t.Fatalf("help line = %q, want -R", helpLine)
	}
	if !strings.Contains(helpLine, "+s") {
		t.Fatalf("help line = %q, want +s", helpLine)
	}
	if !strings.Contains(helpLine, "? help") {
		t.Fatalf("help line = %q, want help binding", helpLine)
	}
	if !strings.Contains(helpLine, "thinking") {
		t.Fatalf("help line = %q, want shared transcript terminology", helpLine)
	}
	if !strings.Contains(helpLine, "editor") {
		t.Fatalf("help line = %q, want shared editor terminology", helpLine)
	}
}

func TestBrowserListFooterOmitsCopyAndExport(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	footer := ansi.Strip(b.footerView())

	if strings.Contains(footer, " copy") {
		t.Fatalf("footer = %q, did not expect list copy action", footer)
	}
	if strings.Contains(footer, " export") {
		t.Fatalf("footer = %q, did not expect list export action", footer)
	}
}

func TestBrowserListHighlightsSearchPreview(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conv := testConv(testConversationIDPrimary)
	conv.searchPreview = archiveMatchesSourceSubtitle
	b.list.SetItems([]list.Item{conv})
	b.list.SetFilterText("archive")

	view := b.list.View()
	if !strings.Contains(ansi.Strip(view), archiveMatchesSourceSubtitle) {
		t.Fatalf("list view missing stripped search preview:\n%s", view)
	}
	if strings.Contains(view, archiveMatchesSourceSubtitle) {
		t.Fatalf("expected highlighted preview to be split by ANSI escapes:\n%s", view)
	}
}

func TestBrowserListHighlightsDeepSearchPreviewWithoutListFiltering(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conv := testConv(testConversationIDPrimary)
	conv.searchPreview = archiveMatchesSourceSubtitle

	items := buildDeepSearchItems("archive", []conversation{conv})
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, item)
	}
	b.list.SetItems(listItems)

	view := b.list.View()
	if !strings.Contains(ansi.Strip(view), archiveMatchesSourceSubtitle) {
		t.Fatalf("list view missing stripped deep-search preview:\n%s", view)
	}
	if strings.Contains(view, archiveMatchesSourceSubtitle) {
		t.Fatalf("expected deep-search preview to be split by ANSI escapes:\n%s", view)
	}
}

func TestBrowserListHelpOmitsCopyAndExport(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	sections := b.helpSections()

	for _, section := range sections {
		for _, item := range section.items {
			if item.desc == "copy transcript" {
				t.Fatalf("unexpected copy action in list help: %+v", item)
			}
			if item.desc == "export markdown" {
				t.Fatalf("unexpected export action in list help: %+v", item)
			}
		}
	}
}

func TestBrowserListFocusIgnoresCopyAndExport(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conv := testConv(testConversationIDPrimary)
	b.list.SetItems([]list.Item{conv})
	b.list.Select(0)

	cases := []struct {
		name string
		msg  tea.KeyPressMsg
	}{
		{name: "copy", msg: tea.KeyPressMsg{Text: "y"}},
		{name: "export", msg: tea.KeyPressMsg{Text: "e"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := b
			after, cmd := b.Update(tc.msg)

			if cmd != nil {
				t.Fatalf("cmd != nil for %s in list focus", tc.name)
			}
			if after.transcriptMode != before.transcriptMode {
				t.Fatalf("transcriptMode = %v, want %v", after.transcriptMode, before.transcriptMode)
			}
			if after.notification.text != "" {
				t.Fatalf("notification = %q, want empty", after.notification.text)
			}
		})
	}
}

func TestBrowserTranscriptFocusAllowsCopyAndExport(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.transcriptMode = transcriptSplit
	b.focus = focusTranscript
	b.loadingConversationID = testConversationIDPrimary
	b, _ = b.Update(openViewerMsg{
		conversationID: testConversationIDPrimary,
		conversation:   testConv(testConversationIDPrimary),
		session:        testSession(testConversationIDPrimary),
	})

	cases := []struct {
		name string
		msg  tea.KeyPressMsg
	}{
		{name: "copy", msg: tea.KeyPressMsg{Text: "y"}},
		{name: "export", msg: tea.KeyPressMsg{Text: "e"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, cmd := b.Update(tc.msg)
			if cmd == nil {
				t.Fatalf("expected %s command in transcript focus", tc.name)
			}
		})
	}
}

func TestBrowserSplitViewKeepsWindowHeightWithLongListItems(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.width = 90
	b.height = 24
	b.transcriptMode = transcriptSplit
	b.focus = focusList
	b.list.SetItems([]list.Item{
		testLongConv(testConversationIDPrimary),
		testLongConv(testConversationIDSecondary),
	})
	b.list.Select(0)
	b.loadingConversationID = testConversationIDPrimary
	b, _ = b.Update(openViewerMsg{
		conversationID: testConversationIDPrimary,
		conversation:   testConv(testConversationIDPrimary),
		session:        testBrowserSessionLong(testConversationIDPrimary, "KEYWORD"),
	})
	b.updateLayout()

	view := b.View()
	if got := lipgloss.Height(view); got != b.height {
		t.Fatalf("view height = %d, want %d", got, b.height)
	}
	if !strings.Contains(view, "╰") {
		t.Fatalf("expected split view to keep bottom frame visible, got: %s", view)
	}
}

func TestBrowserCloseTranscriptRestoresFullWidthList(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conv := testConv(testConversationIDPrimary)
	b.list.SetItems([]list.Item{conv})
	b.list.Select(0)

	b, _ = b.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	b.loadingConversationID = testConversationIDPrimary
	b, _ = b.Update(openViewerMsg{
		conversationID: testConversationIDPrimary,
		conversation:   testConv(testConversationIDPrimary),
		session:        testSession(testConversationIDPrimary),
	})
	b, _ = b.Update(tea.KeyPressMsg{Text: "q"})

	if got, want := b.list.Width(), b.width-2; got != want {
		t.Fatalf("list width = %d, want %d after closing transcript", got, want)
	}
}
