package app

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestConversationMetadataDescriptionIncludesProviderLabel(t *testing.T) {
	t.Parallel()

	conversation := testNamedConversation("codex-one", "import-codex")
	conversation.Ref.Provider = conv.ProviderCodex

	desc := conversationMetadataDescription(conversation)

	assert.Contains(t, desc, "Codex")
	assert.Contains(t, desc, "0 msgs")
}

func TestBuildPlainConversationItemsSeparatesMetadataAndPreview(t *testing.T) {
	t.Parallel()

	conversation := testConv("one")
	conversation.Sessions[0].FirstMessage = "first user prompt"

	items := buildPlainConversationItems([]conv.Conversation{conversation})
	require.Len(t, items, 1)
	assert.Contains(t, items[0].metadata, "Claude")
	assert.NotContains(t, items[0].metadata, "first user prompt")
	assert.Equal(t, "first user prompt", items[0].preview)
}

func TestBuildDeepSearchItemsHighlightsPreviewMatches(t *testing.T) {
	t.Parallel()

	conversation := testConv("one")
	conversation.SetSearchPreview("found the archive needle here")

	items := buildDeepSearchItems("archive", []conv.Conversation{conversation})
	require.Len(t, items, 1)
	assert.Contains(t, items[0].metadata, "Claude")
	assert.Equal(t, "found the archive needle here", items[0].preview)
	assert.Empty(t, items[0].matchRanges.metadata)
	assert.NotEmpty(t, items[0].matchRanges.preview)
}

func TestBuildDeepSearchItemsNoMatchWhenQueryAbsent(t *testing.T) {
	t.Parallel()

	conversation := testConv("one")
	conversation.SetSearchPreview(archiveMatchesSourceSubtitle)

	items := buildDeepSearchItems("", []conv.Conversation{conversation})
	require.Len(t, items, 1)
	assert.Empty(t, items[0].matchRanges.metadata)
	assert.Empty(t, items[0].matchRanges.preview)
}

func TestBuildDeepSearchItemsPrefersSearchPreviewOverFirstMessage(t *testing.T) {
	t.Parallel()

	conversation := testConv("one")
	conversation.Sessions[0].FirstMessage = "first message fallback"
	conversation.SetSearchPreview("actual matched preview")

	items := buildDeepSearchItems("matched", []conv.Conversation{conversation})
	require.Len(t, items, 1)
	assert.Equal(t, "actual matched preview", items[0].preview)
	assert.False(t, strings.Contains(items[0].preview, "fallback"))
}

func TestBrowserSearchBindingUsesSlash(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b, cmd := b.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	require.NotNil(t, cmd)
	assert.True(t, b.search.editing)
	assert.True(t, b.searchInput.Focused())
}

func TestBrowserClearSearchBindingWhileEditingClearsQuery(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-session")
	beta := testNamedConversation("beta-id", "beta-session")

	b := testBrowser(t)
	b.search.baseConversations = []conv.Conversation{alpha, beta}
	b.mainConversations = b.search.baseConversations

	var cmds []tea.Cmd
	b.search.query = testResyncBetaSlug
	b.search.visibleConversations = []conv.Conversation{beta}
	b.search.editing = true
	b.searchInput.Focus()
	b.searchInput.SetValue(testResyncBetaSlug)
	b = b.setSearchItems(buildDeepSearchItems(testResyncBetaSlug, []conv.Conversation{beta}), &cmds)

	b, _ = b.handleSearchKey(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}, &cmds)

	assert.False(t, b.search.editing)
	assert.Equal(t, "", b.search.query)
	assert.Equal(t, "", b.searchInput.Value())
	assert.Equal(t, searchStatusIdle, b.search.status)
	require.Len(t, b.search.visibleConversations, 2)
	assert.Equal(t, alpha.ID(), b.search.visibleConversations[0].ID())
	assert.Equal(t, beta.ID(), b.search.visibleConversations[1].ID())
}

func TestBrowserClearSearchBindingFromListClearsQuery(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-session")
	beta := testNamedConversation("beta-id", "beta-session")

	b := testBrowser(t)
	b.search.baseConversations = []conv.Conversation{alpha, beta}
	b.mainConversations = b.search.baseConversations

	var cmds []tea.Cmd
	b.search.query = testResyncBetaSlug
	b.search.visibleConversations = []conv.Conversation{beta}
	b = b.setSearchItems(buildDeepSearchItems(testResyncBetaSlug, []conv.Conversation{beta}), &cmds)

	b, _ = b.Update(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl})

	assert.Equal(t, "", b.search.query)
	assert.Equal(t, searchStatusIdle, b.search.status)
	require.Len(t, b.search.visibleConversations, 2)
	assert.Equal(t, alpha.ID(), b.search.visibleConversations[0].ID())
	assert.Equal(t, beta.ID(), b.search.visibleConversations[1].ID())
}

