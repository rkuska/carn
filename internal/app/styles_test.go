package app

import (
	"testing"

	"charm.land/lipgloss/v2"
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
