package app

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestRenderHelpOverlayItemRowUsesThreeColumns(t *testing.T) {
	t.Parallel()

	row := ansi.Strip(renderHelpOverlayItemRow(
		helpItem{
			key:    "R",
			desc:   "resync",
			detail: "refresh sources and rebuild the local store",
		},
		7,
		8,
		48,
	))

	assert.Contains(t, row, "R")
	assert.Contains(t, row, "resync")
	assert.Contains(t, row, "refresh sources")
	assert.NotContains(t, row, "R resync")
}

func TestRenderHelpOverlayItemRowTruncatesDetailToFitWidth(t *testing.T) {
	t.Parallel()

	row := ansi.Strip(renderHelpOverlayItemRow(
		helpItem{
			key:    "ctrl+f/b",
			desc:   "page",
			detail: "jump a page up or down while keeping the current selection visible",
		},
		8,
		5,
		24,
	))

	assert.Equal(t, 24, lipgloss.Width(row))
	assert.Contains(t, row, "…")
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
	assert.Contains(t, overlay, "…")
}
