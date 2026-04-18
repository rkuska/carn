package browser

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	el "github.com/rkuska/carn/internal/app/elements"
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

func renderConversationHeader(theme *el.Theme, conversation conv.Conversation, width int, tsFmt string) string {
	if len(conversation.Sessions) == 0 || width <= 0 {
		return ""
	}

	contentWidth := max(width-conversationHeaderInset, 1)
	var lines []string

	if badges := renderWrappedTokens(headerBadges(theme, conversation), contentWidth); badges != "" {
		lines = append(lines, badges)
	}
	if summary := renderWrappedTokens(headerSummaryChips(theme, conversation), contentWidth); summary != "" {
		lines = append(lines, summary)
	}
	if timing := renderWrappedTokens(headerTimingChips(theme, conversation, tsFmt), contentWidth); timing != "" {
		lines = append(lines, timing)
	}

	if tools := conv.FormatToolCounts(conversation.TotalToolCounts()); tools != "" {
		lines = append(lines, renderSingleChip(theme, "tools", tools))
	}

	if cwd := conv.CompactCWD(conversation.ResumeCWD()); cwd != "" {
		lines = append(lines, renderSingleChip(theme, "cwd", cwd))
	}

	return renderInsetBox(width, theme.ColorSecondary, strings.Join(lines, "\n")) + "\n\n"
}

func headerBadges(theme *el.Theme, conversation conv.Conversation) []string {
	var badges []string
	if conversation.IsSubagent() {
		badges = append(badges, renderHeaderBadge(theme, "subagent", theme.ColorAccent))
	}
	if parts := conversation.PartCount(); parts > 1 {
		badges = append(badges, renderHeaderBadge(theme, fmt.Sprintf("%d parts", parts), theme.ColorSecondary))
	}
	if conversation.PlanCount > 0 {
		label := fmt.Sprintf("%d plan", conversation.PlanCount)
		if conversation.PlanCount > 1 {
			label += "s"
		}
		badges = append(badges, renderHeaderBadge(theme, label, theme.ColorPrimary))
	}
	return badges
}

// renderPlanHeader renders plans inside bordered boxes below the conversation
// metadata header. Returns empty if no plans exist.
// When collapsed, shows only the last plan filename; when expanded, shows all plans.
func renderPlanHeader(theme *el.Theme, messages []conv.Message, width int, expanded bool) string {
	plans := conv.AllPlans(messages)
	if len(plans) == 0 {
		return ""
	}
	if !expanded {
		p := plans[len(plans)-1]
		title := fmt.Sprintf("Plan: %s", filepath.Base(p.FilePath))
		return renderInsetBox(width, theme.ColorPrimary, title) + "\n\n"
	}
	var sb strings.Builder
	for _, p := range plans {
		title := fmt.Sprintf("Plan: %s", filepath.Base(p.FilePath))
		sb.WriteString(renderInsetBox(width, theme.ColorPrimary, title+"\n\n"+p.Content))
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func headerSummaryChips(theme *el.Theme, conversation conv.Conversation) []string {
	var chips []string
	if provider := conversationProviderLabel(conversation); provider != "" {
		chips = append(chips, renderSingleChip(theme, "provider", provider))
	}
	if model := conversation.Model(); model != "" {
		chips = append(chips, renderSingleChip(theme, "model", model))
	}
	if version := conversation.Version(); version != "" {
		chips = append(chips, renderSingleChip(theme, "version", version))
	}
	if branch := conversation.GitBranch(); branch != "" {
		chips = append(chips, renderSingleChip(theme, "branch", branch))
	}
	if d := conversation.Duration(); d > 0 {
		chips = append(chips, renderSingleChip(theme, "duration", conv.FormatDuration(d)))
	}
	if msgs := formatConversationMessages(conversation); msgs != "" {
		chips = append(chips, renderSingleChip(theme, "msgs", msgs))
	}
	if total := conversation.TotalTokenUsage().TotalTokens(); total > 0 {
		chips = append(chips, renderSingleChip(theme, "tokens", formatTokenCount(total)))
	}
	return chips
}

func headerTimingChips(theme *el.Theme, conversation conv.Conversation, tsFmt string) []string {
	var chips []string

	if started := conversation.Timestamp(); !started.IsZero() {
		chips = append(chips, renderSingleChip(theme, "started", started.Format(tsFmt)))
	}
	if last := conversationLastTimestamp(conversation); !last.IsZero() {
		chips = append(chips, renderSingleChip(theme, "last", last.Format(tsFmt)))
	}
	if conversation.PartCount() > 1 {
		chips = append(chips, renderSingleChip(theme, "resume", shortID(conversation.ResumeID())))
	}
	return chips
}

func renderHeaderBadge(theme *el.Theme, text string, bg color.Color) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorStatusFg).
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
