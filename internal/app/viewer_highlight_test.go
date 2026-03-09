package app

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindMatchRanges(t *testing.T) {
	t.Parallel()

	style := lipgloss.NewStyle().Bold(true)

	tests := []struct {
		name       string
		stripped   string
		queryLower string
		wantCount  int
		wantStarts []int
		wantEnds   []int
	}{
		{
			name:       "empty query returns nil",
			stripped:   "hello world",
			queryLower: "",
			wantCount:  0,
		},
		{
			name:       "empty text returns nil",
			stripped:   "",
			queryLower: "hello",
			wantCount:  0,
		},
		{
			name:       "no match",
			stripped:   "hello world",
			queryLower: "xyz",
			wantCount:  0,
		},
		{
			name:       "single match at start",
			stripped:   "hello world",
			queryLower: "hello",
			wantCount:  1,
			wantStarts: []int{0},
			wantEnds:   []int{5},
		},
		{
			name:       "single match at end",
			stripped:   "hello world",
			queryLower: "world",
			wantCount:  1,
			wantStarts: []int{6},
			wantEnds:   []int{11},
		},
		{
			name:       "multiple matches",
			stripped:   "foo bar foo baz foo",
			queryLower: "foo",
			wantCount:  3,
			wantStarts: []int{0, 8, 16},
			wantEnds:   []int{3, 11, 19},
		},
		{
			name:       "case insensitive via pre-lowered text",
			stripped:   "Hello HELLO hello",
			queryLower: "hello",
			wantCount:  3,
			wantStarts: []int{0, 6, 12},
			wantEnds:   []int{5, 11, 17},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ranges := findMatchRanges(tt.stripped, tt.queryLower, style)

			assert.Len(t, ranges, tt.wantCount)
			for i, r := range ranges {
				if i < len(tt.wantStarts) {
					assert.Equal(t, tt.wantStarts[i], r.Start, "range %d start", i)
					assert.Equal(t, tt.wantEnds[i], r.End, "range %d end", i)
				}
			}
		})
	}
}

func TestHighlightLine(t *testing.T) {
	t.Parallel()

	style := lipgloss.NewStyle().Bold(true)

	tests := []struct {
		name       string
		line       string
		queryLower string
		wantChange bool
	}{
		{
			name:       "plain text with match",
			line:       "hello world",
			queryLower: "world",
			wantChange: true,
		},
		{
			name:       "text with ANSI codes",
			line:       "\x1b[32mhello\x1b[0m world",
			queryLower: "hello",
			wantChange: true,
		},
		{
			name:       "no match returns original",
			line:       "hello world",
			queryLower: "xyz",
			wantChange: false,
		},
		{
			name:       "empty line returns original",
			line:       "",
			queryLower: "test",
			wantChange: false,
		},
		{
			name:       "empty query returns original",
			line:       "hello world",
			queryLower: "",
			wantChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := highlightLine(tt.line, tt.queryLower, style)
			if tt.wantChange {
				assert.NotEqual(t, tt.line, result)
				// The stripped content should be the same (only styling changed).
				assert.Equal(t, ansi.Strip(tt.line), ansi.Strip(result))
			} else {
				assert.Equal(t, tt.line, result)
			}
		})
	}
}

func TestHighlightLineOccurrences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		line       string
		queryLower string
		occs       []lineOccurrence
		wantChange bool
	}{
		{
			name:       "no occurrences returns original",
			line:       "hello world",
			queryLower: "hello",
			occs:       nil,
			wantChange: false,
		},
		{
			name:       "single current match highlights",
			line:       "hello world",
			queryLower: "hello",
			occs:       []lineOccurrence{{byteStart: 0, isCurrentMatch: true}},
			wantChange: true,
		},
		{
			name:       "single non-current match highlights",
			line:       "hello world",
			queryLower: "hello",
			occs:       []lineOccurrence{{byteStart: 0, isCurrentMatch: false}},
			wantChange: true,
		},
		{
			name:       "two occurrences with different styles",
			line:       "foo bar foo",
			queryLower: "foo",
			occs: []lineOccurrence{
				{byteStart: 0, isCurrentMatch: true},
				{byteStart: 8, isCurrentMatch: false},
			},
			wantChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := highlightLineOccurrences(tt.line, tt.queryLower, tt.occs)
			if tt.wantChange {
				assert.NotEqual(t, tt.line, result)
				assert.Equal(t, ansi.Strip(tt.line), ansi.Strip(result))
			} else {
				assert.Equal(t, tt.line, result)
			}
		})
	}
}

