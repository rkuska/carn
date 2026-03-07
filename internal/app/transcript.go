package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/muesli/reflow/wordwrap"
)

type transcriptOptions struct {
	showThinking    bool
	showTools       bool
	showToolResults bool
	hideSidechain   bool
}

type segmentKind int

const (
	segmentMarkdown   segmentKind = iota
	segmentToolResult segmentKind = iota
	segmentRoleHeader segmentKind = iota
	segmentThinking   segmentKind = iota
	segmentToolCall   segmentKind = iota
)

type transcriptSegment struct {
	kind   segmentKind
	text   string     // for markdown, thinking, and tool call segments
	result toolResult // for tool result segments
	role   role       // for role header segments
}

// renderTranscriptSegmented walks messages and produces segments.
// Markdown text accumulates into markdown segments; each tool result
// becomes its own segmentToolResult. When a tool result is encountered,
// the accumulated markdown is flushed first.
func renderTranscriptSegmented(session sessionFull, opts transcriptOptions) []transcriptSegment {
	var segments []transcriptSegment
	var md strings.Builder

	flush := func() {
		if md.Len() > 0 {
			segments = append(segments, transcriptSegment{kind: segmentMarkdown, text: md.String()})
			md.Reset()
		}
	}

	for _, msg := range session.messages {
		if opts.hideSidechain && msg.isSidechain {
			continue
		}

		if msg.isAgentDivider {
			md.WriteString("---\n### Subagent\n")
			md.WriteString(msg.text)
			md.WriteString("\n---\n\n")
			continue
		}

		switch msg.role {
		case roleUser:
			userText := msg.text
			if isSystemInterrupt(userText) {
				userText = ""
			}
			hasContent := userText != "" || (opts.showToolResults && len(msg.toolResults) > 0)
			if !hasContent {
				continue
			}

			flush()
			segments = append(segments, transcriptSegment{kind: segmentRoleHeader, role: roleUser})
			if userText != "" {
				md.WriteString(userText)
				md.WriteString("\n\n")
			}
			if opts.showToolResults && len(msg.toolResults) > 0 {
				flush()
				for _, tr := range msg.toolResults {
					segments = append(segments, transcriptSegment{kind: segmentToolResult, result: tr})
				}
				md.WriteString("\n")
			}

		case roleAssistant:
			hasContent := msg.text != "" ||
				(opts.showThinking && msg.thinking != "") ||
				(opts.showTools && len(msg.toolCalls) > 0)
			if !hasContent {
				continue
			}

			flush()
			segments = append(segments, transcriptSegment{kind: segmentRoleHeader, role: roleAssistant})
			if opts.showThinking && msg.thinking != "" {
				flush()
				segments = append(segments, transcriptSegment{kind: segmentThinking, text: msg.thinking})
			}
			if msg.text != "" {
				md.WriteString(msg.text)
				md.WriteString("\n\n")
			}
			if opts.showTools && len(msg.toolCalls) > 0 {
				flush()
				for _, tc := range msg.toolCalls {
					segments = append(segments, transcriptSegment{kind: segmentToolCall, text: formatToolCall(tc)})
				}
				md.WriteString("\n")
			}
		}
	}

	flush()
	return segments
}

