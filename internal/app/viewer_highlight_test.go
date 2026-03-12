package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSearchLineIndexStripsAnsi(t *testing.T) {
	t.Parallel()

	index := buildSearchLineIndex("\x1b[31mhello\x1b[0m world", 0)
	assert.Equal(t, "hello world", index.lower)
}

func TestCollectSearchOccurrencesReturnsOrderedMatches(t *testing.T) {
	t.Parallel()

	lines := []searchLineIndex{
		buildSearchLineIndex("\x1b[31mhello\x1b[0m world", 0),
		buildSearchLineIndex("say hello again", 0),
	}

	matches := collectSearchOccurrences(lines, "hello")
	require.Len(t, matches, 2)

	assert.Equal(t, 0, matches[0].line)
	assert.Equal(t, 0, matches[0].byteStart)
	assert.Equal(t, 1, matches[1].line)
	assert.Equal(t, 4, matches[1].byteStart)
}

func TestHighlightLineOccurrences(t *testing.T) {
	t.Parallel()

	result := highlightLineOccurrences(
		"foo bar foo",
		"foo",
		[]lineOccurrence{
			{isCurrentMatch: true},
			{isCurrentMatch: false},
		},
	)

	assert.NotEqual(t, "foo bar foo", result)
	assert.Equal(t, "foo bar foo", ansi.Strip(result))
}

func TestHighlightViewportMatchesHighlightsOnlyVisibleLines(t *testing.T) {
	t.Parallel()

	content := "\ntwo hello\nthree"
	matches := []searchOccurrence{
		{line: 1, byteStart: 4},
		{line: 3, byteStart: 4},
	}

	result := highlightViewportMatches(content, "hello", matches, 1, 2)
	lines := strings.Split(result, "\n")
	require.Len(t, lines, 3)

	assert.Equal(t, "", ansi.Strip(lines[0]))
	assert.Equal(t, "two hello", ansi.Strip(lines[1]))
	assert.Equal(t, "three", ansi.Strip(lines[2]))
	assert.Equal(t, lines[0], "")
	assert.NotEqual(t, lines[1], "two hello")
}
