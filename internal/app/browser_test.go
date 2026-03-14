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
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testConversationIDPrimary   = "id-1"
	testConversationIDSecondary = "id-2"
)

func testBrowser(t *testing.T) browserModel {
	t.Helper()
	b := newBrowserModel(context.Background(), t.TempDir(), "dark", "2006-01-02 15:04", 20, 200)
	b.width = 120
	b.height = 40
	b = b.updateLayout()
	return b
}

func testSession(id string) conv.Session {
	return conv.Session{
		Meta: conv.SessionMeta{
			ID:        id,
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "hello"},
			{Role: conv.RoleAssistant, Text: "hi there"},
		},
	}
}

func testBrowserSessionLong(id string, keyword string) conv.Session {
	msgs := make([]conv.Message, 0, 40)
	for i := range 20 {
		msgs = append(msgs, conv.Message{Role: conv.RoleUser, Text: "user message"})
		text := "assistant response"
		if i == 15 {
			text = "assistant response with " + keyword
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

func testConv(id string) conv.Conversation {
	return conv.Conversation{
		Ref:     conv.Ref{Provider: conv.ProviderClaude, ID: id},
		Name:    "test-slug",
		Project: conv.Project{DisplayName: "test"},
		Sessions: []conv.SessionMeta{
			{
				ID:        id,
				Slug:      "test-slug",
				Timestamp: time.Now(),
				Project:   conv.Project{DisplayName: "test"},
			},
		},
	}
}

func testLongConv(id string) conv.Conversation {
	return conv.Conversation{
		Ref:  conv.Ref{Provider: conv.ProviderClaude, ID: id},
		Name: "very-long-conversation-name-that-should-wrap-in-the-split-pane-because-the-list-is-narrow",
		Project: conv.Project{
			DisplayName: "Projects/claude-search/with-a-very-long-project-name",
		},
		Sessions: []conv.SessionMeta{
			{
				ID:           id,
				Slug:         "very-long-conversation-name-that-should-wrap-in-the-split-pane-because-the-list-is-narrow",
				Timestamp:    time.Now(),
				Project:      conv.Project{DisplayName: "Projects/claude-search/with-a-very-long-project-name"},
				FirstMessage: strings.Repeat("This is a long first message that wraps. ", 4),
				Model:        "claude-opus-4-1",
				Version:      "1.2.3",
				MessageCount: 20,
			},
		},
	}
}

func helpItemKeys(items []helpItem) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.key)
	}
	return keys
}

func TestBrowserEnterOpensTranscriptSplit(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conversation := testConv(testConversationIDPrimary)
	b.list.SetItems([]list.Item{conversation})
	b.list.Select(0)

	b, cmd := b.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	assert.Equal(t, transcriptSplit, b.transcriptMode)
	assert.Equal(t, conversation.CacheKey(), b.loadingConversationID)
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
	b = b.updateLayout()

	assert.Equal(t, 32, b.listPaneWidth())
	assert.Equal(t, 17, b.viewerWidth())
	assert.Equal(t, 30, b.list.Width())
}

func TestBrowserOpenViewerMsgSetsViewerState(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	session := testSession(testConversationIDPrimary)
	b.transcriptMode = transcriptSplit
	b.loadingConversationID = session.Meta.ID

	b, _ = b.Update(openViewerMsg{
		conversationID: session.Meta.ID,
		conversation:   singleSessionConversation(session.Meta),
		session:        session,
	})

	assert.Equal(t, session.Meta.ID, b.openConversationID)
	assert.Empty(t, b.loadingConversationID)
	assert.Equal(t, session.Meta.ID, b.viewer.session.Meta.ID)
	_, ok := b.transcriptCache[session.Meta.ID]
	assert.True(t, ok)
}

