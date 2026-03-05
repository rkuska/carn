package main

import (
	"fmt"
	"strings"

	"github.com/muesli/reflow/wordwrap"
)

type transcriptOptions struct {
	showThinking    bool
	showTools       bool
	showToolResults bool
	hideSidechain   bool
}

// renderTranscript produces a clean text transcript from a parsed session.
func renderTranscript(session sessionFull, opts transcriptOptions) string {
	var sb strings.Builder

	for _, msg := range session.messages {
		if opts.hideSidechain && msg.isSidechain {
			continue
		}

		if msg.isAgentDivider {
			renderAgentDivider(&sb, msg)
			continue
		}

		switch msg.role {
		case roleUser:
			renderUserMessage(&sb, msg, opts)
		case roleAssistant:
			renderAssistantMessage(&sb, msg, opts)
		}
	}

	return sb.String()
}

func renderUserMessage(sb *strings.Builder, msg message, opts transcriptOptions) {
	hasContent := msg.text != "" || (opts.showToolResults && len(msg.toolResults) > 0)
	if !hasContent {
		return
	}

	sb.WriteString("## You\n\n")
	if msg.text != "" {
		sb.WriteString(msg.text)
		sb.WriteString("\n\n")
	}
	if opts.showToolResults && len(msg.toolResults) > 0 {
		for _, tr := range msg.toolResults {
			sb.WriteString(formatToolResult(tr))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
}

func renderAssistantMessage(sb *strings.Builder, msg message, opts transcriptOptions) {
	hasContent := msg.text != "" ||
		(opts.showThinking && msg.thinking != "") ||
		(opts.showTools && len(msg.toolCalls) > 0)
	if !hasContent {
		return
	}

	sb.WriteString("## Assistant\n\n")

	if opts.showThinking && msg.thinking != "" {
		sb.WriteString("*Thinking:*\n")
		sb.WriteString(msg.thinking)
		sb.WriteString("\n\n")
	}

	if msg.text != "" {
		sb.WriteString(msg.text)
		sb.WriteString("\n\n")
	}

	if opts.showTools && len(msg.toolCalls) > 0 {
		for _, tc := range msg.toolCalls {
			sb.WriteString(formatToolCall(tc))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
}

func renderAgentDivider(sb *strings.Builder, msg message) {
	sb.WriteString("---\n### Subagent\n")
	sb.WriteString(msg.text)
	sb.WriteString("\n---\n\n")
}

func formatToolCall(tc toolCall) string {
	if tc.summary != "" {
		return fmt.Sprintf("[%s: %s]", tc.name, tc.summary)
	}
	return fmt.Sprintf("[%s]", tc.name)
}

// formatToolResult renders a tool result with its resolved name and content.
func formatToolResult(tr toolResult) string {
	var sb strings.Builder

	header := "tool_result"
	if tr.toolName != "" {
		header = tr.toolName
	}

	if tr.toolSummary != "" {
		fmt.Fprintf(&sb, "**%s**: `%s`\n", header, tr.toolSummary)
	} else {
		fmt.Fprintf(&sb, "**%s**\n", header)
	}

	if len(tr.structuredPatch) > 0 {
		sb.WriteString("```diff\n")
		for _, hunk := range tr.structuredPatch {
			fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n",
				hunk.oldStart, hunk.oldLines,
				hunk.newStart, hunk.newLines)
			for _, line := range hunk.lines {
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
		sb.WriteString("```")
	} else if tr.content != "" {
		sb.WriteString("```\n")
		sb.WriteString(tr.content)
		if !strings.HasSuffix(tr.content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("```")
	}

	return sb.String()
}

// renderPreview renders a short preview of a session (first few exchanges).
// width controls word wrapping; values <= 0 disable wrapping.
func renderPreview(session sessionFull, maxMessages int, width int) string {
	var sb strings.Builder
	count := 0

	for _, msg := range session.messages {
		if count >= maxMessages {
			sb.WriteString("...\n")
			break
		}

		if msg.isAgentDivider {
			sb.WriteString("--- Subagent ---\n")
			sb.WriteString(wrapText(msg.text, width))
			sb.WriteString("\n\n")
			count++
			continue
		}

		switch msg.role {
		case roleUser:
			if msg.text == "" {
				continue
			}
			sb.WriteString("▶ You\n")
			sb.WriteString(wrapText(msg.text, width))
			sb.WriteString("\n\n")
		case roleAssistant:
			sb.WriteString("◀ Assistant\n")
			text := msg.text
			if text == "" && len(msg.toolCalls) > 0 {
				text = formatToolCall(msg.toolCalls[0])
			}
			sb.WriteString(wrapText(text, width))
			sb.WriteString("\n\n")
		}
		count++
	}

	return sb.String()
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	return wordwrap.String(text, width)
}
