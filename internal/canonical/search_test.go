package canonical

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestIsASCII(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{name: "empty", input: "", expect: true},
		{name: "pure_ascii", input: "hello world 123!@#", expect: true},
		{name: "byte_127", input: string([]byte{127}), expect: true},
		{name: "byte_128", input: string([]byte{128}), expect: false},
		{name: "unicode", input: "héllo", expect: false},
		{name: "emoji", input: "hello 🌍", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, isASCII(tt.input))
		})
	}
}

func TestChunkSearchTextShortASCII(t *testing.T) {
	t.Parallel()
	chunks := slices.Collect(chunkSearchText("short text", 160, 48))
	require.Len(t, chunks, 1)
	assert.Equal(t, "short text", chunks[0])
}

func TestChunkSearchTextShortUnicode(t *testing.T) {
	t.Parallel()
	text := "héllo wörld"
	chunks := slices.Collect(chunkSearchText(text, 160, 48))
	require.Len(t, chunks, 1)
	assert.Equal(t, text, chunks[0])
}

func TestChunkSearchTextLongASCII(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("a", 300)
	chunks := slices.Collect(chunkSearchText(text, 160, 48))

	require.Greater(t, len(chunks), 1)
	for _, chunk := range chunks {
		assert.LessOrEqual(t, len(chunk), 160)
	}
}

func TestChunkSearchTextLongUnicode(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("ö", 300)
	chunks := slices.Collect(chunkSearchText(text, 160, 48))

	require.Greater(t, len(chunks), 1)
	for _, chunk := range chunks {
		assert.LessOrEqual(t, len([]rune(chunk)), 160)
	}
}

func TestChunkSearchTextExactBoundary(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("x", 160)
	chunks := slices.Collect(chunkSearchText(text, 160, 48))
	require.Len(t, chunks, 1)
	assert.Equal(t, text, chunks[0])
}

func TestChunkSearchTextOverlapClamped(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("x", 300)
	chunks := slices.Collect(chunkSearchText(text, 160, 200))

	require.Greater(t, len(chunks), 1)
	for _, chunk := range chunks {
		assert.LessOrEqual(t, len(chunk), 160)
	}
}

func TestCanUseSearchFastPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		text   string
		expect bool
	}{
		{name: "empty", text: "", expect: false},
		{name: "pure ascii single line", text: "hello world", expect: true},
		{name: "ascii with internal tab", text: "hello\tworld", expect: true},
		{name: "ascii with newline", text: "hello\nworld", expect: false},
		{name: "ascii with leading space", text: " hello", expect: false},
		{name: "ascii with trailing tab", text: "hello\t", expect: false},
		{name: "unicode", text: "héllo world", expect: false},
		{name: "mixed ascii and emoji", text: "hello world 🚀", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, canUseSearchFastPath(tt.text))
		})
	}
}

func TestChunkSearchTextPreservesAllContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
	}{
		{name: "ascii", text: chunkSearchFixture("ascii")},
		{name: "unicode", text: chunkSearchFixture("żółw漢字")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chunks := slices.Collect(chunkSearchText(tt.text, 160, 48))
			require.Greater(t, len(chunks), 1)
			assert.Equal(t, tt.text, reassembleChunkedText(chunks))
		})
	}
}

func TestYieldSearchTextUnitsSkipsEmptyText(t *testing.T) {
	t.Parallel()

	called := false
	ordinal := yieldSearchTextUnits("", 7, func(int, string) bool {
		called = true
		return true
	})

	assert.False(t, called)
	assert.Equal(t, 7, ordinal)
}

func TestAppendSearchUnitsEmpty(t *testing.T) {
	t.Parallel()
	units := appendSearchUnits(nil, "id", "")
	assert.Empty(t, units)
}

func TestAppendSearchUnitsSingleLine(t *testing.T) {
	t.Parallel()
	units := appendSearchUnits(nil, "conv-1", "hello world")
	require.Len(t, units, 1)
	assert.Equal(t, "conv-1", units[0].conversationID)
	assert.Equal(t, 0, units[0].ordinal)
	assert.Equal(t, "hello world", units[0].text)
}

func TestAppendSearchUnitsMultiLineSkipsBlanks(t *testing.T) {
	t.Parallel()
	text := "line one\n\n  \nline two\n"
	units := appendSearchUnits(nil, "id", text)
	require.Len(t, units, 2)
	assert.Equal(t, "line one", units[0].text)
	assert.Equal(t, "line two", units[1].text)
}

