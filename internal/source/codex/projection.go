package codex

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

func appendAssistantMessage(
	messages []conv.Message,
	thinking string,
	calls []conv.ToolCall,
	results []conv.ToolResult,
	plans []conv.Plan,
	text string,
) []conv.Message {
	if text == "" && thinking == "" && len(calls) == 0 && len(results) == 0 && len(plans) == 0 {
		return messages
	}

	msg := conv.Message{
		Role:        conv.RoleAssistant,
		Text:        text,
		Thinking:    thinking,
		ToolCalls:   append([]conv.ToolCall(nil), calls...),
		ToolResults: append([]conv.ToolResult(nil), results...),
		Plans:       append([]conv.Plan(nil), plans...),
	}

	if len(messages) == 0 {
		return append(messages, msg)
	}

	last := &messages[len(messages)-1]
	if last.Role == conv.RoleAssistant && last.Text == msg.Text && !last.IsAgentDivider {
		last.Thinking = joinUniqueText(last.Thinking, msg.Thinking)
		last.ToolCalls = append(last.ToolCalls, msg.ToolCalls...)
		last.ToolResults = append(last.ToolResults, msg.ToolResults...)
		last.Plans = appendUniquePlans(last.Plans, msg.Plans)
		return messages
	}

	return append(messages, msg)
}

func joinText(parts []string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.TrimSpace(strings.Join(filtered, "\n\n"))
}

func joinUniqueText(existing, added string) string {
	switch {
	case strings.TrimSpace(added) == "":
		return existing
	case strings.TrimSpace(existing) == "":
		return added
	case strings.TrimSpace(existing) == strings.TrimSpace(added):
		return existing
	default:
		return existing + "\n\n" + added
	}
}

func appendUniquePlans(existing []conv.Plan, added []conv.Plan) []conv.Plan {
	if len(added) == 0 {
		return existing
	}

	seen := make(map[string]struct{}, len(existing))
	for _, plan := range existing {
		seen[plan.FilePath] = struct{}{}
	}
	for _, plan := range added {
		if _, ok := seen[plan.FilePath]; ok {
			continue
		}
		seen[plan.FilePath] = struct{}{}
		existing = append(existing, plan)
	}
	return existing
}

func extractCompletedPlan(raw json.RawMessage, ts time.Time) (conv.Plan, bool) {
	if len(raw) == 0 {
		return conv.Plan{}, false
	}

	var item completedItemPayload
	if err := json.Unmarshal(raw, &item); err != nil {
		return conv.Plan{}, false
	}
	if item.Type != eventItemTypePlan || strings.TrimSpace(item.Text) == "" {
		return conv.Plan{}, false
	}

	filePath := strings.TrimSpace(item.ID)
	if filePath == "" {
		filePath = "plan"
	}
	if !strings.Contains(filepath.Base(filePath), ".") {
		filePath += ".md"
	}

	return conv.Plan{
		FilePath:  filePath,
		Content:   strings.TrimSpace(item.Text),
		Timestamp: ts,
	}, true
}
