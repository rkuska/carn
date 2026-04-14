package elements

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestInitPaletteForTestReinitializesPalette(t *testing.T) {
	InitPaletteForTest(true)
	t.Cleanup(func() {
		InitPaletteForTest(true)
	})

	beforePrimary := ColorPrimary
	beforeAccent := ColorAccent
	beforeToolCall := StyleToolCall.Render("tool")

	InitPaletteForTest(false)

	assert.NotEqual(t, beforePrimary, ColorPrimary)
	assert.NotEqual(t, beforeAccent, ColorAccent)
	assert.NotEqual(t, beforeToolCall, StyleToolCall.Render("tool"))
}

func TestInitPaletteForTestUsesSuggestedTokenChartPalette(t *testing.T) {
	InitPaletteForTest(true)
	t.Cleanup(func() {
		InitPaletteForTest(true)
	})

	assert.Equal(t, lipgloss.Color("#a371f7"), ColorChartToken)
	assert.Equal(t, lipgloss.Color("#d2a8ff"), ColorChartTime)
}

func TestRenderFramedBoxKeepsBorderWidth(t *testing.T) {
	InitPaletteForTest(true)
	t.Cleanup(func() {
		InitPaletteForTest(true)
	})

	got := ansi.Strip(RenderFramedBox("Box", 14, ColorPrimary, "alpha\nbeta"))
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
	InitPaletteForTest(true)
	t.Cleanup(func() {
		InitPaletteForTest(true)
	})

	got := ansi.Strip(RenderFramedPane("Pane", 14, 3, ColorPrimary, "alpha"))
	lines := strings.Split(got, "\n")

	assert.Len(t, lines, 5)
	for _, line := range lines {
		assert.Equal(t, 14, lipgloss.Width(line))
	}
	assert.Contains(t, lines[0], "Pane")
	assert.Contains(t, lines[1], "alpha")
	assert.Contains(t, lines[4], "╯")
}
