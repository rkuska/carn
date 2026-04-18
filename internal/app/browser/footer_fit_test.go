package browser

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestRenderHelpFooterKeepsHelpVisibleWhenNarrow(t *testing.T) {
	t.Parallel()

	footer := renderHelpFooter(testTheme(),
		44,
		[]helpItem{
			{Key: "j/k", Desc: "move"},
			{Key: "gg", Desc: "top"},
			{Key: "G", Desc: "bottom"},
			{Key: "ctrl+f/b", Desc: "page"},
			{Key: "?", Desc: "help", Priority: helpPriorityEssential},
			{Key: "q/esc", Desc: "back", Priority: helpPriorityHigh},
		},
		[]string{"very long status value that leaves little room"},
		notification{},
	)

	lines := strings.Split(ansi.Strip(footer), "\n")
	assert.Contains(t, lines[0], "? help")
	assert.NotContains(t, lines[0], "j/k move")
}

func TestViewerFooterStatusShowsLineRange(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSessionLong("viewer-lines", "KEYWORD"), 120, 12)
	m.viewport.SetYOffset(5)

	status := ansi.Strip(renderHelpFooter(testTheme(), m.width, m.footerItems(), m.footerStatusParts(), notification{}))

	assert.Contains(t, status, viewerLineRangeStatus(m.viewport))
	assert.NotContains(t, status, "%")
}