func TestBrowserOpenViewerMsgIgnoresStaleLoad(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conversationA := testConv(testConversationIDPrimary)
	conversationB := testConv(testConversationIDSecondary)
	b.list.SetItems([]list.Item{conversationA, conversationB})
	b.list.Select(1)
	b.transcriptMode = transcriptSplit
	b.loadingConversationID = conversationB.CacheKey()

	b, _ = b.Update(openViewerMsg{
		conversationID: conversationA.CacheKey(),
		conversation:   conversationA,
		session:        testSession(conversationA.ID()),
	})

	assert.Empty(t, b.openConversationID)
	assert.Equal(t, conversationB.CacheKey(), b.loadingConversationID)
	assert.Empty(t, b.viewer.session.Meta.ID)
}

func TestBrowserOKeyTogglesTranscriptFullscreen(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	session := testSession(testConversationIDPrimary)
	b.transcriptMode = transcriptSplit
	b.focus = focusList
	b.loadingConversationID = session.Meta.ID
	b, _ = b.Update(openViewerMsg{
		conversationID: session.Meta.ID,
		conversation:   singleSessionConversation(session.Meta),
		session:        session,
	})

	b, _ = b.Update(tea.KeyPressMsg{Text: "O"})
	assert.Equal(t, transcriptFullscreen, b.transcriptMode)
	assert.Equal(t, session.Meta.ID, b.openConversationID)

	b, _ = b.Update(tea.KeyPressMsg{Text: "O"})
	assert.Equal(t, transcriptSplit, b.transcriptMode)
	assert.Equal(t, session.Meta.ID, b.openConversationID)
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
	conversationA := testConv(testConversationIDPrimary)
	conversationB := testConv(testConversationIDSecondary)
	b.list.SetItems([]list.Item{conversationA, conversationB})
	b.list.Select(0)
	b.transcriptMode = transcriptSplit
	b.focus = focusList
	b.loadingConversationID = testConversationIDPrimary
	b, _ = b.Update(openViewerMsg{
		conversationID: testConversationIDPrimary,
		conversation:   conversationA,
		session:        testSession(testConversationIDPrimary),
	})

	b, cmd := b.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	assert.Equal(t, 1, b.list.Index())
	assert.Equal(t, conversationB.CacheKey(), b.loadingConversationID)
	require.NotNil(t, cmd)
}

func TestBrowserTranscriptFocusDoesNotMoveList(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conversationA := testConv(testConversationIDPrimary)
	conversationB := testConv(testConversationIDSecondary)
	b.list.SetItems([]list.Item{conversationA, conversationB})
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

	helpLine := ansi.Strip(renderHelpItems(b.viewer.footerItems()))
	assertContainsAll(t, helpLine, "-t", "-T", "-R", "+s", "? help", "thinking", "open")
}

func TestBrowserListFooterShowsDeepSearchAsToggle(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	items := b.listFooterItems()

	var found bool
	for _, item := range items {
		if item.key != "ctrl+s" {
			continue
		}
		found = true
		assert.Equal(t, "deep search", item.desc)
		assert.True(t, item.toggle)
		assert.False(t, item.on)
	}

	assert.True(t, found)
}

func TestRenderHelpItemUsesGlowForPurpleToggleHighlight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		item     helpItem
		expected string
	}{
		{
			name:     "active toggle without glow stays accent",
			item:     helpItem{key: "ctrl+s", desc: "deep search", toggle: true, on: true},
			expected: lipgloss.NewStyle().Foreground(colorAccent).Render("+ctrl+s"),
		},
		{
			name:     "inactive toggle without glow stays accent",
			item:     helpItem{key: "ctrl+s", desc: "deep search", toggle: true},
			expected: lipgloss.NewStyle().Foreground(colorAccent).Render("-ctrl+s"),
		},
		{
			name:     "glowing toggle uses primary",
			item:     helpItem{key: "t", desc: "thinking", toggle: true, glow: true},
			expected: lipgloss.NewStyle().Foreground(colorPrimary).Render("-t"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, renderHelpItem(tt.item), tt.expected)
		})
	}
}

func TestBrowserListFooterOrdersItemsByWorkflow(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)

	assert.Equal(
		t,
		[]string{"j/k", "gg", "G", "ctrl+f/b", "/", "ctrl+s", "enter", "o", "r", "R", "?", "q"},
		helpItemKeys(b.listFooterItems()),
	)
}

