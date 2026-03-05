package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderStyledToolResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tr       toolResult
		width    int
		contains []string
	}{
		{
			name: "header contains tool name",
			tr: toolResult{
				toolName:    "Read",
				toolSummary: "/path/file.go",
				content:     "package main",
			},
			width:    80,
			contains: []string{"Read", "/path/file.go"},
		},
		{
			name: "diff lines present in output",
			tr: toolResult{
				toolName:    "Edit",
				toolSummary: "/file.go",
				structuredPatch: []diffHunk{
					{
						oldStart: 1, oldLines: 2, newStart: 1, newLines: 3,
						lines: []string{" ctx", "-old", "+new1", "+new2"},
					},
				},
			},
			width:    80,
			contains: []string{"Edit", "-old", "+new1", "+new2", "@@"},
		},
		{
			name: "plain content when no patch",
			tr: toolResult{
				toolName: "Bash",
				content:  "ls output here",
			},
			width:    80,
			contains: []string{"Bash", "ls output here"},
		},
		{
			name: "multiple hunks produce multiple @@ markers",
			tr: toolResult{
				toolName: "Edit",
				structuredPatch: []diffHunk{
					{oldStart: 5, oldLines: 2, newStart: 5, newLines: 3, lines: []string{"+a"}},
					{oldStart: 20, oldLines: 1, newStart: 21, newLines: 2, lines: []string{"+b"}},
				},
			},
			width:    80,
			contains: []string{"@@"},
		},
		{
			name: "fallback header for unnamed tool",
			tr: toolResult{
				toolUseID: "toolu_xyz",
				content:   "output",
			},
			width:    80,
			contains: []string{"tool_result", "output"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderStyledToolResult(tt.tr, tt.width)
			stripped := ansi.Strip(got)

			for _, want := range tt.contains {
				if !strings.Contains(stripped, want) {
					t.Errorf("output missing %q\nstripped:\n%s\nraw:\n%s", want, stripped, got)
				}
			}
		})
	}
}

func TestRenderStyledToolResultMultipleHunksCount(t *testing.T) {
	t.Parallel()

	tr := toolResult{
		toolName: "Edit",
		structuredPatch: []diffHunk{
			{oldStart: 5, oldLines: 2, newStart: 5, newLines: 3, lines: []string{"+a"}},
			{oldStart: 20, oldLines: 1, newStart: 21, newLines: 2, lines: []string{"+b"}},
		},
	}
	got := renderStyledToolResult(tr, 80)
	stripped := ansi.Strip(got)

	count := strings.Count(stripped, "@@")
	// 2 hunks × 2 @@ each = 4
	if count != 4 {
		t.Errorf("expected 4 @@ markers, got %d\n%s", count, stripped)
	}
}
