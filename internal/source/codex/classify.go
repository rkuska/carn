package codex

import (
	"bytes"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

type visibleMessage struct {
	role           conv.Role
	text           string
	visibility     conv.MessageVisibility
	isAgentDivider bool
}

func classifyResponseMessage(role string, raw []byte) (visibleMessage, bool) {
	return classifyTextMessage(role, extractScanContentText(raw))
}

func classifyResponseMessageRaw(roleRaw []byte, raw []byte) (visibleMessage, bool) {
	return classifyTextMessageRaw(roleRaw, extractScanContentText(raw))
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

func classifyTextMessageRaw(roleRaw []byte, text string) (visibleMessage, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return visibleMessage{}, false
	}

	switch {
	case bytes.Equal(roleRaw, responseRoleDeveloperRaw):
		return hiddenSystemMessage(text), true
	case bytes.Equal(roleRaw, responseRoleUserRaw):
		if notification, ok := unwrapTagText(text, "subagent_notification"); ok {
			return visibleMessage{
				role:           conv.RoleUser,
				text:           notification,
				isAgentDivider: true,
			}, true
		}
		if isCodexBootstrapMessage(text) {
			return hiddenSystemMessage(text), true
		}
		return visibleMessage{role: conv.RoleUser, text: text}, true
	case bytes.Equal(roleRaw, responseRoleAssistantRaw):
		return visibleMessage{role: conv.RoleAssistant, text: text}, true
	default:
		role, ok := readRawJSONString(roleRaw)
		if !ok {
			return visibleMessage{}, false
		}
		return classifyTextMessage(role, text)
	}
}

func classifyTextMessage(role string, text string) (visibleMessage, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return visibleMessage{}, false
	}

	switch role {
	case responseRoleDeveloper:
		return hiddenSystemMessage(text), true
	case responseRoleUser:
		if notification, ok := unwrapTagText(text, "subagent_notification"); ok {
			return visibleMessage{
				role:           conv.RoleUser,
				text:           notification,
				isAgentDivider: true,
			}, true
		}
		if isCodexBootstrapMessage(text) {
			return hiddenSystemMessage(text), true
		}
		return visibleMessage{role: conv.RoleUser, text: text}, true
	case responseRoleAssistant:
		return visibleMessage{role: conv.RoleAssistant, text: text}, true
	default:
		return visibleMessage{}, false
	}
}

func hiddenSystemMessage(text string) visibleMessage {
	return visibleMessage{
		role:       conv.RoleSystem,
		text:       text,
		visibility: conv.MessageVisibilityHiddenSystem,
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
