package app

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

const conversationHeaderInset = 4

func singleSessionConversation(meta sessionMeta) conversation {
	return conversation{
		name:    meta.slug,
		project: meta.project,
		sessions: []sessionMeta{
			meta,
		},
	}
}

func renderConversationHeader(conv conversation, width int) string {
	if len(conv.sessions) == 0 || width <= 0 {
		return ""
	}

	contentWidth := max(width-conversationHeaderInset, 1)
	var lines []string

	if badges := renderWrappedTokens(headerBadges(conv), contentWidth); badges != "" {
		lines = append(lines, badges)
	}
	if summary := renderWrappedTokens(headerSummaryChips(conv), contentWidth); summary != "" {
		lines = append(lines, summary)
	}
	if timing := renderWrappedTokens(headerTimingChips(conv), contentWidth); timing != "" {
		lines = append(lines, timing)
	}

	if tools := formatToolCounts(conv.totalToolCounts()); tools != "" {
		lines = append(lines, renderSingleChip("tools", tools))
	}

	if cwd := compactCWD(conv.resumeCWD()); cwd != "" {
		lines = append(lines, renderSingleChip("cwd", cwd))
	}

	return renderInsetBox(width, colorSecondary, strings.Join(lines, "\n")) + "\n\n"
}

func headerBadges(conv conversation) []string {
	var badges []string
	if conv.isSubagent() {
		badges = append(badges, renderHeaderBadge("subagent", colorAccent))
	}
	if parts := len(conv.sessions); parts > 1 {
		badges = append(badges, renderHeaderBadge(fmt.Sprintf("%d parts", parts), colorSecondary))
	}
	return badges
}

func headerSummaryChips(conv conversation) []string {
	var chips []string
	if model := conv.model(); model != "" {
		chips = append(chips, renderSingleChip("model", model))
	}
	if version := conv.version(); version != "" {
		chips = append(chips, renderSingleChip("version", version))
	}
	if branch := conv.gitBranch(); branch != "" {
		chips = append(chips, renderSingleChip("branch", branch))
	}
	if d := conv.duration(); d > 0 {
		chips = append(chips, renderSingleChip("duration", formatDuration(d)))
	}
	if msgs := formatConversationMessages(conv); msgs != "" {
		chips = append(chips, renderSingleChip("msgs", msgs))
	}
	if total := conv.totalTokenUsage().totalTokens(); total > 0 {
		chips = append(chips, renderSingleChip("tokens", formatTokenCount(total)))
	}
	return chips
}

func headerTimingChips(conv conversation) []string {
	var chips []string
	const tsFmt = "2006-01-02 15:04"

	if started := conv.timestamp(); !started.IsZero() {
		chips = append(chips, renderSingleChip("started", started.Format(tsFmt)))
	}
	if last := conversationLastTimestamp(conv); !last.IsZero() {
		chips = append(chips, renderSingleChip("last", last.Format(tsFmt)))
	}
	if len(conv.sessions) > 1 {
		chips = append(chips, renderSingleChip("resume", shortID(conv.resumeID())))
	}
	return chips
}

func renderWrappedTokens(tokens []string, width int) string {
	if len(tokens) == 0 {
		return ""
	}

	const sep = "  "
	lines := make([]string, 0, len(tokens))
	current := tokens[0]
	for _, token := range tokens[1:] {
		if lipgloss.Width(current+sep+token) <= width {
			current += sep + token
			continue
		}
		lines = append(lines, current)
		current = token
	}
	lines = append(lines, current)
	return strings.Join(lines, "\n")
}

func renderSingleChip(label, value string) string {
	return styleMetaLabel.Render(label) + " " + styleMetaValue.Render(value)
}

func renderHeaderBadge(text string, bg color.Color) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(colorStatusFg).
		Background(bg).
		Padding(0, 1).
		Render(text)
}

func formatConversationMessages(conv conversation) string {
	total := conv.totalMessageCount()
	if total == 0 {
		return ""
	}

	main := conv.mainMessageCount()
	if main > 0 && main != total {
		return fmt.Sprintf("%d/%d", main, total)
	}
	return fmt.Sprintf("%d", total)
}

func formatTokenCount(total int) string {
	if total < 1000 {
		return fmt.Sprintf("%d", total)
	}
	return fmt.Sprintf("%dk", total/1000)
}

func conversationLastTimestamp(conv conversation) time.Time {
	var latest time.Time
	for _, session := range conv.sessions {
		if session.lastTimestamp.After(latest) {
			latest = session.lastTimestamp
		}
	}
	return latest
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func compactCWD(cwd string) string {
	if cwd == "" {
		return ""
	}

	clean := filepath.ToSlash(filepath.Clean(cwd))
	parts := strings.Split(clean, "/")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) >= 2 {
		return strings.Join(filtered[len(filtered)-2:], "/")
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return clean
}
