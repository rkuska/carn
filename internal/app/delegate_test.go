package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitItemMatches(t *testing.T) {
	t.Parallel()

	title := "my/project / cheerful-ocean  2024-06-15 14:30"
	desc := "claude-3  25 msgs\n" + archiveMatchesSourceSubtitle
	full := title + "\n" + desc
	matchAt := strings.Index(full, "archive")
	require.GreaterOrEqual(t, matchAt, 0)

	matches := make([]int, len("archive"))
	for i := range matches {
		matches[i] = matchAt + i
	}

	got := splitItemMatches(title, desc, matches)
	assert.Empty(t, got.title)

	descRunes := []rune(desc)
	var highlighted strings.Builder
	for _, idx := range got.desc {
		highlighted.WriteRune(descRunes[idx])
	}

	assert.Equal(t, "archive", highlighted.String())
}
