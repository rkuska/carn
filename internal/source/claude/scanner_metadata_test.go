package claude

import (
	"bufio"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanMetadataCapturesCountsAndUsage(t *testing.T) {
	t.Parallel()

	baseDir := copyScannerFixtureCorpus(t)
	path := filepath.Join(baseDir, "project-a", "session-with-tools.jsonl")

	result, err := scanMetadataResult(context.Background(), path, project{DisplayName: "demo"})
	require.NoError(t, err)
	meta := result.meta
	assert.Equal(t, "session-tools", meta.ID)
	assert.Equal(t, "tool-runbook", meta.Slug)
	assert.Equal(t, "Inspect the main package and run the tests.", meta.FirstMessage)
	assert.Equal(t, "claude-sonnet-4", meta.Model)
	assert.Equal(t, 1, meta.ToolCounts["Read"])
	assert.Equal(t, 1, meta.ToolCounts["Bash"])
	assert.Equal(t, 440, meta.TotalUsage.TotalTokens())
	assert.Equal(t, 10*time.Second, meta.Duration())
}

func TestJSONLLinesHandlesLargeLines(t *testing.T) {
	t.Parallel()

	reader := strings.NewReader(strings.Repeat("x", jsonlScanBufferSize+32) + "\n")
	lines, err := collectJSONLLines(jsonlLines(bufio.NewReaderSize(reader, 32)))
	require.NoError(t, err)
	require.Len(t, lines, 1)
	assert.Len(t, lines[0], jsonlScanBufferSize+32)
}

func TestExtractType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		line string
		want string
	}{
		{
			name: "user record",
			line: `{"type":"user","message":{"role":"user","content":"hi"}}`,
			want: "user",
		},
		{
			name: "assistant record",
			line: `{"type":"assistant","message":{"role":"assistant","content":"hi"}}`,
			want: "assistant",
		},
		{
			name: "other record",
			line: `{"type":"summary","message":{"role":"assistant","content":"hi"}}`,
			want: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, extractType([]byte(testCase.line)))
		})
	}
}
