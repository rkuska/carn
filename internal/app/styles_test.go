package app

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestInitPaletteForTestReinitializesPalette(t *testing.T) {
	initPaletteForTest(true)
	t.Cleanup(func() {
		initPaletteForTest(true)
	})

	beforePrimary := colorPrimary
	beforeAccent := colorAccent
	beforeToolCall := styleToolCall.Render("tool")

	initPaletteForTest(false)

	assert.NotEqual(t, beforePrimary, colorPrimary)
	assert.NotEqual(t, beforeAccent, colorAccent)
	assert.NotEqual(t, beforeToolCall, styleToolCall.Render("tool"))
}

func TestInitPaletteForTestUsesSuggestedTokenChartPalette(t *testing.T) {
	initPaletteForTest(true)
	t.Cleanup(func() {
		initPaletteForTest(true)
	})

	assert.Equal(t, lipgloss.Color("#a371f7"), colorChartToken)
	assert.Equal(t, lipgloss.Color("#d2a8ff"), colorChartTime)
}

func TestRenderFramedBoxKeepsBorderWidth(t *testing.T) {
	initPaletteForTest(true)
	t.Cleanup(func() {
		initPaletteForTest(true)
	})

	got := ansi.Strip(renderFramedBox("Box", 14, colorPrimary, "alpha\nbeta"))
	lines := strings.Split(got, "\n")

	assert.Len(t, lines, 4)
	for _, line := range lines {
		assert.Equal(t, 14, lipgloss.Width(line))
	}
	assert.Contains(t, lines[0], "Box")
	assert.Contains(t, lines[1], "alpha")
	assert.Contains(t, lines[2], "beta")
	assert.Contains(t, lines[3], "╯")
}

func TestRenderFramedPanePadsBodyHeight(t *testing.T) {
	initPaletteForTest(true)
	t.Cleanup(func() {
		initPaletteForTest(true)
	})

	got := ansi.Strip(renderFramedPane("Pane", 14, 3, colorPrimary, "alpha"))
	lines := strings.Split(got, "\n")

	assert.Len(t, lines, 5)
	for _, line := range lines {
		assert.Equal(t, 14, lipgloss.Width(line))
	}
	assert.Contains(t, lines[0], "Pane")
	assert.Contains(t, lines[1], "alpha")
	assert.Contains(t, lines[4], "╯")
}
