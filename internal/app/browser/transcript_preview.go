package browser

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/muesli/reflow/wordwrap"

	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

func renderPreviewHeader(theme *el.Theme, meta conv.SessionMeta, tsFmt string) string {
	sep := theme.StyleMetaLabel.Render("  ")
	lines := []string{strings.Join(renderPreviewPrimaryParts(theme, meta), sep)}

	if secondary := renderPreviewSecondaryParts(theme, meta, sep); secondary != "" {
		lines = append(lines, secondary)
	}
	if timing := renderPreviewTimingParts(theme, meta, sep, tsFmt); timing != "" {
		lines = append(lines, timing)
	}

	return strings.Join(lines, "\n") + "\n"
}

func renderPreviewPrimaryParts(theme *el.Theme, meta conv.SessionMeta) []string {
	parts := make([]string, 0, 4)
	if meta.Model != "" {
		parts = append(parts, theme.StyleMetaValue.Render(meta.Model))
	}
	if d := meta.Duration(); d > 0 {
		parts = append(parts, theme.StyleMetaValue.Render(conv.FormatDuration(d)))
	}
	parts = append(parts, theme.StyleMetaValue.Render(fmt.Sprintf("%d msgs", meta.MessageCount)))
	if total := meta.TotalUsage.TotalTokens(); total > 0 {
		parts = append(parts, theme.StyleMetaValue.Render(fmt.Sprintf("%dk", total/1000)))
	}
	return parts
}

func renderPreviewSecondaryParts(theme *el.Theme, meta conv.SessionMeta, sep string) string {
	parts := make([]string, 0, 2)
	if meta.GitBranch != "" {
		parts = append(parts, theme.StyleMetaValue.Render(meta.GitBranch))
	}
	if toolCounts := conv.FormatToolCounts(meta.ToolCounts); toolCounts != "" {
		parts = append(parts, theme.StyleMetaValue.Render(toolCounts))
	}
	return strings.Join(parts, sep)
}

func renderPreviewTimingParts(theme *el.Theme, meta conv.SessionMeta, sep, tsFmt string) string {
	var parts []string
	if !meta.Timestamp.IsZero() {
		parts = append(
			parts,
			theme.StyleMetaLabel.Render("started ")+
				theme.StyleMetaValue.Render(meta.Timestamp.Format(tsFmt)),
		)
	}
	if !meta.LastTimestamp.IsZero() {
		parts = append(
			parts,
			theme.StyleMetaLabel.Render("last ")+
				theme.StyleMetaValue.Render(meta.LastTimestamp.Format(tsFmt)),
		)
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

func renderInitialPrompt(theme *el.Theme, prompt string, width int) string {
	if prompt == "" {
		return ""
	}

	promptStyle := lipgloss.NewStyle().Foreground(theme.ColorPrimary)
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

func renderPreview(theme *el.Theme, session conv.Session, maxMessages int, width int, tsFmt string) string {
	var sb strings.Builder
	sb.WriteString(renderPreviewHeader(theme, session.Meta, tsFmt))
	sb.WriteString(renderInitialPrompt(theme, firstUserMessage(session.Messages), width))

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
