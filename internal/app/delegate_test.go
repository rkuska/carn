package app

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	"github.com/charmbracelet/x/ansi"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitItemMatches(t *testing.T) {
	t.Parallel()

	title := "my/project / cheerful-ocean  2024-06-15 14:30"
	desc := "claude-3  25 msgs\n" + archiveMatchesSourceSubtitle
	full := title + "\n" + desc
	matchAt := strings.Index(full, "archive")
	require.GreaterOrEqual(t, matchAt, 0)

	matches := make([]int, len("archive"))
	for i := range matches {
		matches[i] = matchAt + i
	}

	got := splitItemMatches(title, desc, matches)
	assert.Empty(t, got.title)

	descRunes := []rune(desc)
	var highlighted strings.Builder
	for _, idx := range got.desc {
		highlighted.WriteRune(descRunes[idx])
	}

	assert.Equal(t, "archive", highlighted.String())
}

// renderDeepSearchView builds deep search items from the given query and
// conversation, sets them on a browser list with deep search delegate height,
// and returns the rendered list view.
func renderDeepSearchView(t *testing.T, query string, conversation conv.Conversation) string {
	t.Helper()

	b := testBrowser(t)
	b.delegate.SetHeight(delegateHeightDeepSearch)
	b.list.SetDelegate(b.delegate)

	items := buildDeepSearchItems(query, []conv.Conversation{conversation})
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	b.list.SetItems(listItems)

	return b.list.View()
}

func TestDeepSearchRenderHighlightsMultiWordQuery(t *testing.T) {
	t.Parallel()

	conv := testConv(testConversationIDPrimary)
	conv.SearchPreview = "the matching strategy uses string comparison"

	highlighted := renderDeepSearchView(t, "matching strings", conv)
	unhighlighted := renderDeepSearchView(t, "", conv)

	strippedHighlighted := ansi.Strip(highlighted)
	strippedUnhighlighted := ansi.Strip(unhighlighted)

	// Both should contain the preview text.
	assert.Contains(t, strippedHighlighted, "matching")
	assert.Contains(t, strippedUnhighlighted, "matching")

	// Highlighted output must differ from unhighlighted due to ANSI styling.
	assert.NotEqual(t, highlighted, unhighlighted,
		"multi-word query should produce highlighted output different from unhighlighted")
}

func TestDeepSearchRenderHighlightsSingleWordQuery(t *testing.T) {
	t.Parallel()

	conv := testConv(testConversationIDPrimary)
	conv.SearchPreview = "analysis complete; archive already matches the source"

	highlighted := renderDeepSearchView(t, "archive", conv)
	unhighlighted := renderDeepSearchView(t, "", conv)

	assert.NotEqual(t, highlighted, unhighlighted,
		"single-word query should produce highlighted output different from unhighlighted")
}

func TestDeepSearchRenderHighlightsAcrossPreviewLines(t *testing.T) {
	t.Parallel()

	conv := testConv(testConversationIDPrimary)
	conv.SearchPreview = "the cache layer is fast\nrebuild cache on startup"

	highlighted := renderDeepSearchView(t, "cache", conv)
	unhighlighted := renderDeepSearchView(t, "", conv)

	strippedHighlighted := ansi.Strip(highlighted)

	// Both lines should be visible in the rendered output.
	assert.Contains(t, strippedHighlighted, "cache layer")
	assert.Contains(t, strippedHighlighted, "rebuild cache")

	// Highlighting must produce different ANSI output.
	assert.NotEqual(t, highlighted, unhighlighted,
		"query matching across lines should produce highlighted output different from unhighlighted")
}

func TestDeepSearchRenderNoHighlightWhenQueryAbsent(t *testing.T) {
	t.Parallel()

	conv := testConv(testConversationIDPrimary)
	conv.SearchPreview = "some unrelated preview text"

	withQuery := renderDeepSearchView(t, "xyznotfound", conv)
	withoutQuery := renderDeepSearchView(t, "", conv)

	// When query words don't appear in the description, rendering should
	// be identical to the unhighlighted (empty query) rendering.
	assert.Equal(t, withQuery, withoutQuery,
		"non-matching query should produce identical output to empty query")
}
