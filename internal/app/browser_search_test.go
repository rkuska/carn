package app

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMetadataSearchItemsUsesFuzzyMatches(t *testing.T) {
	t.Parallel()

	convs := []conversation{
		testConv("one"),
		{
			name:    "archiver",
			project: project{displayName: "test"},
			sessions: []sessionMeta{
				{id: "two", slug: "archiver", timestamp: testConv("two").sessions[0].timestamp},
			},
		},
	}

	items := buildMetadataSearchItems("arv", convs)
	require.Len(t, items, 1)
	assert.Equal(t, "two", items[0].conversation.id())
	assert.NotEmpty(t, items[0].matchRanges.title)
}

func TestBuildMetadataSearchItemsIgnoresDeepSearchPreviewText(t *testing.T) {
	t.Parallel()

	conv := testConv("one")
	conv.searchPreview = "needle only in preview"

	items := buildMetadataSearchItems("needle", []conversation{conv})
	assert.Empty(t, items)
}

func TestBuildDeepSearchItemsDoesNotHighlightPreviewMatches(t *testing.T) {
	t.Parallel()

	conv := testConv("one")
	conv.searchPreview = archiveMatchesSourceSubtitle

	items := buildDeepSearchItems([]conversation{conv})
	require.Len(t, items, 1)
	assert.Empty(t, items[0].matchRanges.title)
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
	b.search.baseConversations = []conversation{testNamedConversation("one", "one")}
	b.deepSearchAvailable = true
	var cmds []tea.Cmd
	b.applyFullConversationList(&cmds)
	b.search.editing = true
	b.searchInput.Focus()

	b, _ = b.handleSearchKey(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, &cmds)
	assert.Equal(t, searchModeDeep, b.search.mode)
}

func TestBrowserDeepSearchRefreshesWhenQueryChanges(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-session")
	beta := testNamedConversation("beta-id", "beta-session")

	b := testBrowser(t)
	b.sessionCache[alpha.id()] = testSearchSession(alpha.id(), "contains alpha needle")
	b.sessionCache[beta.id()] = testSearchSession(beta.id(), "contains beta needle")
	b.search.baseConversations = []conversation{alpha, beta}
	b.searchCorpus = searchCorpus{
		units: []searchUnit{
			{conversationID: alpha.cacheKey(), text: "contains alpha needle"},
			{conversationID: beta.cacheKey(), text: "contains beta needle"},
		},
	}
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
	assert.Equal(t, alpha.id(), b.search.visibleConversations[0].id())

	cmds = nil
	b.setSearchQuery("beta", &cmds)
	b, cmd = b.Update(deepSearchDebounceMsg{revision: b.search.revision, query: b.search.query})
	require.NotNil(t, cmd)
	b, _ = b.Update(requireMsgType[deepSearchResultMsg](t, cmd()))

	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, beta.id(), b.search.visibleConversations[0].id())
}

func TestBrowserIgnoresStaleDeepSearchResults(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-session")
	beta := testNamedConversation("beta-id", "beta-session")

	b := testBrowser(t)
	b.search.baseConversations = []conversation{alpha, beta}
	b.mainConversationCount = 2
	var cmds []tea.Cmd
	b.applyFullConversationList(&cmds)

	b.search.mode = searchModeDeep
	b.search.query = "beta"
	b.search.revision = 3
	b.search.visibleConversations = []conversation{beta}
	b.setSearchItems(buildDeepSearchItems([]conversation{beta}), &cmds)

	b, _ = b.Update(deepSearchResultMsg{
		revision:      2,
		query:         "alpha",
		conversations: []conversation{alpha},
	})

	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, beta.id(), b.search.visibleConversations[0].id())
}

func TestBrowserToggleSearchModeReappliesCurrentQuery(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-browser")
	beta := testNamedConversation("beta-id", "beta-browser")

	b := testBrowser(t)
	b.search.baseConversations = []conversation{alpha, beta}
	b.mainConversationCount = 2
	b.sessionCache[alpha.id()] = testSearchSession(alpha.id(), "contains alpha needle")
	b.sessionCache[beta.id()] = testSearchSession(beta.id(), "contains beta needle")
	b.searchCorpus = searchCorpus{
		units: []searchUnit{
			{conversationID: alpha.cacheKey(), text: "contains alpha needle"},
			{conversationID: beta.cacheKey(), text: "contains beta needle"},
		},
	}
	b.deepSearchAvailable = true

	var cmds []tea.Cmd
	b.setSearchQuery("beta-browser", &cmds)
	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, beta.id(), b.search.visibleConversations[0].id())

	b.toggleSearchMode(&cmds)
	require.NotEmpty(t, cmds)
	deepCmd := cmds[len(cmds)-1]
	b, _ = b.Update(requireMsgType[deepSearchResultMsg](t, deepCmd()))
	assert.Empty(t, b.search.visibleConversations)

	cmds = nil
	b.toggleSearchMode(&cmds)
	assert.Equal(t, searchModeMetadata, b.search.mode)
	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, beta.id(), b.search.visibleConversations[0].id())
}

func TestBrowserToggleDeepSearchWhenUnavailableShowsNotification(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.search.baseConversations = []conversation{testNamedConversation("one", "one")}
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

func testNamedConversation(id, slug string) conversation {
	return conversation{
		name:    slug,
		project: project{displayName: "test"},
		sessions: []sessionMeta{
			{id: id, slug: slug, timestamp: time.Now(), project: project{displayName: "test"}},
		},
	}
}

func testSearchSession(id, text string) sessionFull {
	return sessionFull{
		meta: sessionMeta{
			id:        id,
			timestamp: time.Now(),
			project:   project{displayName: "test"},
		},
		messages: []message{{role: roleAssistant, text: text}},
	}
}
