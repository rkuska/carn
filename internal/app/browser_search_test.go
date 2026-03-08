package app

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestBuildMetadataSearchItemsUsesFuzzyMatches(t *testing.T) {
	t.Parallel()

	convs := []conversation{
		testConv("one"),
		{
			name:    "archiver",
			project: project{dirName: "test", displayName: "test"},
			sessions: []sessionMeta{
				{id: "two", slug: "archiver", timestamp: testConv("two").sessions[0].timestamp},
			},
		},
	}

	items := buildMetadataSearchItems("arv", convs)
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	if items[0].conversation.id() != "two" {
		t.Fatalf("matched id = %q, want two", items[0].conversation.id())
	}
	if len(items[0].matchRanges.title) == 0 {
		t.Fatal("expected title matches for fuzzy metadata search")
	}
}

func TestBuildDeepSearchItemsHighlightsPreviewMatches(t *testing.T) {
	t.Parallel()

	conv := testConv("one")
	conv.searchPreview = archiveMatchesSourceSubtitle

	items := buildDeepSearchItems("archive", []conversation{conv})
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	if len(items[0].matchRanges.title) != 0 {
		t.Fatalf("title matches = %v, want none", items[0].matchRanges.title)
	}
	if len(items[0].matchRanges.desc) == 0 {
		t.Fatal("expected description matches for deep search preview")
	}
}

func TestSubstringMatchIndicesFindsAllCaseInsensitiveMatches(t *testing.T) {
	t.Parallel()

	matches := substringMatchIndices("Archive archive", "ARCH")
	if len(matches) != 8 {
		t.Fatalf("matches len = %d, want 8", len(matches))
	}
}

func TestBrowserSearchBindingUsesSlash(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b, cmd := b.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	if cmd == nil {
		t.Fatal("expected search blink command")
	}
	if !b.search.editing {
		t.Fatal("expected browser search editing to be active")
	}
	if !b.searchInput.Focused() {
		t.Fatal("expected browser search input to be focused")
	}
}

func TestBrowserCanToggleDeepSearchWhileEditingQuery(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.search.baseConversations = []conversation{testNamedConversation("one", "one")}
	var cmds []tea.Cmd
	b.applyFullConversationList(&cmds)
	b.search.editing = true
	b.searchInput.Focus()

	b, _ = b.handleSearchKey(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, &cmds)
	if b.search.mode != searchModeDeep {
		t.Fatalf("search mode = %v, want deep", b.search.mode)
	}
}

func TestBrowserDeepSearchRefreshesWhenQueryChanges(t *testing.T) {
	t.Parallel()

	alpha := testNamedConversation("alpha-id", "alpha-session")
	beta := testNamedConversation("beta-id", "beta-session")

	b := testBrowser(t)
	b.sessionCache[alpha.id()] = testSearchSession(alpha.id(), "contains alpha needle")
	b.sessionCache[beta.id()] = testSearchSession(beta.id(), "contains beta needle")
	b.search.baseConversations = []conversation{alpha, beta}
	b.mainConversationCount = 2

	var cmds []tea.Cmd
	b.applyFullConversationList(&cmds)
	b.toggleSearchMode(&cmds)

	cmds = nil
	b.setSearchQuery("alpha", &cmds)
	if b.search.status != searchStatusDebouncing {
		t.Fatalf("search status = %v, want debouncing", b.search.status)
	}

	b, cmd := b.Update(deepSearchDebounceMsg{revision: b.search.revision, query: b.search.query})
	if cmd == nil {
		t.Fatal("expected deep search command")
	}
	b, _ = b.Update(cmd().(deepSearchResultMsg))

	if len(b.search.visibleConversations) != 1 {
		t.Fatalf("visible conversations = %d, want 1", len(b.search.visibleConversations))
	}
	if b.search.visibleConversations[0].id() != alpha.id() {
		t.Fatalf("first visible id = %q, want %q", b.search.visibleConversations[0].id(), alpha.id())
	}

	cmds = nil
	b.setSearchQuery("beta", &cmds)
	b, cmd = b.Update(deepSearchDebounceMsg{revision: b.search.revision, query: b.search.query})
	if cmd == nil {
		t.Fatal("expected deep search command for updated query")
	}
	b, _ = b.Update(cmd().(deepSearchResultMsg))

	if len(b.search.visibleConversations) != 1 {
		t.Fatalf("visible conversations = %d, want 1 after query change", len(b.search.visibleConversations))
	}
	if b.search.visibleConversations[0].id() != beta.id() {
		t.Fatalf("first visible id = %q, want %q after query change", b.search.visibleConversations[0].id(), beta.id())
	}
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
	b.setSearchItems(buildDeepSearchItems("beta", []conversation{beta}), &cmds)

	b, _ = b.Update(deepSearchResultMsg{
		revision:      2,
		query:         "alpha",
		conversations: []conversation{alpha},
	})

	if len(b.search.visibleConversations) != 1 {
		t.Fatalf("visible conversations = %d, want 1", len(b.search.visibleConversations))
	}
	if b.search.visibleConversations[0].id() != beta.id() {
		t.Fatalf("first visible id = %q, want %q", b.search.visibleConversations[0].id(), beta.id())
	}
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

	var cmds []tea.Cmd
	b.setSearchQuery("beta-browser", &cmds)
	if len(b.search.visibleConversations) != 1 || b.search.visibleConversations[0].id() != beta.id() {
		t.Fatalf("metadata visible = %#v, want only beta", b.search.visibleConversations)
	}

	b.toggleSearchMode(&cmds)
	if len(cmds) == 0 {
		t.Fatal("expected deep search command when toggling search mode with active query")
	}
	deepCmd := cmds[len(cmds)-1]
	b, _ = b.Update(deepCmd().(deepSearchResultMsg))
	if len(b.search.visibleConversations) != 0 {
		t.Fatalf("deep visible conversations = %d, want 0", len(b.search.visibleConversations))
	}

	cmds = nil
	b.toggleSearchMode(&cmds)
	if b.search.mode != searchModeMetadata {
		t.Fatalf("search mode = %v, want metadata", b.search.mode)
	}
	if len(b.search.visibleConversations) != 1 || b.search.visibleConversations[0].id() != beta.id() {
		t.Fatalf("metadata visible after toggle = %#v, want only beta", b.search.visibleConversations)
	}
}

func testNamedConversation(id, slug string) conversation {
	return conversation{
		name:    slug,
		project: project{dirName: "test", displayName: "test"},
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
