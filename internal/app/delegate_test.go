package app

import (
	"strings"
	"testing"
)

func TestSplitItemMatches(t *testing.T) {
	t.Parallel()

	title := "my/project / cheerful-ocean  2024-06-15 14:30"
	desc := "claude-3  25 msgs\n" + archiveMatchesSourceSubtitle
	full := title + "\n" + desc
	matchAt := strings.Index(full, "archive")
	if matchAt < 0 {
		t.Fatal("expected test fixture to contain archive")
	}

	matches := make([]int, len("archive"))
	for i := range matches {
		matches[i] = matchAt + i
	}

	got := splitItemMatches(title, desc, matches)
	if len(got.title) != 0 {
		t.Fatalf("title matches = %v, want none", got.title)
	}

	descRunes := []rune(desc)
	var highlighted strings.Builder
	for _, idx := range got.desc {
		highlighted.WriteRune(descRunes[idx])
	}

	if gotWord := highlighted.String(); gotWord != "archive" {
		t.Fatalf("desc highlighted = %q, want archive", gotWord)
	}
}
