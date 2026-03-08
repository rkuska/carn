package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindQueryMatchIndices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		text  string
		query string
		want  []int
	}{
		{
			name:  "empty query returns nil",
			text:  "hello world",
			query: "",
			want:  nil,
		},
		{
			name:  "empty text returns nil",
			text:  "",
			query: "hello",
			want:  nil,
		},
		{
			name:  "no match returns nil",
			text:  "hello world",
			query: "xyz",
			want:  nil,
		},
		{
			name:  "single word match at start",
			text:  "hello world",
			query: "hello",
			want:  []int{0, 1, 2, 3, 4},
		},
		{
			name:  "single word match in middle",
			text:  "say hello world",
			query: "hello",
			want:  []int{4, 5, 6, 7, 8},
		},
		{
			name:  "case insensitive match",
			text:  "Hello World",
			query: "hello",
			want:  []int{0, 1, 2, 3, 4},
		},
		{
			name:  "multiple occurrences of single word",
			text:  "ab ab ab",
			query: "ab",
			want:  []int{0, 1, 3, 4, 6, 7},
		},
		{
			name:  "overlapping matches only finds non-overlapping",
			text:  "aaa",
			query: "aa",
			want:  []int{0, 1},
		},
		{
			name:  "unicode text",
			text:  "hello wörld hello",
			query: "wörld",
			want:  []int{6, 7, 8, 9, 10},
		},
		{
			name:  "query longer than text returns nil",
			text:  "hi",
			query: "hello",
			want:  nil,
		},
		{
			name:  "multi-word query highlights each word",
			text:  "the matching of strings is fun",
			query: "matching strings",
			want:  []int{4, 5, 6, 7, 8, 9, 10, 11, 16, 17, 18, 19, 20, 21, 22},
		},
		{
			name:  "multi-word query with only partial word match",
			text:  "only matching here",
			query: "matching strings",
			want:  []int{5, 6, 7, 8, 9, 10, 11, 12},
		},
		{
			name:  "multi-word query no words match",
			text:  "completely unrelated text",
			query: "matching strings",
			want:  nil,
		},
		{
			name:  "multi-word deduplicates overlapping indices",
			text:  "ab cd ab",
			query: "ab ab",
			want:  []int{0, 1, 6, 7},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := findQueryMatchIndices(tc.text, tc.query)
			assert.Equal(t, tc.want, got)
		})
	}
}
