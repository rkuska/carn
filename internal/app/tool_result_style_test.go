package app

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderStyledToolResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tr       conv.ToolResult
		width    int
		contains []string
	}{
		{
			name: "header contains tool name",
			tr: conv.ToolResult{
				ToolName:    "Read",
				ToolSummary: "/path/file.go",
				Content:     "package main",
			},
			width:    80,
			contains: []string{"Read", "/path/file.go"},
		},
		{
			name: "diff lines present in output",
			tr: conv.ToolResult{
				ToolName:    "Edit",
				ToolSummary: "/file.go",
				StructuredPatch: []conv.DiffHunk{
					{
						OldStart: 1, OldLines: 2, NewStart: 1, NewLines: 3,
						Lines: []string{" ctx", "-old", "+new1", "+new2"},
					},
				},
			},
			width:    80,
			contains: []string{"Edit", "-old", "+new1", "+new2", "@@"},
		},
		{
			name: "plain content when no patch",
			tr: conv.ToolResult{
				ToolName: "Bash",
				Content:  "ls output here",
			},
			width:    80,
			contains: []string{"Bash", "ls output here"},
		},
		{
			name: "multiple hunks produce multiple @@ markers",
			tr: conv.ToolResult{
				ToolName: "Edit",
				StructuredPatch: []conv.DiffHunk{
					{OldStart: 5, OldLines: 2, NewStart: 5, NewLines: 3, Lines: []string{"+a"}},
					{OldStart: 20, OldLines: 1, NewStart: 21, NewLines: 2, Lines: []string{"+b"}},
				},
			},
			width:    80,
			contains: []string{"@@"},
		},
		{
			name: "fallback header for unnamed tool",
			tr: conv.ToolResult{
				Content: "output",
			},
			width:    80,
			contains: []string{"Result", "output"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderStyledToolResult(tt.tr, tt.width)
			stripped := ansi.Strip(got)

			assertContainsAll(t, stripped, tt.contains...)
		})
	}
}

func TestRenderStyledToolResultMultipleHunksCount(t *testing.T) {
	t.Parallel()

	tr := conv.ToolResult{
		ToolName: "Edit",
		StructuredPatch: []conv.DiffHunk{
			{OldStart: 5, OldLines: 2, NewStart: 5, NewLines: 3, Lines: []string{"+a"}},
			{OldStart: 20, OldLines: 1, NewStart: 21, NewLines: 2, Lines: []string{"+b"}},
		},
	}
	got := renderStyledToolResult(tr, 80)
	stripped := ansi.Strip(got)

	count := strings.Count(stripped, "@@")
	// 2 hunks × 2 @@ each = 4
	assert.Equal(t, 4, count)
}

func TestFitToWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		width     int
		wantWidth int
	}{
		{
			name:      "ASCII shorter than width pads with spaces",
			input:     "hello",
			width:     10,
			wantWidth: 10,
		},
		{
			name:      "ASCII exact width unchanged",
			input:     "hello",
			width:     5,
			wantWidth: 5,
		},
		{
			name:      "multi-byte arrow pads correctly",
			input:     "350→  parentID",
			width:     20,
			wantWidth: 20,
		},
		{
			name:      "empty string pads to full width",
			input:     "",
			width:     5,
			wantWidth: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := fitToWidth(tt.input, tt.width)
			gotWidth := lipgloss.Width(got)
			assert.Equal(t, tt.wantWidth, gotWidth)
		})
	}
}

func TestRenderContentAreaConsistentWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lines []string
		width int
	}{
		{
			name:  "lines with multi-byte characters align",
			lines: []string{"hello", "350→    parentID", "normal line"},
			width: 40,
		},
		{
			name:  "long lines wrapped to same width",
			lines: []string{"short", strings.Repeat("x", 100)},
			width: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var sb strings.Builder
			renderContentArea(&sb, tt.lines, false, tt.width, colorPrimary)
			output := sb.String()

			rendered := strings.Split(strings.TrimRight(output, "\n"), "\n")
			widths := make([]int, len(rendered))
			for i, line := range rendered {
				widths[i] = lipgloss.Width(line)
			}

			for i := 1; i < len(widths); i++ {
				assert.Equal(t, widths[0], widths[i])
			}
		})
	}
}

func TestRenderContentAreaWrapsLongLines(t *testing.T) {
	t.Parallel()

	width := 30
	longLine := strings.Repeat("a", 50)
	var sb strings.Builder
	renderContentArea(&sb, []string{longLine}, false, width, colorPrimary)

	output := sb.String()
	rendered := strings.Split(strings.TrimRight(output, "\n"), "\n")

	require.GreaterOrEqual(t, len(rendered), 2)

	contentWidth := width - 2 // border (1) + space (1)
	for _, line := range rendered {
		stripped := ansi.Strip(line)
		w := lipgloss.Width(stripped)
		// Each line should be border(1) + space(1) + contentWidth
		expected := contentWidth + 2
		assert.Equal(t, expected, w)
	}
}

func TestRenderStyledToolResultErrorStyling(t *testing.T) {
	t.Parallel()

	t.Run("error result uses different ANSI styling than success", func(t *testing.T) {
		t.Parallel()
		errTR := conv.ToolResult{
			ToolName: "Bash",
			Content:  "command failed",
			IsError:  true,
		}
		okTR := conv.ToolResult{
			ToolName: "Bash",
			Content:  "command succeeded",
			IsError:  false,
		}
		errOutput := renderStyledToolResult(errTR, 80)
		okOutput := renderStyledToolResult(okTR, 80)
		// Both should contain the tool name
		assert.Contains(t, ansi.Strip(errOutput), "Bash")
		// The raw ANSI output should differ (different colors)
		assert.NotEqual(t, errOutput, okOutput)
	})
}

func TestRenderStyledToolResultLineCount(t *testing.T) {
	t.Parallel()

	tr := conv.ToolResult{
		ToolName: "Read",
		Content:  "line1\nline2\nline3",
	}
	got := renderStyledToolResult(tr, 80)
	stripped := ansi.Strip(got)
	assert.Contains(t, stripped, "3 lines")
}

func TestRenderStyledToolResultContentFallbackSummary(t *testing.T) {
	t.Parallel()

	tr := conv.ToolResult{
		Content: "first line of output\nsecond line",
	}
	got := renderStyledToolResult(tr, 80)
	stripped := ansi.Strip(got)
	assert.Contains(t, stripped, "first line of output")
}

func TestContentFallbackSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "first line used",
			content: "hello world\nsecond line",
			want:    "hello world",
		},
		{
			name:    "skips empty lines",
			content: "\n\n  actual content\nmore",
			want:    "actual content",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "long line truncated",
			content: strings.Repeat("x", 100),
			want:    strings.Repeat("x", 80) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := contentFallbackSummary(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}
