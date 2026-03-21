package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testViewportAlphaBeta = "alpha\nbeta"

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

func TestCollectSearchOccurrencesEmptyQueryReturnsNil(t *testing.T) {
	t.Parallel()

	lines := []searchLineIndex{buildSearchLineIndex("hello world", 0)}

	assert.Nil(t, collectSearchOccurrences(lines, ""))
}

func TestCollectSearchOccurrencesEmptyLinesReturnsEmpty(t *testing.T) {
	t.Parallel()

	lines := []searchLineIndex{
		buildSearchLineIndex("", 0),
		buildSearchLineIndex("   ", 0),
	}

	assert.Empty(t, collectSearchOccurrences(lines, "hello"))
}

func TestCollectSearchOccurrencesCaseInsensitive(t *testing.T) {
	t.Parallel()

	lines := []searchLineIndex{buildSearchLineIndex("HeLLo world", 0)}
	matches := collectSearchOccurrences(lines, "hello")

	require.Len(t, matches, 1)
	assert.Equal(t, searchOccurrence{line: 0, byteStart: 0}, matches[0])
}

func TestCollectSearchOccurrencesMultipleMatchesOnOneLine(t *testing.T) {
	t.Parallel()

	lines := []searchLineIndex{buildSearchLineIndex("foo bar foo baz foo", 0)}
	matches := collectSearchOccurrences(lines, "foo")

	require.Len(t, matches, 3)
	assert.Equal(t, searchOccurrence{line: 0, byteStart: 0}, matches[0])
	assert.Equal(t, searchOccurrence{line: 0, byteStart: 8}, matches[1])
	assert.Equal(t, searchOccurrence{line: 0, byteStart: 16}, matches[2])
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

func TestHighlightLineOccurrencesNoOccurrencesReturnsOriginal(t *testing.T) {
	t.Parallel()

	line := "foo bar baz"
	assert.Equal(t, line, highlightLineOccurrences(line, "foo", nil))
}

func TestHighlightLineOccurrencesEmptyQueryReturnsOriginal(t *testing.T) {
	t.Parallel()

	line := "foo bar baz"
	assert.Equal(t, line, highlightLineOccurrences(line, "", []lineOccurrence{{}}))
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

func TestHighlightViewportMatchesEmptyQueryReturnsOriginal(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		testViewportAlphaBeta,
		highlightViewportMatches(testViewportAlphaBeta, "", []searchOccurrence{{line: 0, byteStart: 0}}, 0, 0),
	)
}

func TestHighlightViewportMatchesNoMatchesReturnsOriginal(t *testing.T) {
	t.Parallel()

	assert.Equal(t, testViewportAlphaBeta, highlightViewportMatches(testViewportAlphaBeta, "hello", nil, 0, 0))
}

func TestHighlightViewportMatchesAllMatchesOutOfViewport(t *testing.T) {
	t.Parallel()

	matches := []searchOccurrence{
		{line: 4, byteStart: 0},
		{line: 5, byteStart: 3},
	}

	assert.Equal(t, testViewportAlphaBeta, highlightViewportMatches(testViewportAlphaBeta, "alpha", matches, 0, 0))
}
