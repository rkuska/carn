package app

import (
	"testing"

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
