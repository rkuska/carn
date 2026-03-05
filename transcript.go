package main

import (
	"fmt"
	"strings"

	"github.com/muesli/reflow/wordwrap"
)

type transcriptOptions struct {
	showThinking bool
	showTools    bool
}

// renderTranscript produces a clean text transcript from a parsed session.
func renderTranscript(session sessionFull, opts transcriptOptions) string {
	var sb strings.Builder

	for _, msg := range session.messages {
		switch msg.role {
		case roleUser:
			renderUserMessage(&sb, msg)
		case roleAssistant:
			renderAssistantMessage(&sb, msg, opts)
		}
	}

	return sb.String()
}

func renderUserMessage(sb *strings.Builder, msg message) {
	sb.WriteString("## You\n\n")
	sb.WriteString(msg.text)
	sb.WriteString("\n\n")
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

func formatToolCall(tc toolCall) string {
	if tc.summary != "" {
		return fmt.Sprintf("[%s: %s]", tc.name, tc.summary)
	}
	return fmt.Sprintf("[%s]", tc.name)
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

		switch msg.role {
		case roleUser:
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
