package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitPaletteKeepsFirstPalette(t *testing.T) {
	t.Parallel()

	beforePrimary := colorPrimary
	beforeAccent := colorAccent
	beforeToolCall := styleToolCall.Render("tool")

	initPalette(false)

	assert.Equal(t, beforePrimary, colorPrimary)
	assert.Equal(t, beforeAccent, colorAccent)
	assert.Equal(t, beforeToolCall, styleToolCall.Render("tool"))
}
