package app

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
)

func TestBrowserSplitViewKeepsBottomBordersWithDeepSearchItems(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.width = 90
	b.height = 24
	b.transcriptMode = transcriptSplit
	b.focus = focusList

	first := testLongConv(testConversationIDPrimary)
	first.SetSearchPreview(strings.Join([]string{
		"recent refactor moved the import flow into a pipeline stage",
		"deep search still needs the previous rebuild result",
		"the final border should stay visible in split view",
	}, "\n"))

	second := testLongConv(testConversationIDSecondary)
	second.SetSearchPreview(strings.Join([]string{
		"the browser view should not lose its bottom frame",
		"even when preview lines are expanded for deep search",
		"selection and transcript should keep the layout stable",
	}, "\n"))

	b.search.mode = searchModeDeep
	b.search.query = "refactor"
	b = b.setDelegateHeight(delegateHeightDeepSearch)
	items := buildDeepSearchItems("refactor", []conv.Conversation{first, second})
	b.list.SetItems([]list.Item{items[0], items[1]})
	b.list.Select(0)

	b.loadingConversationID = testConversationIDPrimary
	b, _ = b.Update(openViewerMsg{
		conversationID: testConversationIDPrimary,
		conversation:   first,
		session:        testBrowserSessionLong(testConversationIDPrimary, "refactor"),
	})
	b = b.updateLayout()

	view := ansi.Strip(b.View())
	lines := strings.Split(view, "\n")

	assert.Equal(t, b.height, lipgloss.Height(view))
	assert.Len(t, lines, b.height)

	borderLine := lines[b.height-3]
	assert.Equal(t, 2, strings.Count(borderLine, "╰"), view)
	assert.Equal(t, 2, strings.Count(borderLine, "╯"), view)
}
