package app

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	conv "github.com/rkuska/carn/internal/conversation"
)

const conversationHeaderInset = 4

func singleSessionConversation(meta conv.SessionMeta) conv.Conversation {
	return conv.Conversation{
		Name:    meta.Slug,
		Project: meta.Project,
		Sessions: []conv.SessionMeta{
			meta,
		},
	}
}

func renderConversationHeader(conversation conv.Conversation, width int, tsFmt string) string {
	if len(conversation.Sessions) == 0 || width <= 0 {
		return ""
	}

	contentWidth := max(width-conversationHeaderInset, 1)
	var lines []string

	if badges := renderWrappedTokens(headerBadges(conversation), contentWidth); badges != "" {
		lines = append(lines, badges)
	}
	if summary := renderWrappedTokens(headerSummaryChips(conversation), contentWidth); summary != "" {
		lines = append(lines, summary)
	}
	if timing := renderWrappedTokens(headerTimingChips(conversation, tsFmt), contentWidth); timing != "" {
		lines = append(lines, timing)
	}

	if tools := conv.FormatToolCounts(conversation.TotalToolCounts()); tools != "" {
		lines = append(lines, renderSingleChip("tools", tools))
	}

	if cwd := compactCWD(conversation.ResumeCWD()); cwd != "" {
		lines = append(lines, renderSingleChip("cwd", cwd))
	}

	return renderInsetBox(width, colorSecondary, strings.Join(lines, "\n")) + "\n\n"
}

func headerBadges(conversation conv.Conversation) []string {
	var badges []string
	if conversation.IsSubagent() {
		badges = append(badges, renderHeaderBadge("subagent", colorAccent))
	}
	if parts := conversation.PartCount(); parts > 1 {
		badges = append(badges, renderHeaderBadge(fmt.Sprintf("%d parts", parts), colorSecondary))
	}
	if conversation.PlanCount > 0 {
		label := fmt.Sprintf("%d plan", conversation.PlanCount)
		if conversation.PlanCount > 1 {
			label += "s"
		}
		badges = append(badges, renderHeaderBadge(label, colorPrimary))
	}
	return badges
}

// renderPlanHeader renders plans inside bordered boxes below the conversation
// metadata header. Returns empty if no plans exist.
// When collapsed, shows only the last plan filename; when expanded, shows all plans.
func renderPlanHeader(messages []conv.Message, width int, expanded bool) string {
	plans := conv.AllPlans(messages)
	if len(plans) == 0 {
		return ""
	}
	if !expanded {
		p := plans[len(plans)-1]
		title := fmt.Sprintf("Plan: %s", filepath.Base(p.FilePath))
		return renderInsetBox(width, colorPrimary, title) + "\n\n"
	}
	var sb strings.Builder
	for _, p := range plans {
		title := fmt.Sprintf("Plan: %s", filepath.Base(p.FilePath))
		sb.WriteString(renderInsetBox(width, colorPrimary, title+"\n\n"+p.Content))
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func headerSummaryChips(conversation conv.Conversation) []string {
	var chips []string
	if provider := conversationProviderLabel(conversation); provider != "" {
		chips = append(chips, renderSingleChip("provider", provider))
	}
	if model := conversation.Model(); model != "" {
		chips = append(chips, renderSingleChip("model", model))
	}
	if version := conversation.Version(); version != "" {
		chips = append(chips, renderSingleChip("version", version))
	}
	if branch := conversation.GitBranch(); branch != "" {
		chips = append(chips, renderSingleChip("branch", branch))
	}
	if d := conversation.Duration(); d > 0 {
		chips = append(chips, renderSingleChip("duration", conv.FormatDuration(d)))
	}
	if msgs := formatConversationMessages(conversation); msgs != "" {
		chips = append(chips, renderSingleChip("msgs", msgs))
	}
	if total := conversation.TotalTokenUsage().TotalTokens(); total > 0 {
		chips = append(chips, renderSingleChip("tokens", formatTokenCount(total)))
	}
	return chips
}

func headerTimingChips(conversation conv.Conversation, tsFmt string) []string {
	var chips []string

	if started := conversation.Timestamp(); !started.IsZero() {
		chips = append(chips, renderSingleChip("started", started.Format(tsFmt)))
	}
	if last := conversationLastTimestamp(conversation); !last.IsZero() {
		chips = append(chips, renderSingleChip("last", last.Format(tsFmt)))
	}
	if conversation.PartCount() > 1 {
		chips = append(chips, renderSingleChip("resume", shortID(conversation.ResumeID())))
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

func formatConversationMessages(conversation conv.Conversation) string {
	total := conversation.TotalMessageCount()
	if total == 0 {
		return ""
	}

	main := conversation.MainMessageCount()
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

func conversationLastTimestamp(conversation conv.Conversation) time.Time {
	var latest time.Time
	for _, session := range conversation.Sessions {
		if session.LastTimestamp.After(latest) {
			latest = session.LastTimestamp
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
