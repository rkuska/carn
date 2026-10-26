package elements

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContextLineMap(t *testing.T) {
	t.Parallel()

	content := "header\nheader2\n\nuser\nasst\nasst2\n"

	tests := []struct {
		name    string
		offsets []SegmentOffset
		want    ContextLineMap
	}{
		{name: "no segments", offsets: nil, want: nil},
		{
			name: "one anchor per segment line",
			offsets: []SegmentOffset{
				{Offset: 16, MsgIndex: 0},
				{Offset: 21, MsgIndex: 1},
				{Offset: 26, MsgIndex: 2},
			},
			want: ContextLineMap{{Line: 3, MsgIndex: 0}, {Line: 4, MsgIndex: 1}, {Line: 5, MsgIndex: 2}},
		},
		{
			name:    "segments on one line collapse into the last",
			offsets: []SegmentOffset{{Offset: 16, MsgIndex: 0}, {Offset: 16, MsgIndex: 1}, {Offset: 21, MsgIndex: 2}},
			want:    ContextLineMap{{Line: 3, MsgIndex: 1}, {Line: 4, MsgIndex: 2}},
		},
		{
			name:    "offsets beyond the content clamp to its end",
			offsets: []SegmentOffset{{Offset: 0, MsgIndex: 0}, {Offset: 999, MsgIndex: 1}},
			want:    ContextLineMap{{Line: 0, MsgIndex: 0}, {Line: 6, MsgIndex: 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, NewContextLineMap(content, tt.offsets))
		})
	}
}

func TestContextLineMapMessageAt(t *testing.T) {
	t.Parallel()

	lines := ContextLineMap{{Line: 4, MsgIndex: 0}, {Line: 9, MsgIndex: 3}, {Line: 15, MsgIndex: 7}}

	tests := []struct {
		name string
		line int
		want int
	}{
		{name: "before the first message", line: 0, want: -1},
		{name: "negative", line: -3, want: -1},
		{name: "on the first anchor", line: 4, want: 0},
		{name: "inside a segment", line: 8, want: 0},
		{name: "on the second anchor", line: 9, want: 3},
		{name: "after the last anchor", line: 400, want: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, lines.MessageAt(tt.line))
		})
	}
	assert.Equal(t, -1, ContextLineMap(nil).MessageAt(3))
}

func TestContextLineMapLineOf(t *testing.T) {
	t.Parallel()

	lines := ContextLineMap{{Line: 4, MsgIndex: 0}, {Line: 9, MsgIndex: 3}, {Line: 15, MsgIndex: 7}}

	tests := []struct {
		name     string
		msgIndex int
		wantLine int
		wantOK   bool
	}{
		{name: "exact message", msgIndex: 3, wantLine: 9, wantOK: true},
		{name: "hidden message resolves to the next visible one", msgIndex: 4, wantLine: 15, wantOK: true},
		{name: "before the first", msgIndex: -2, wantLine: 4, wantOK: true},
		{name: "after the last", msgIndex: 8, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			line, ok := lines.LineOf(tt.msgIndex)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantLine, line)
		})
	}
}

func TestNextAndPrevLine(t *testing.T) {
	t.Parallel()

	lines := []int{4, 9, 15}

	next, ok := NextLine(lines, 4)
	assert.True(t, ok)
	assert.Equal(t, 9, next)
	next, ok = NextLine(lines, 3)
	assert.True(t, ok)
	assert.Equal(t, 4, next)
	_, ok = NextLine(lines, 15)
	assert.False(t, ok)
	_, ok = NextLine(nil, 0)
	assert.False(t, ok)

	prev, ok := PrevLine(lines, 15)
	assert.True(t, ok)
	assert.Equal(t, 9, prev)
	prev, ok = PrevLine(lines, 400)
	assert.True(t, ok)
	assert.Equal(t, 15, prev)
	_, ok = PrevLine(lines, 4)
	assert.False(t, ok)
	_, ok = PrevLine(nil, 4)
	assert.False(t, ok)
}

func TestCollapseLeadingBlankLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no leading blank line", input: "text\nmore\n", want: "text\nmore\n"},
		{name: "one leading blank line", input: "\ntext\n", want: "\ntext\n"},
		{name: "two leading blank lines", input: "\n\ntext\n", want: "\ntext\n"},
		{name: "styled blank lines", input: "\x1b[1m \x1b[0m\n  \n\x1b[1mtext\x1b[0m\n", want: "  \n\x1b[1mtext\x1b[0m\n"},
		{name: "only blank lines", input: "\n\n\n", want: "\n"},
		{name: "empty", input: "", want: ""},
		{name: "blank lines inside are kept", input: "\ntext\n\n\nmore\n", want: "\ntext\n\n\nmore\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, CollapseLeadingBlankLines(tt.input))
		})
	}
}

func TestContextLineMapScrollTarget(t *testing.T) {
	t.Parallel()

	lines := ContextLineMap{{Line: 4, MsgIndex: 0}, {Line: 9, MsgIndex: 3}, {Line: 15, MsgIndex: 7}}

	tests := []struct {
		name     string
		msg      int
		delta    int
		total    int
		wantLine int
		wantOK   bool
	}{
		{name: "same message, same offset", msg: 3, delta: 2, total: 30, wantLine: 11, wantOK: true},
		{name: "offset clamped inside the message", msg: 3, delta: 40, total: 30, wantLine: 14, wantOK: true},
		{name: "last message clamps to the end", msg: 7, delta: 99, total: 20, wantLine: 19, wantOK: true},
		{name: "hidden message starts the next visible one", msg: 5, delta: 3, total: 30, wantLine: 15, wantOK: true},
		{name: "message after the last", msg: 8, delta: 0, total: 30, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			line, ok := lines.ScrollTarget(tt.msg, tt.delta, tt.total)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantLine, line)
		})
	}
}

func TestNotableLines(t *testing.T) {
	t.Parallel()

	series := dockSeries() // steps at messages 0..3: baseline, +140k, reset, +6k (all but the first notable)
	lines := ContextLineMap{
		{Line: 2, MsgIndex: 0}, {Line: 8, MsgIndex: 1}, {Line: 14, MsgIndex: 2}, {Line: 20, MsgIndex: 3},
	}

	assert.Equal(t, []int{8, 14, 20}, NotableLines(series, lines))
	assert.Equal(t, []int{14, 20}, NotableLines(series, ContextLineMap{{Line: 14, MsgIndex: 2}, {Line: 20, MsgIndex: 3}}),
		"a step whose message is hidden resolves to the next visible message")
	assert.Empty(t, NotableLines(series, nil))
}
