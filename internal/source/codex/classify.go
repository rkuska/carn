package codex

import (
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

type visibleMessage struct {
	role           conv.Role
	text           string
	isAgentDivider bool
}

func classifyResponseMessage(role string, blocks []contentBlock) (visibleMessage, bool) {
	return classifyTextMessage(role, extractMessageText(blocks))
}

func classifyEventUserMessage(text string) (visibleMessage, bool) {
	return classifyTextMessage(responseRoleUser, text)
}

func classifyEventAssistantMessage(text string) (visibleMessage, bool) {
	return classifyTextMessage(responseRoleAssistant, text)
}

func classifyTaskCompleteMessage(text string) (visibleMessage, bool) {
	return classifyEventAssistantMessage(text)
}

func classifyTextMessage(role string, text string) (visibleMessage, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return visibleMessage{}, false
	}

	switch role {
	case responseRoleDeveloper:
		return visibleMessage{}, false
	case responseRoleUser:
		if isCodexBootstrapMessage(text) {
			return visibleMessage{}, false
		}
		if notification, ok := unwrapTagText(text, "subagent_notification"); ok {
			return visibleMessage{
				role:           conv.RoleUser,
				text:           notification,
				isAgentDivider: true,
			}, true
		}
		return visibleMessage{role: conv.RoleUser, text: text}, true
	case responseRoleAssistant:
		return visibleMessage{role: conv.RoleAssistant, text: text}, true
	default:
		return visibleMessage{}, false
	}
}

func isCodexBootstrapMessage(text string) bool {
	return strings.HasPrefix(text, "# AGENTS.md instructions for ") ||
		isWrappedTag(text, "environment_context") ||
		isWrappedTag(text, "permissions instructions")
}

func isWrappedTag(text, tag string) bool {
	_, ok := unwrapTagText(text, tag)
	return ok
}

func unwrapTagText(text, tag string) (string, bool) {
	text = strings.TrimSpace(text)
	open := "<" + tag + ">"
	close := "</" + tag + ">"
	if !strings.HasPrefix(text, open) || !strings.HasSuffix(text, close) {
		return "", false
	}
	body := strings.TrimSpace(text[len(open) : len(text)-len(close)])
	return body, true
}
