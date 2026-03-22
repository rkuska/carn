package app

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderHelpOverlayItemRowUsesThreeColumns(t *testing.T) {
	t.Parallel()

	row := ansi.Strip(renderHelpOverlayItemRows(
		helpItem{
			key:    "R",
			desc:   "resync",
			detail: "refresh sources and rebuild the local store",
		},
		7,
		8,
		48,
	)[0])

	assert.Contains(t, row, "R")
	assert.Contains(t, row, "resync")
	assert.Contains(t, row, "refresh sources")
	assert.NotContains(t, row, "R resync")
}

func TestRenderHelpOverlayItemRowsWrapDetailToFitWidth(t *testing.T) {
	t.Parallel()

	rows := renderHelpOverlayItemRows(
		helpItem{
			key:    "ctrl+f/b",
			desc:   "page",
			detail: "jump a page up or down while keeping the current selection visible",
		},
		8,
		5,
		24,
	)

	require.Greater(t, len(rows), 1)
	for _, row := range rows {
		assert.LessOrEqual(t, lipgloss.Width(ansi.Strip(row)), 26)
	}
	joined := ansi.Strip(strings.Join(rows, "\n"))
	assert.Contains(t, joined, "ctrl+f/b")
	assert.Contains(t, joined, "page")
	assert.Contains(t, joined, "jump")
	assert.Contains(t, joined, "visible")
}

func TestRenderHelpOverlayKeepsRowsWithinViewportWidth(t *testing.T) {
	t.Parallel()

	overlay := ansi.Strip(renderHelpOverlay(
		48,
		16,
		"Help",
		[]helpSection{
			{
				title: "Actions",
				items: []helpItem{
					{
						key:    "enter",
						desc:   "open",
						detail: "open the selected conversation in split view",
					},
					{
						key:    "R",
						desc:   "resync",
						detail: "refresh raw sessions and rebuild the local store",
					},
				},
			},
		},
	))

	for line := range strings.SplitSeq(overlay, "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), 48)
	}
	assert.Contains(t, overlay, "resync")
	assert.Contains(t, overlay, "refresh raw sessions")
	assert.Contains(t, overlay, "and rebuild the local")
	assert.Contains(t, overlay, "store")
}
