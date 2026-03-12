package app

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMetadataSearchItemsUsesFuzzyMatches(t *testing.T) {
	t.Parallel()

	conversations := []conv.Conversation{
		testConv("one"),
		{
			Name:    "archiver",
			Project: conv.Project{DisplayName: "test"},
			Sessions: []conv.SessionMeta{
				{
					ID:        "two",
					Slug:      "archiver",
					Timestamp: testConv("two").Sessions[0].Timestamp,
					Project:   conv.Project{DisplayName: "test"},
				},
			},
		},
	}

	items := buildMetadataSearchItems("arv", conversations)
	require.Len(t, items, 1)
	assert.Equal(t, "two", items[0].conversation.ID())
	assert.NotEmpty(t, items[0].matchRanges.title)
}

func TestBuildMetadataSearchItemsIgnoresDeepSearchPreviewText(t *testing.T) {
	t.Parallel()

	conversation := testConv("one")
	conversation.SearchPreview = "needle only in preview"

	items := buildMetadataSearchItems("needle", []conv.Conversation{conversation})
	assert.Empty(t, items)
}

func TestBuildDeepSearchItemsHighlightsDescriptionMatches(t *testing.T) {
	t.Parallel()

	conversation := testConv("one")
	conversation.SearchPreview = "found the archive needle here"

	items := buildDeepSearchItems("archive", []conv.Conversation{conversation})
	require.Len(t, items, 1)
	assert.NotEmpty(t, items[0].matchRanges.desc)
}

func TestBuildDeepSearchItemsNoMatchWhenQueryAbsent(t *testing.T) {
	t.Parallel()

	conversation := testConv("one")
	conversation.SearchPreview = archiveMatchesSourceSubtitle

	items := buildDeepSearchItems("", []conv.Conversation{conversation})
	require.Len(t, items, 1)
	assert.Empty(t, items[0].matchRanges.desc)
}

func TestBrowserSearchBindingUsesSlash(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b, cmd := b.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	require.NotNil(t, cmd)
	assert.True(t, b.search.editing)
	assert.True(t, b.searchInput.Focused())
}

func TestBrowserCanToggleDeepSearchWhileEditingQuery(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.search.baseConversations = []conv.Conversation{testNamedConversation("one", "one")}
	b.deepSearchAvailable = true
	var cmds []tea.Cmd
	b.applyFullConversationList(&cmds)
	b.search.editing = true
	b.searchInput.Focus()

	b, _ = b.handleSearchKey(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, &cmds)
	assert.Equal(t, searchModeDeep, b.search.mode)
}

func TestBrowserToggleDeepSearchWhileEditingQuerySyncsTranscriptSelection(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha")
	beta := testNamedConversation("beta-id", "beta")

	b := testBrowser(t)
	b.search.baseConversations = []conv.Conversation{alpha, beta}
	b.mainConversationCount = 2
	b.transcriptMode = transcriptSplit
	b.focus = focusList
	b.loadingConversationID = alpha.CacheKey()
	b, _ = b.Update(openViewerMsg{
		conversationID: alpha.CacheKey(),
		conversation:   alpha,
		session:        testSession(alpha.ID()),
	})

	var cmds []tea.Cmd
	b.search.mode = searchModeDeep
	b.search.query = "beta"
	b.search.editing = true
	b.searchInput.Focus()
	b.searchInput.SetValue("beta")
	b.setSearchItems(buildDeepSearchItems("beta", []conv.Conversation{alpha}), &cmds)

	b, _ = b.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})

	assert.Equal(t, searchModeMetadata, b.search.mode)
	assert.Equal(t, beta.CacheKey(), b.loadingConversationID)
}

func TestBrowserToggleDeepSearchShowsScopeNotification(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.search.baseConversations = []conv.Conversation{testNamedConversation("one", "one")}
	b.deepSearchAvailable = true
	var cmds []tea.Cmd
	b.applyFullConversationList(&cmds)

	b, _ = b.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	assert.Equal(t, "search scope: deep search", b.notification.text)

	b, _ = b.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	assert.Equal(t, "search scope: metadata search", b.notification.text)
}

