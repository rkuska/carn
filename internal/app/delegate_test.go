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

const (
	ansiColorBright = "38;5;249m"
	ansiColorGrey   = "38;5;243m"
	ansiColorGreen  = "38;5;156m"
)

func TestSplitItemMatches(t *testing.T) {
	t.Parallel()

	title := "my/project / cheerful-ocean  2024-06-15 14:30"
	metadata := "claude-3  25 msgs"
	preview := archiveMatchesSourceSubtitle
	full := title + "\n" + metadata + "\n" + preview
	matchAt := strings.Index(full, "archive")
	require.GreaterOrEqual(t, matchAt, 0)

	matches := make([]int, len("archive"))
	for i := range matches {
		matches[i] = matchAt + i
	}

	got := splitItemMatches(title, metadata, preview, matches)
	assert.Empty(t, got.title)
	assert.Empty(t, got.metadata)

	descRunes := []rune(preview)
	var highlighted strings.Builder
	for _, idx := range got.preview {
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

func renderConversationItem(
	t *testing.T,
	item conversationListItem,
	height int,
	selected bool,
) string {
	t.Helper()

	d := newDelegate()
	d.SetHeight(height)

	items := []list.Item{
		conversationListItem{title: "other", metadata: "other metadata", preview: "other preview"},
		item,
	}

	l := list.New(items, d, 120, 10)
	if selected {
		l.Select(1)
	} else {
		l.Select(0)
	}

	var rendered strings.Builder
	d.Render(&rendered, l, 1, item)
	return rendered.String()
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

func TestRenderPlainItemUsesGreyMetadataAndBrightPreview(t *testing.T) {
	t.Parallel()

	conversation := testConv(testConversationIDPrimary)
	conversation.Sessions[0].FirstMessage = "plain preview"

	items := buildPlainConversationItems([]conv.Conversation{conversation})
	require.Len(t, items, 1)

	rendered := renderConversationItem(t, items[0], delegateHeightDefault, false)
	lines := strings.Split(rendered, "\n")
	require.Len(t, lines, 3)

	assert.Contains(t, ansi.Strip(lines[0]), conversation.Title())
	assert.Contains(t, lines[0], ansiColorGrey)
	assert.NotContains(t, lines[0], ansiColorBright)
	assert.Contains(t, ansi.Strip(lines[1]), "Claude")
	assert.Contains(t, lines[1], ansiColorGrey)
	assert.NotContains(t, lines[1], ansiColorGreen)
	assert.Contains(t, ansi.Strip(lines[2]), "plain preview")
	assert.Contains(t, lines[2], ansiColorBright)
	assert.NotContains(t, lines[2], ansiColorGrey)
}

func TestRenderSelectedDeepSearchItemHighlightsAllLinesGreen(t *testing.T) {
	t.Parallel()

	conversation := testConv(testConversationIDPrimary)
	conversation.SearchPreview = "preview match line"

	items := buildDeepSearchItems("", []conv.Conversation{conversation})
	require.Len(t, items, 1)

	rendered := renderConversationItem(t, items[0], delegateHeightDeepSearch, true)
	lines := strings.Split(rendered, "\n")
	require.GreaterOrEqual(t, len(lines), 3)

	assert.Contains(t, ansi.Strip(lines[0]), conversation.Title())
	assert.Contains(t, lines[0], ansiColorGreen)
	assert.Contains(t, ansi.Strip(lines[1]), "Claude")
	assert.Contains(t, lines[1], ansiColorGreen)
	assert.Contains(t, ansi.Strip(lines[2]), "preview match line")
	assert.Contains(t, lines[2], ansiColorGreen)
}