func TestBrowserSearchFooterShowsDeepSearchStateWhileEditing(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.search.editing = true
	b.search.mode = searchModeDeep
	b.search.status = searchStatusSearching
	b.searchInput.Focus()
	b.searchInput.SetValue("hello")
	b.notification = infoNotification("search ready").notification

	lines := strings.Split(b.footerView(), "\n")
	require.Len(t, lines, 2)

	searchLine := ansi.Strip(lines[0])
	statusLine := ansi.Strip(lines[1])

	assert.Contains(t, searchLine, "/hello")
	assert.Contains(t, searchLine, "+ctrl+s")
	assert.Contains(t, searchLine, "deep search")
	assert.Contains(t, searchLine, "[UPDATING]")
	assert.Contains(t, statusLine, "search ready")
}

func TestBrowserSplitListFooterUsesConsistentActionLabels(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.transcriptMode = transcriptSplit
	items := b.listFooterItems()

	assert.Equal(
		t,
		[]string{"j/k", "gg", "G", "ctrl+f/b", "/", "ctrl+s", "enter", "o", "r", "R", "tab", "O", "?", "q/esc"},
		helpItemKeys(items),
	)

	var sawFocus bool
	var sawLayout bool
	for _, item := range items {
		if item.key == "tab" && item.desc == "focus transcript" {
			sawFocus = true
		}
		if item.key == "O" && item.desc == "fullscreen transcript" {
			sawLayout = true
		}
	}

	assert.True(t, sawFocus)
	assert.True(t, sawLayout)
}

func TestBrowserSplitTranscriptFooterUsesConsistentActionLabels(t *testing.T) {
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

	assert.Equal(
		t,
		[]string{"/", "n/N", "t", "T", "R", "s", "m", "y", "o", "e", "tab", "O", "?", "q/esc"},
		helpItemKeys(b.transcriptFooterItems()),
	)

	items := b.transcriptActionItems()
	require.Len(t, items, 2)
	assert.Equal(t, helpItem{key: "tab", desc: "focus list"}, items[0])
	assert.Equal(t, helpItem{key: "O", desc: "fullscreen transcript"}, items[1])
}

func TestBrowserListFooterStatusDoesNotChangeWithSelectedProject(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.mainConversationCount = 2

	conversationA := testConv(testConversationIDPrimary)
	conversationA.Project.DisplayName = "alpha/project"
	conversationA.Sessions[0].Project.DisplayName = "alpha/project"

	conversationB := testConv(testConversationIDSecondary)
	conversationB.Project.DisplayName = "beta/project"
	conversationB.Sessions[0].Project.DisplayName = "beta/project"

	b.list.SetItems([]list.Item{conversationA, conversationB})
	b.list.Select(0)
	first := strings.Join(b.listFooterStatusParts(), "  ")

	b.list.Select(1)
	second := strings.Join(b.listFooterStatusParts(), "  ")

	assert.Equal(t, first, second)
	assert.NotContains(t, first, "alpha/project")
	assert.NotContains(t, second, "beta/project")
}

func TestBrowserTranscriptCopyOverlayFooterOmitsLayoutItems(t *testing.T) {
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

	b, cmd := b.Update(tea.KeyPressMsg{Text: "y"})

	assert.Nil(t, cmd)
	assert.Equal(t, []string{"c", "r", "?", "q/esc"}, helpItemKeys(b.transcriptFooterItems()))
}

func TestBrowserTranscriptHelpKeepsCopyOverlayActive(t *testing.T) {
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

	b, _ = b.Update(tea.KeyPressMsg{Text: "y"})
	require.Equal(t, viewerActionCopy, b.viewer.actionMode)

	b, cmd := b.Update(tea.KeyPressMsg{Text: "?"})

	assert.Nil(t, cmd)
	assert.True(t, b.helpOpen)
	assert.Equal(t, viewerActionCopy, b.viewer.actionMode)
	require.Len(t, b.helpSections(), 1)
	assert.Equal(t, "Select Target", b.helpSections()[0].title)

	b, _ = b.Update(tea.KeyPressMsg{Text: "?"})

	assert.False(t, b.helpOpen)
	assert.Equal(t, viewerActionCopy, b.viewer.actionMode)
}

