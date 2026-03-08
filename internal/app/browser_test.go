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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		project: project{displayName: "test"},
		sessions: []sessionMeta{
			{id: id, slug: "test-slug", timestamp: time.Now(), project: project{displayName: "test"}},
		},
	}
}

func testLongConv(id string) conversation {
	return conversation{
		name: "very-long-conversation-name-that-should-wrap-in-the-split-pane-because-the-list-is-narrow",
		project: project{
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

	assert.Equal(t, transcriptSplit, b.transcriptMode)
	assert.Equal(t, conv.id(), b.loadingConversationID)
	require.NotNil(t, cmd)
	assert.Equal(t, b.listPaneWidth()-2, b.list.Width())
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

	assert.Equal(t, b.width, listWidth+viewerWidth+1)
	assert.LessOrEqual(t, diff, 1)
}

func TestBrowserSplitModeKeepsMinimumListWidthOnNarrowTerminal(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.width = 50
	b.transcriptMode = transcriptSplit
	b.updateLayout()

	assert.Equal(t, 32, b.listPaneWidth())
	assert.Equal(t, 17, b.viewerWidth())
	assert.Equal(t, 30, b.list.Width())
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

	assert.Equal(t, session.meta.id, b.openConversationID)
	assert.Empty(t, b.loadingConversationID)
	assert.Equal(t, session.meta.id, b.viewer.session.meta.id)
	_, ok := b.transcriptCache[session.meta.id]
	assert.True(t, ok)
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

	assert.Empty(t, b.openConversationID)
	assert.Equal(t, conv2.id(), b.loadingConversationID)
	assert.Empty(t, b.viewer.session.meta.id)
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
	assert.Equal(t, transcriptFullscreen, b.transcriptMode)
	assert.Equal(t, session.meta.id, b.openConversationID)

	b, _ = b.Update(tea.KeyPressMsg{Text: "O"})
	assert.Equal(t, transcriptSplit, b.transcriptMode)
	assert.Equal(t, session.meta.id, b.openConversationID)
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
	assert.Equal(t, transcriptClosed, b.transcriptMode)
	if cmd != nil {
		_, ok := cmd().(tea.QuitMsg)
		assert.False(t, ok)
	}

	_, cmd = b.Update(tea.KeyPressMsg{Text: "q"})
	require.NotNil(t, cmd)
	requireMsgType[tea.QuitMsg](t, cmd())
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

	assert.Equal(t, 1, b.list.Index())
	assert.Equal(t, conv2.id(), b.loadingConversationID)
	require.NotNil(t, cmd)
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

	assert.Equal(t, indexBefore, b.list.Index())
	assert.Greater(t, b.viewer.viewport.YOffset(), yBefore)
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
	require.Len(t, lines, 2)

	helpLine := ansi.Strip(lines[0])
	assertContainsAll(t, helpLine, "-t", "-T", "-R", "+s", "? help", "thinking", "editor")
}

func TestBrowserListFooterOmitsCopyAndExport(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	footer := ansi.Strip(b.footerView())

	assertNotContainsAll(t, footer, " copy", " export")
}

func TestBrowserListHighlightsSearchPreview(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conv := testConv(testConversationIDPrimary)
	conv.searchPreview = archiveMatchesSourceSubtitle
	b.list.SetItems([]list.Item{conv})
	b.list.SetFilterText("archive")

	view := b.list.View()
	assert.Contains(t, ansi.Strip(view), archiveMatchesSourceSubtitle)
	assert.NotContains(t, view, archiveMatchesSourceSubtitle)
}

func TestBrowserListHighlightsDeepSearchPreviewWithoutListFiltering(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conv := testConv(testConversationIDPrimary)
	conv.searchPreview = archiveMatchesSourceSubtitle

	items := buildDeepSearchItems([]conversation{conv})
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, item)
	}
	b.list.SetItems(listItems)

	view := b.list.View()
	assert.Contains(t, ansi.Strip(view), archiveMatchesSourceSubtitle)
	assert.Contains(t, view, archiveMatchesSourceSubtitle)
}

func TestBrowserListHelpOmitsCopyAndExport(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	sections := b.helpSections()

	for _, section := range sections {
		for _, item := range section.items {
			assert.NotEqual(t, "copy transcript", item.desc)
			assert.NotEqual(t, "export markdown", item.desc)
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

			assert.Nil(t, cmd)
			assert.Equal(t, before.transcriptMode, after.transcriptMode)
			assert.Empty(t, after.notification.text)
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
			require.NotNil(t, cmd)
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
	assert.Equal(t, b.height, lipgloss.Height(view))
	assert.Contains(t, view, "╰")
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

	assert.Equal(t, b.width-2, b.list.Width())
}