// flattenSegments produces a plain-text transcript from segments.
func flattenSegments(segments []transcriptSegment) string {
	var sb strings.Builder
	for _, seg := range segments {
		switch seg.kind {
		case segmentMarkdown:
			sb.WriteString(seg.text)
		case segmentToolResult:
			sb.WriteString(formatToolResult(seg.result))
			sb.WriteString("\n")
		case segmentRoleHeader:
			switch seg.role {
			case roleUser:
				sb.WriteString("## You\n\n")
			case roleAssistant:
				sb.WriteString("## Assistant\n\n")
			}
		case segmentThinking:
			sb.WriteString("*Thinking:*\n")
			sb.WriteString(seg.text)
			sb.WriteString("\n\n")
		case segmentToolCall:
			sb.WriteString(seg.text)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// renderTranscript produces a clean text transcript from a parsed session.
func renderTranscript(session sessionFull, opts transcriptOptions) string {
	return flattenSegments(renderTranscriptSegmented(session, opts))
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

	header := "Result"
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

// renderPreviewHeader renders metadata lines for the preview pane.
// Empty fields are omitted; entire lines are omitted if all fields are empty.
func renderPreviewHeader(meta sessionMeta) string {
	sep := styleMetaLabel.Render("  ")

	var lines []string

	// Line 1: model  duration  msgs  tokens
	var parts1 []string
	if meta.model != "" {
		parts1 = append(parts1, styleMetaValue.Render(meta.model))
	}
	if d := meta.duration(); d > 0 {
		parts1 = append(parts1, styleMetaValue.Render(formatDuration(d)))
	}
	parts1 = append(parts1, styleMetaValue.Render(fmt.Sprintf("%d msgs", meta.messageCount)))
	if total := meta.totalUsage.totalTokens(); total > 0 {
		parts1 = append(parts1, styleMetaValue.Render(fmt.Sprintf("%dk", total/1000)))
	}
	lines = append(lines, strings.Join(parts1, sep))

	// Line 2: branch  tools
	var parts2 []string
	if meta.gitBranch != "" {
		parts2 = append(parts2, styleMetaValue.Render(meta.gitBranch))
	}
	if tc := formatToolCounts(meta.toolCounts); tc != "" {
		parts2 = append(parts2, styleMetaValue.Render(tc))
	}
	if len(parts2) > 0 {
		lines = append(lines, strings.Join(parts2, sep))
	}

	// Line 3: started <timestamp>  last <timestamp>
	var parts3 []string
	const tsFmt = "2006-01-02 15:04"
	if !meta.timestamp.IsZero() {
		started := styleMetaLabel.Render("started ") + styleMetaValue.Render(meta.timestamp.Format(tsFmt))
		parts3 = append(parts3, started)
	}
	if !meta.lastTimestamp.IsZero() {
		last := styleMetaLabel.Render("last ") + styleMetaValue.Render(meta.lastTimestamp.Format(tsFmt))
		parts3 = append(parts3, last)
	}
	if len(parts3) > 0 {
		lines = append(lines, strings.Join(parts3, sep))
	}

	return strings.Join(lines, "\n") + "\n"
}

// firstUserMessage extracts the text of the first non-interrupt, non-empty
// user message from the session.
func firstUserMessage(messages []message) string {
	for _, msg := range messages {
		if msg.role != roleUser {
			continue
		}
		if msg.isAgentDivider {
			continue
		}
		if msg.text == "" || isSystemInterrupt(msg.text) {
			continue
		}
		return msg.text
	}
	return ""
}

// renderInitialPrompt renders the initial user prompt in a quoted style
// with a ▎ left border in colorPrimary.
func renderInitialPrompt(prompt string, width int) string {
	if prompt == "" {
		return ""
	}

	promptStyle := lipgloss.NewStyle().Foreground(colorPrimary)
	wrapWidth := max(width-2, 10) // account for "▎ " prefix
	wrapped := wrapText(prompt, wrapWidth)

	var sb strings.Builder
	sb.WriteString("\n")
	for line := range strings.SplitSeq(wrapped, "\n") {
		sb.WriteString(promptStyle.Render("▎ " + line))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

// renderPreview renders a short preview of a session (first few exchanges).
// width controls word wrapping; values <= 0 disable wrapping.
func renderPreview(session sessionFull, maxMessages int, width int) string {
	var sb strings.Builder

	sb.WriteString(renderPreviewHeader(session.meta))

	prompt := firstUserMessage(session.messages)
	sb.WriteString(renderInitialPrompt(prompt, width))

	count := 0
	skippedFirstUser := false

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
			if msg.text == "" || isSystemInterrupt(msg.text) {
				continue
			}
			if !skippedFirstUser {
				skippedFirstUser = true
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
