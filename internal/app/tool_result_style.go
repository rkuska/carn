package app

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	conv "github.com/rkuska/carn/internal/conversation"
)

const toolResultPrefixW = 2 // ▎ border (1 cell) + space (1 cell)

// renderStyledToolResult renders a tool result with lipgloss styling:
// colored header badge, dark background content area with left border,
// and per-line diff coloring for structured patches.
func renderStyledToolResult(tr conv.ToolResult, width int) string {
	var sb strings.Builder

	// Choose badge color based on error status
	badgeBg := colorPrimary
	borderColor := colorPrimary
	if tr.IsError {
		badgeBg = colorDiffRemove
		borderColor = colorDiffRemove
	}

	badgeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorStatusFg).
		Background(badgeBg).
		Padding(0, 1)

	name := "Result"
	if tr.ToolName != "" {
		name = tr.ToolName
	}
	sb.WriteString(badgeStyle.Render(name))

	// Build content lines early so we can show line count
	contentLines := buildContentLines(tr)

	summaryStyle := lipgloss.NewStyle().Foreground(colorSecondary)
	summary := tr.ToolSummary
	if summary == "" && tr.ToolName == "" {
		summary = contentFallbackSummary(tr.Content)
	}
	if summary != "" {
		sb.WriteString(" ")
		sb.WriteString(summaryStyle.Render(summary))
	}
	if len(contentLines) > 0 {
		lineCount := fmt.Sprintf(" %d lines", len(contentLines))
		sb.WriteString(summaryStyle.Render(lineCount))
	}
	sb.WriteString("\n")

	// Content area
	if len(contentLines) > 0 {
		renderContentArea(&sb, contentLines, tr.StructuredPatch != nil, width, borderColor)
	}

	sb.WriteString("\n")
	return sb.String()
}

// contentFallbackSummary returns the first non-empty trimmed line of content,
// truncated to 80 chars. Used when toolSummary is empty and tool is unresolved.
func contentFallbackSummary(content string) string {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return conv.Truncate(trimmed, 80)
		}
	}
	return ""
}

func buildContentLines(tr conv.ToolResult) []string {
	if len(tr.StructuredPatch) > 0 {
		var lines []string
		for _, hunk := range tr.StructuredPatch {
			lines = append(lines, fmt.Sprintf("@@ -%d,%d +%d,%d @@",
				hunk.OldStart, hunk.OldLines,
				hunk.NewStart, hunk.NewLines))
			lines = append(lines, hunk.Lines...)
		}
		return lines
	}

	if tr.Content != "" {
		return strings.Split(tr.Content, "\n")
	}

	return nil
}

func renderContentArea(sb *strings.Builder, lines []string, isDiff bool, width int, borderClr color.Color) {
	border := lipgloss.NewStyle().
		Foreground(borderClr).
		Render("▎")

	bgStyle := lipgloss.NewStyle().
		Background(colorToolBg).
		Foreground(colorFgOnBg)

	addStyle := lipgloss.NewStyle().
		Background(colorToolBg).
		Foreground(colorAccent)

	removeStyle := lipgloss.NewStyle().
		Background(colorToolBg).
		Foreground(colorDiffRemove)

	hunkStyle := lipgloss.NewStyle().
		Background(colorToolBg).
		Foreground(colorDiffHunk)

	contentWidth := max(width-toolResultPrefixW, 1)

	for _, line := range lines {
		style := bgStyle
		if isDiff {
			switch {
			case strings.HasPrefix(line, "+"):
				style = addStyle
			case strings.HasPrefix(line, "-"):
				style = removeStyle
			case strings.HasPrefix(line, "@@"):
				style = hunkStyle
			}
		}

		wrapped := ansi.Hardwrap(line, contentWidth, false)
		for subLine := range strings.SplitSeq(wrapped, "\n") {
			sb.WriteString(border)
			sb.WriteString(" ")
			padded := fitToWidth(subLine, contentWidth)
			sb.WriteString(style.Render(padded))
			sb.WriteString("\n")
		}
	}
}

func fitToWidth(s string, width int) string {
	sw := lipgloss.Width(s)
	if sw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-sw)
}