func TestHighlightLineOccurrencesCurrentVsNonCurrentDiffer(t *testing.T) {
	t.Parallel()

	line := "foo bar foo"
	query := "foo"

	// First occurrence is current.
	result1 := highlightLineOccurrences(line, query, []lineOccurrence{
		{byteStart: 0, isCurrentMatch: true},
		{byteStart: 8, isCurrentMatch: false},
	})

	// Second occurrence is current.
	result2 := highlightLineOccurrences(line, query, []lineOccurrence{
		{byteStart: 0, isCurrentMatch: false},
		{byteStart: 8, isCurrentMatch: true},
	})

	require.NotEqual(t, result1, result2)
}

func TestHighlightSearchMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		query        string
		matches      []searchOccurrence
		currentMatch int
		wantChanged  bool
	}{
		{
			name:         "empty query returns unchanged",
			content:      "line one\nline two",
			query:        "",
			matches:      nil,
			currentMatch: 0,
			wantChanged:  false,
		},
		{
			name:         "no matches returns unchanged",
			content:      "line one\nline two",
			query:        "xyz",
			matches:      nil,
			currentMatch: 0,
			wantChanged:  false,
		},
		{
			name:         "highlights matched line",
			content:      "first line\nsecond line\nthird line",
			query:        "second",
			matches:      []searchOccurrence{{line: 1, byteStart: 0}},
			currentMatch: 0,
			wantChanged:  true,
		},
		{
			name:    "current match differs from non-current",
			content: "foo bar\nfoo baz\nfoo qux",
			query:   "foo",
			matches: []searchOccurrence{
				{line: 0, byteStart: 0},
				{line: 1, byteStart: 0},
				{line: 2, byteStart: 0},
			},
			currentMatch: 1,
			wantChanged:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := highlightSearchMatches(tt.content, tt.query, tt.matches, tt.currentMatch)
			if tt.wantChanged {
				assert.NotEqual(t, tt.content, result)
				// Stripped text content should be preserved.
				assert.Equal(t, ansi.Strip(tt.content), ansi.Strip(result))
			} else {
				assert.Equal(t, tt.content, result)
			}
		})
	}
}

func TestHighlightSearchMatchesCurrentVsNonCurrent(t *testing.T) {
	t.Parallel()

	content := "foo bar\nfoo baz\nfoo qux"
	matches := []searchOccurrence{
		{line: 0, byteStart: 0},
		{line: 1, byteStart: 0},
		{line: 2, byteStart: 0},
	}

	resultCurrent0 := highlightSearchMatches(content, "foo", matches, 0)
	resultCurrent1 := highlightSearchMatches(content, "foo", matches, 1)

	// Different current match should produce different output.
	require.NotEqual(t, resultCurrent0, resultCurrent1)

	// Line 0 in resultCurrent0 should differ from line 0 in resultCurrent1
	// because line 0 is current in one and non-current in the other.
	lines0 := strings.Split(resultCurrent0, "\n")
	lines1 := strings.Split(resultCurrent1, "\n")
	assert.NotEqual(t, lines0[0], lines1[0])
	assert.NotEqual(t, lines0[1], lines1[1])
}

func TestHighlightSearchMatchesOnlyCurrentOccurrenceGetsCurrentStyle(t *testing.T) {
	t.Parallel()

	content := "foo bar foo"
	matches := []searchOccurrence{
		{line: 0, byteStart: 0},
		{line: 0, byteStart: 8},
	}

	resultCurrent0 := highlightSearchMatches(content, "foo", matches, 0)
	resultCurrent1 := highlightSearchMatches(content, "foo", matches, 1)

	// Different current occurrence on the same line should produce different output.
	require.NotEqual(t, resultCurrent0, resultCurrent1)
}