func TestBrowserTranscriptHelpKeepsPlanPickerActive(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.transcriptMode = transcriptSplit
	b.focus = focusTranscript
	b.loadingConversationID = testConversationIDPrimary
	b, _ = b.Update(openViewerMsg{
		conversationID: testConversationIDPrimary,
		conversation:   testConv(testConversationIDPrimary),
		session:        testSessionWithPlans(testConversationIDPrimary, 2),
	})

	b, _ = b.Update(tea.KeyPressMsg{Text: "y"})
	b, _ = b.Update(tea.KeyPressMsg{Text: "p"})
	require.True(t, b.viewer.planPicker.active)

	b, cmd := b.Update(tea.KeyPressMsg{Text: "?"})

	assert.Nil(t, cmd)
	assert.True(t, b.helpOpen)
	assert.True(t, b.viewer.planPicker.active)
	require.Len(t, b.helpSections(), 1)
	assert.Equal(t, "Select Plan", b.helpSections()[0].title)

	b, _ = b.Update(tea.KeyPressMsg{Text: "?"})

	assert.False(t, b.helpOpen)
	assert.True(t, b.viewer.planPicker.active)
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
	conversation := testConv(testConversationIDPrimary)
	conversation.SearchPreview = archiveMatchesSourceSubtitle
	b.list.SetItems([]list.Item{conversation})
	b.list.SetFilterText("archive")

	view := b.list.View()
	assert.Contains(t, ansi.Strip(view), archiveMatchesSourceSubtitle)
	assert.NotContains(t, view, archiveMatchesSourceSubtitle)
}

func TestBrowserListHighlightsDeepSearchPreviewWithoutListFiltering(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conversation := testConv(testConversationIDPrimary)
	conversation.SearchPreview = archiveMatchesSourceSubtitle

	items := buildDeepSearchItems("archive", []conv.Conversation{conversation})
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, item)
	}
	b.list.SetItems(listItems)

	view := b.list.View()
	stripped := ansi.Strip(view)
	assert.Contains(t, stripped, archiveMatchesSourceSubtitle)
	assert.NotEqual(t, stripped, view)
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

func TestBrowserListHelpShowsDeepSearchInTogglesSection(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	sections := b.helpSections()

	var sawToggles bool
	for _, section := range sections {
		if section.title != "Toggles" {
			for _, item := range section.items {
				assert.NotEqual(t, "toggle deep scope", item.desc)
			}
			continue
		}

		sawToggles = true
		require.Len(t, section.items, 1)
		assert.Equal(t, "ctrl+s", section.items[0].key)
		assert.Equal(t, "deep search", section.items[0].desc)
		assert.True(t, section.items[0].toggle)
	}

	assert.True(t, sawToggles)
}

func TestBrowserListFocusIgnoresCopyAndExport(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conversation := testConv(testConversationIDPrimary)
	b.list.SetItems([]list.Item{conversation})
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

func TestBrowserTranscriptFocusAllowsActionPrefixAndExport(t *testing.T) {
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
		name    string
		msg     tea.KeyPressMsg
		wantCmd bool
	}{
		{name: "copy prefix", msg: tea.KeyPressMsg{Text: "y"}, wantCmd: false},
		{name: "export", msg: tea.KeyPressMsg{Text: "e"}, wantCmd: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			after, cmd := b.Update(tc.msg)
			if tc.wantCmd {
				require.NotNil(t, cmd)
				return
			}
			assert.Nil(t, cmd)
			assert.Equal(t, viewerActionCopy, after.viewer.actionMode)
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
	b = b.updateLayout()

	view := b.View()
	assert.Equal(t, b.height, lipgloss.Height(view))
	assert.Contains(t, view, "╰")
}

func TestBrowserCloseTranscriptRestoresFullWidthList(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	conversation := testConv(testConversationIDPrimary)
	b.list.SetItems([]list.Item{conversation})
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