func TestAppendSearchUnitsTrimsWhitespace(t *testing.T) {
	t.Parallel()
	units := appendSearchUnits(nil, "id", "  hello  ")
	require.Len(t, units, 1)
	assert.Equal(t, "hello", units[0].text)
}

func TestAppendSearchUnitsOrdinalSequence(t *testing.T) {
	t.Parallel()
	existing := []searchUnit{{conversationID: "id", ordinal: 0, text: "first"}}
	units := appendSearchUnits(existing, "id", "second\nthird")
	require.Len(t, units, 3)
	assert.Equal(t, 0, units[0].ordinal)
	assert.Equal(t, 1, units[1].ordinal)
	assert.Equal(t, 2, units[2].ordinal)
}

func TestBuildSearchUnitsExcludesHiddenSystem(t *testing.T) {
	t.Parallel()
	session := sessionFull{
		Messages: []message{
			{Role: conv.RoleUser, Text: "visible", Visibility: conv.MessageVisibilityVisible},
			{Role: conv.RoleSystem, Text: "hidden", Visibility: conv.MessageVisibilityHiddenSystem},
		},
	}
	units := buildSearchUnits("conv-1", session)
	for _, u := range units {
		assert.NotContains(t, u.text, "hidden")
	}
	assert.Greater(t, len(units), 0)
}

func TestBuildSearchUnitsIndexesToolCalls(t *testing.T) {
	t.Parallel()
	session := sessionFull{
		Messages: []message{
			{
				Role: conv.RoleAssistant,
				Text: "response",
				ToolCalls: []toolCall{
					{Name: "Read", Summary: "Read /tmp/file.go"},
				},
			},
		},
	}
	units := buildSearchUnits("conv-1", session)
	found := false
	for _, u := range units {
		if strings.Contains(u.text, "Read /tmp/file.go") {
			found = true
		}
	}
	assert.True(t, found, "tool call summary should be indexed")
}

func TestBuildSearchUnitsIndexesPlans(t *testing.T) {
	t.Parallel()
	session := sessionFull{
		Messages: []message{
			{
				Role: conv.RoleAssistant,
				Text: "thinking",
				Plans: []plan{
					{Content: "plan content here"},
				},
			},
		},
	}
	units := buildSearchUnits("conv-1", session)
	found := false
	for _, u := range units {
		if strings.Contains(u.text, "plan content here") {
			found = true
		}
	}
	assert.True(t, found, "plan content should be indexed")
}

func TestBuildSearchUnitsEmptySession(t *testing.T) {
	t.Parallel()
	units := buildSearchUnits("conv-1", sessionFull{})
	assert.Empty(t, units)
}

func TestYieldSessionSearchUnitsMatchesBuildSearchUnits(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		Messages: []message{
			{
				Role:       conv.RoleUser,
				Text:       "intro line\n\nsecond line",
				Visibility: conv.MessageVisibilityVisible,
			},
			{
				Role:      conv.RoleAssistant,
				Text:      strings.Repeat("chunk ", 40),
				ToolCalls: []toolCall{{Name: "Read", Summary: "Read /tmp/main.go"}},
				Plans:     []plan{{Content: "ship it"}},
			},
			{
				Role:       conv.RoleSystem,
				Text:       "hidden system",
				Visibility: conv.MessageVisibilityHiddenSystem,
			},
		},
	}

	want := buildSearchUnits("conv-1", session)
	got := make([]searchUnit, 0, len(want))
	yieldSessionSearchUnits(session, func(ordinal int, text string) bool {
		got = append(got, searchUnit{
			conversationID: "conv-1",
			ordinal:        ordinal,
			text:           text,
		})
		return true
	})

	assert.Equal(t, want, got)
}

func chunkSearchFixture(prefix string) string {
	var b strings.Builder
	for i := range 80 {
		fmt.Fprintf(&b, "%s-%03d|", prefix, i)
	}
	return b.String()
}

func reassembleChunkedText(chunks []string) string {
	if len(chunks) == 0 {
		return ""
	}

	merged := chunks[0]
	for _, chunk := range chunks[1:] {
		merged = appendChunkWithOverlap(merged, chunk)
	}
	return merged
}

func appendChunkWithOverlap(left, right string) string {
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	maxOverlap := min(len(leftRunes), len(rightRunes))
	for overlap := maxOverlap; overlap > 0; overlap-- {
		if slices.Equal(leftRunes[len(leftRunes)-overlap:], rightRunes[:overlap]) {
			return left + string(rightRunes[overlap:])
		}
	}
	return left + right
}
