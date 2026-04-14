package browser

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/muesli/reflow/wordwrap"

	conv "github.com/rkuska/carn/internal/conversation"
)

func renderPreviewHeader(meta conv.SessionMeta, tsFmt string) string {
	sep := styleMetaLabel.Render("  ")
	lines := []string{strings.Join(renderPreviewPrimaryParts(meta), sep)}

	if secondary := renderPreviewSecondaryParts(meta, sep); secondary != "" {
		lines = append(lines, secondary)
	}
	if timing := renderPreviewTimingParts(meta, sep, tsFmt); timing != "" {
		lines = append(lines, timing)
	}

	return strings.Join(lines, "\n") + "\n"
}

func renderPreviewPrimaryParts(meta conv.SessionMeta) []string {
	parts := make([]string, 0, 4)
	if meta.Model != "" {
		parts = append(parts, styleMetaValue.Render(meta.Model))
	}
	if d := meta.Duration(); d > 0 {
		parts = append(parts, styleMetaValue.Render(conv.FormatDuration(d)))
	}
	parts = append(parts, styleMetaValue.Render(fmt.Sprintf("%d msgs", meta.MessageCount)))
	if total := meta.TotalUsage.TotalTokens(); total > 0 {
		parts = append(parts, styleMetaValue.Render(fmt.Sprintf("%dk", total/1000)))
	}
	return parts
}

func renderPreviewSecondaryParts(meta conv.SessionMeta, sep string) string {
	parts := make([]string, 0, 2)
	if meta.GitBranch != "" {
		parts = append(parts, styleMetaValue.Render(meta.GitBranch))
	}
	if toolCounts := conv.FormatToolCounts(meta.ToolCounts); toolCounts != "" {
		parts = append(parts, styleMetaValue.Render(toolCounts))
	}
	return strings.Join(parts, sep)
}

func renderPreviewTimingParts(meta conv.SessionMeta, sep, tsFmt string) string {
	var parts []string
	if !meta.Timestamp.IsZero() {
		parts = append(parts, styleMetaLabel.Render("started ")+styleMetaValue.Render(meta.Timestamp.Format(tsFmt)))
	}
	if !meta.LastTimestamp.IsZero() {
		parts = append(parts, styleMetaLabel.Render("last ")+styleMetaValue.Render(meta.LastTimestamp.Format(tsFmt)))
	}
	return strings.Join(parts, sep)
}

func firstUserMessage(messages []conv.Message) string {
	for _, msg := range messages {
		if msg.Role != conv.RoleUser || !msg.IsVisible() || msg.IsAgentDivider || msg.Text == "" {
			continue
		}
		return msg.Text
	}
	return ""
}

func renderInitialPrompt(prompt string, width int) string {
	if prompt == "" {
		return ""
	}

	promptStyle := lipgloss.NewStyle().Foreground(colorPrimary)
	wrapped := wrapText(prompt, max(width-2, 10))

	var sb strings.Builder
	sb.WriteString("\n")
	for line := range strings.SplitSeq(wrapped, "\n") {
		sb.WriteString(promptStyle.Render("▎ " + line))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

func renderPreview(session conv.Session, maxMessages int, width int, tsFmt string) string {
	var sb strings.Builder
	sb.WriteString(renderPreviewHeader(session.Meta, tsFmt))
	sb.WriteString(renderInitialPrompt(firstUserMessage(session.Messages), width))

	count := 0
	skippedFirstUser := false
	for _, msg := range session.Messages {
		if count >= maxMessages {
			sb.WriteString("...\n")
			break
		}
		if msg.IsAgentDivider {
			sb.WriteString("--- Subagent ---\n")
			sb.WriteString(wrapText(msg.Text, width))
			sb.WriteString("\n\n")
			count++
			continue
		}

		rendered, skip := renderPreviewMessage(msg, &skippedFirstUser, width)
		if skip {
			continue
		}
		sb.WriteString(rendered)
		count++
	}

	return sb.String()
}

func renderPreviewMessage(msg conv.Message, skippedFirstUser *bool, width int) (string, bool) {
	switch msg.Role {
	case conv.RoleUser:
		if !msg.IsVisible() || msg.Text == "" {
			return "", true
		}
		if !*skippedFirstUser {
			*skippedFirstUser = true
			return "", true
		}
		return "▶ You\n" + wrapText(msg.Text, width) + "\n\n", false
	case conv.RoleAssistant:
		text := msg.Text
		if text == "" && len(msg.ToolCalls) > 0 {
			text = formatToolCall(msg.ToolCalls[0])
		}
		return "◀ Assistant\n" + wrapText(text, width) + "\n\n", false
	case conv.RoleSystem:
		return "", true
	}
	return "", true
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	return wordwrap.String(text, width)
}
