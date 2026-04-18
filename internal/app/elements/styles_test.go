package elements

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestNewThemeReturnsIndependentPalettes(t *testing.T) {
	dark := NewTheme(true)
	light := NewTheme(false)

	assert.NotEqual(t, dark.ColorPrimary, light.ColorPrimary)
	assert.NotEqual(t, dark.ColorAccent, light.ColorAccent)
	assert.NotEqual(t, dark.StyleToolCall.Render("tool"), light.StyleToolCall.Render("tool"))
}

func TestNewThemeUsesSuggestedTokenChartPalette(t *testing.T) {
	theme := NewTheme(true)

	assert.Equal(t, lipgloss.Color("#a371f7"), theme.ColorChartToken)
	assert.Equal(t, lipgloss.Color("#d2a8ff"), theme.ColorChartTime)
}

func TestThemeRenderFramedBoxKeepsBorderWidth(t *testing.T) {
	theme := NewTheme(true)

	got := ansi.Strip(theme.RenderFramedBox("Box", 14, theme.ColorPrimary, "alpha\nbeta"))
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

func TestThemeRenderFramedPanePadsBodyHeight(t *testing.T) {
	theme := NewTheme(true)

	got := ansi.Strip(theme.RenderFramedPane("Pane", 14, 3, theme.ColorPrimary, "alpha"))
	lines := strings.Split(got, "\n")

	assert.Len(t, lines, 5)
	for _, line := range lines {
		assert.Equal(t, 14, lipgloss.Width(line))
	}
	assert.Contains(t, lines[0], "Pane")
	assert.Contains(t, lines[1], "alpha")
	assert.Contains(t, lines[4], "╯")
}