func TestBrowserSearchEscapeRestoresFullList(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-session")
	beta := testNamedConversation("beta-id", "beta-session")

	b := testBrowser(t)
	b.search.baseConversations = []conv.Conversation{alpha, beta}
	b.mainConversations = b.search.baseConversations

	var cmds []tea.Cmd
	b.search.query = testResyncBetaSlug
	b.search.visibleConversations = []conv.Conversation{beta}
	b.search.editing = true
	b.searchInput.Focus()
	b.searchInput.SetValue(testResyncBetaSlug)
	b = b.setSearchItems(buildDeepSearchItems(testResyncBetaSlug, []conv.Conversation{beta}), &cmds)

	b, _ = b.handleSearchKey(tea.KeyPressMsg{Code: tea.KeyEscape}, &cmds)

	assert.False(t, b.search.editing)
	assert.Equal(t, "", b.search.query)
	assert.Equal(t, "", b.searchInput.Value())
	require.Len(t, b.search.visibleConversations, 2)
	assert.Equal(t, alpha.ID(), b.search.visibleConversations[0].ID())
	assert.Equal(t, beta.ID(), b.search.visibleConversations[1].ID())
}

func TestBrowserSearchEnterKeepsQueryActive(t *testing.T) {
	t.Parallel()

	beta := testNamedConversation("beta-id", "beta-session")

	b := testBrowser(t)
	b.search.query = testResyncBetaSlug
	b.search.visibleConversations = []conv.Conversation{beta}
	b.search.editing = true
	b.searchInput.Focus()
	b.searchInput.SetValue(testResyncBetaSlug)

	var cmds []tea.Cmd
	b, _ = b.handleSearchKey(tea.KeyPressMsg{Code: tea.KeyEnter}, &cmds)

	assert.False(t, b.search.editing)
	assert.Equal(t, testResyncBetaSlug, b.search.query)
	assert.Equal(t, testResyncBetaSlug, b.searchInput.Value())
	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, beta.ID(), b.search.visibleConversations[0].ID())
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
	}

	b := testBrowser(t)
	b.store = store
	b.search.baseConversations = []conv.Conversation{alpha, beta}
	b.mainConversations = b.search.baseConversations

	var cmds []tea.Cmd
	b = b.applyFullConversationList(&cmds)

	cmds = nil
	b = b.setSearchQuery("alpha", &cmds)
	assert.Equal(t, searchStatusDebouncing, b.search.status)

	b, cmd := b.Update(deepSearchDebounceMsg{revision: b.search.revision, query: b.search.query})
	require.NotNil(t, cmd)
	b, _ = b.Update(requireMsgType[deepSearchResultMsg](t, cmd()))

	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, alpha.ID(), b.search.visibleConversations[0].ID())

	cmds = nil
	b = b.setSearchQuery("beta", &cmds)
	b, cmd = b.Update(deepSearchDebounceMsg{revision: b.search.revision, query: b.search.query})
	require.NotNil(t, cmd)
	b, _ = b.Update(requireMsgType[deepSearchResultMsg](t, cmd()))

	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, beta.ID(), b.search.visibleConversations[0].ID())
}

func TestBrowserDeepSearchEmptyResultShowsEmptyList(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-session")
	beta := testNamedConversation("beta-id", "beta-session")

	b := testBrowser(t)
	b.store = &fakeBrowserStore{}
	b.search.baseConversations = []conv.Conversation{alpha, beta}
	b.mainConversations = b.search.baseConversations

	var cmds []tea.Cmd
	b = b.applyFullConversationList(&cmds)

	cmds = nil
	b = b.setSearchQuery("missing", &cmds)
	b, cmd := b.Update(deepSearchDebounceMsg{revision: b.search.revision, query: b.search.query})
	require.NotNil(t, cmd)
	b, _ = b.Update(requireMsgType[deepSearchResultMsg](t, cmd()))

	assert.Equal(t, searchStatusIdle, b.search.status)
	assert.Empty(t, b.search.visibleConversations)
	assert.Empty(t, b.list.Items())
}

func TestBrowserIgnoresStaleDeepSearchResults(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-session")
	beta := testNamedConversation("beta-id", "beta-session")

	b := testBrowser(t)
	b.search.baseConversations = []conv.Conversation{alpha, beta}
	b.mainConversations = b.search.baseConversations
	var cmds []tea.Cmd
	b = b.applyFullConversationList(&cmds)

	b.search.query = "beta"
	b.search.revision = 3
	b.search.visibleConversations = []conv.Conversation{beta}
	b = b.setSearchItems(buildDeepSearchItems("beta", []conv.Conversation{beta}), &cmds)

	b, _ = b.Update(deepSearchResultMsg{
		revision:      2,
		query:         "alpha",
		conversations: []conv.Conversation{alpha},
	})

	require.Len(t, b.search.visibleConversations, 1)
	assert.Equal(t, beta.ID(), b.search.visibleConversations[0].ID())
}

func TestBrowserDeepSearchErrorShowsNotification(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-session")
	store := &fakeBrowserStore{deepSearchErr: assert.AnError}

	b := testBrowser(t)
	b.store = store
	b.search.baseConversations = []conv.Conversation{alpha}
	b.mainConversations = b.search.baseConversations

	var cmds []tea.Cmd
	b = b.applyFullConversationList(&cmds)
	cmds = nil
	b = b.setSearchQuery("alpha", &cmds)
	b, cmd := b.Update(deepSearchDebounceMsg{revision: b.search.revision, query: b.search.query})
	require.NotNil(t, cmd)
	b, _ = b.Update(requireMsgType[deepSearchResultMsg](t, cmd()))

	assert.Equal(t, searchStatusIdle, b.search.status)
	assert.Equal(t, "deep search failed: assert.AnError general error for testing", b.notification.text)
}

func testNamedConversation(id, slug string) conv.Conversation {
	return conv.Conversation{
		Ref:     conv.Ref{Provider: conv.ProviderClaude, ID: id},
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
