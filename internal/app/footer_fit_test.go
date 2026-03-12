package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestRenderHelpFooterKeepsHelpVisibleWhenNarrow(t *testing.T) {
	t.Parallel()

	footer := renderHelpFooter(
		44,
		[]helpItem{
			{key: "j/k", desc: "move"},
			{key: "gg", desc: "top"},
			{key: "G", desc: "bottom"},
			{key: "ctrl+f/b", desc: "page"},
			{key: "?", desc: "help", priority: helpPriorityEssential},
			{key: "q/esc", desc: "back", priority: helpPriorityHigh},
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

	status := ansi.Strip(renderHelpFooter(m.width, m.footerItems(), m.footerStatusParts(), notification{}))

	assert.Contains(t, status, viewerLineRangeStatus(m.viewport))
	assert.NotContains(t, status, "%")
}
