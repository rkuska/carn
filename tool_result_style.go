package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// renderStyledToolResult renders a tool result with lipgloss styling:
// colored header badge, dark background content area with left border,
// and per-line diff coloring for structured patches.
func renderStyledToolResult(tr toolResult, width int) string {
	var sb strings.Builder

	// Header badge: bold white on purple background
	badgeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(colorPrimary).
		Padding(0, 1)

	name := contentTypeToolResult
	if tr.toolName != "" {
		name = tr.toolName
	}
	sb.WriteString(badgeStyle.Render(name))

	if tr.toolSummary != "" {
		summaryStyle := lipgloss.NewStyle().Foreground(colorSecondary)
		sb.WriteString(" ")
		sb.WriteString(summaryStyle.Render(tr.toolSummary))
	}
	sb.WriteString("\n")

	// Content area
	contentLines := buildContentLines(tr)
	if len(contentLines) > 0 {
		renderContentArea(&sb, contentLines, tr.structuredPatch != nil, width)
	}

	sb.WriteString("\n")
	return sb.String()
}

func buildContentLines(tr toolResult) []string {
	if len(tr.structuredPatch) > 0 {
		var lines []string
		for _, hunk := range tr.structuredPatch {
			lines = append(lines, fmt.Sprintf("@@ -%d,%d +%d,%d @@",
				hunk.oldStart, hunk.oldLines,
				hunk.newStart, hunk.newLines))
			lines = append(lines, hunk.lines...)
		}
		return lines
	}

	if tr.content != "" {
		return strings.Split(tr.content, "\n")
	}

	return nil
}

func renderContentArea(sb *strings.Builder, lines []string, isDiff bool, width int) {
	border := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Render("▎")

	bgStyle := lipgloss.NewStyle().
		Background(colorToolBg).
		Foreground(lipgloss.Color("252"))

	addStyle := lipgloss.NewStyle().
		Background(colorToolBg).
		Foreground(colorAccent)

	removeStyle := lipgloss.NewStyle().
		Background(colorToolBg).
		Foreground(colorDiffRemove)

	hunkStyle := lipgloss.NewStyle().
		Background(colorToolBg).
		Foreground(colorDiffHunk)

	// Content width: total width minus border char (2 cells) minus padding (1 space)
	contentWidth := max(width-3, 1)

	for _, line := range lines {
		sb.WriteString(border)
		sb.WriteString(" ")

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

		padded := padRight(line, contentWidth)
		sb.WriteString(style.Render(padded))
		sb.WriteString("\n")
	}
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
