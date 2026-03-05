package main

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
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
			if gotWidth != tt.wantWidth {
				t.Errorf("fitToWidth(%q, %d): visual width = %d, want %d\ngot: %q",
					tt.input, tt.width, gotWidth, tt.wantWidth, got)
			}
		})
	}
}

func TestRenderContentAreaConsistentWidth(t *testing.T) {
	t.Parallel()

	initPalette(true)

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
			renderContentArea(&sb, tt.lines, false, tt.width)
			output := sb.String()

			rendered := strings.Split(strings.TrimRight(output, "\n"), "\n")
			widths := make([]int, len(rendered))
			for i, line := range rendered {
				widths[i] = lipgloss.Width(line)
			}

			for i := 1; i < len(widths); i++ {
				if widths[i] != widths[0] {
					t.Errorf("line %d width %d != line 0 width %d\nwidths: %v",
						i, widths[i], widths[0], widths)
					break
				}
			}
		})
	}
}

func TestRenderContentAreaWrapsLongLines(t *testing.T) {
	t.Parallel()

	initPalette(true)

	width := 30
	longLine := strings.Repeat("a", 50)
	var sb strings.Builder
	renderContentArea(&sb, []string{longLine}, false, width)

	output := sb.String()
	rendered := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(rendered) < 2 {
		t.Errorf("expected long line to wrap into multiple lines, got %d line(s)", len(rendered))
	}

	contentWidth := width - 2 // border (1) + space (1)
	for i, line := range rendered {
		stripped := ansi.Strip(line)
		w := lipgloss.Width(stripped)
		// Each line should be border(1) + space(1) + contentWidth
		expected := contentWidth + 2
		if w != expected {
			t.Errorf("wrapped line %d: width %d, want %d\nline: %q", i, w, expected, stripped)
		}
	}
}