func TestBrowserDeepSearchRefreshesWhenQueryChanges(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-session")
	beta := testNamedConversation("beta-id", "beta-session")

	store := &fakeBrowserStore{
		deepSearchResults: map[string][]conv.Conversation{
			"alpha": {alpha},
			"beta":  {beta},
		},
		deepSearchAvailable: true,
	}

	b := testBrowser(t)
	b.store = store
	b.search.baseConversations = []conv.Conversation{alpha, beta}
	b.deepSearchAvailable = true
	b.mainConversationCount = 2

	var cmds []tea.Cmd
	b.applyFullConversationList(&cmds)
	b.toggleSearchMode(&cmds)

	cmds = nil
	b.setSearchQuery("alpha", &cmds)
	assert.Equal(t, searchStatusDebouncing, b.search.status)

	b, cmd := b.Update(deepSearchDebounceMsg{revision: b.search.revision, query: b.search.query})
	require.NotNil(t, cmd)
	b, _ = b.Update(requireMsgType[deepSearchResultMsg](t, cmd()))

	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, alpha.ID(), b.search.visibleConversations[0].ID())

	cmds = nil
	b.setSearchQuery("beta", &cmds)
	b, cmd = b.Update(deepSearchDebounceMsg{revision: b.search.revision, query: b.search.query})
	require.NotNil(t, cmd)
	b, _ = b.Update(requireMsgType[deepSearchResultMsg](t, cmd()))

	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, beta.ID(), b.search.visibleConversations[0].ID())
}

func TestBrowserIgnoresStaleDeepSearchResults(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-session")
	beta := testNamedConversation("beta-id", "beta-session")

	b := testBrowser(t)
	b.search.baseConversations = []conv.Conversation{alpha, beta}
	b.mainConversationCount = 2
	var cmds []tea.Cmd
	b.applyFullConversationList(&cmds)

	b.search.mode = searchModeDeep
	b.search.query = "beta"
	b.search.revision = 3
	b.search.visibleConversations = []conv.Conversation{beta}
	b.setSearchItems(buildDeepSearchItems("beta", []conv.Conversation{beta}), &cmds)

	b, _ = b.Update(deepSearchResultMsg{
		revision:      2,
		query:         "alpha",
		conversations: []conv.Conversation{alpha},
	})

	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, beta.ID(), b.search.visibleConversations[0].ID())
}

func TestBrowserToggleSearchModeReappliesCurrentQuery(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-browser")
	beta := testNamedConversation("beta-id", "beta-browser")

	store := &fakeBrowserStore{
		deepSearchResults: map[string][]conv.Conversation{
			"beta-browser": {beta},
		},
		deepSearchAvailable: true,
	}

	b := testBrowser(t)
	b.store = store
	b.search.baseConversations = []conv.Conversation{alpha, beta}
	b.mainConversationCount = 2
	b.deepSearchAvailable = true

	var cmds []tea.Cmd
	b.setSearchQuery("beta-browser", &cmds)
	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, beta.ID(), b.search.visibleConversations[0].ID())

	cmds = nil
	b.toggleSearchMode(&cmds)
	require.NotEmpty(t, cmds)
	b, _ = b.Update(requireMsgType[deepSearchResultMsg](t, cmds[len(cmds)-1]()))
	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, beta.ID(), b.search.visibleConversations[0].ID())

	cmds = nil
	b.toggleSearchMode(&cmds)
	assert.Equal(t, searchModeMetadata, b.search.mode)
	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, beta.ID(), b.search.visibleConversations[0].ID())
}

func TestBrowserToggleDeepSearchWhenUnavailableShowsNotification(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.search.baseConversations = []conv.Conversation{testNamedConversation("one", "one")}
	var cmds []tea.Cmd
	b.applyFullConversationList(&cmds)
	b.deepSearchAvailable = false

	b.toggleSearchMode(&cmds)

	assert.Equal(t, searchModeMetadata, b.search.mode)
	assert.Equal(t, searchStatusIdle, b.search.status)
	assert.Equal(
		t,
		"deep search unavailable; re-import to rebuild the local index",
		b.notification.text,
	)
}

func testNamedConversation(id, slug string) conv.Conversation {
	return conv.Conversation{
		Name:    slug,
		Project: conv.Project{DisplayName: "test"},
		Sessions: []conv.SessionMeta{
			{
				ID:        id,
				Slug:      slug,
				Timestamp: time.Now(),
				Project:   conv.Project{DisplayName: "test"},
			},
		},
	}
}
