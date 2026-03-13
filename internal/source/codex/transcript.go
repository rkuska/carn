package codex

import (
	"strings"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type parsedMessage struct {
	role              conv.Role
	timestamp         time.Time
	text              string
	thinking          string
	hasHiddenThinking bool
	toolCalls         []conv.ToolCall
	toolResults       []conv.ToolResult
	plans             []conv.Plan
	visibility        conv.MessageVisibility
	isAgentDivider    bool
}

func appendParsedAssistantMessage(
	messages []parsedMessage,
	thinking string,
	hasHiddenThinking bool,
	calls []conv.ToolCall,
	results []conv.ToolResult,
	plans []conv.Plan,
	text string,
	timestamp time.Time,
) []parsedMessage {
	if assistantMessageEmpty(thinking, hasHiddenThinking, calls, results, plans, text) {
		return messages
	}

	msg := newParsedAssistantMessage(thinking, hasHiddenThinking, calls, results, plans, text, timestamp)

	if len(messages) == 0 {
		return append(messages, msg)
	}

	if merged := mergeAdjacentAssistantMessage(messages, msg); merged != nil {
		return merged
	}

	return append(messages, msg)
}

func assistantMessageEmpty(
	thinking string,
	hasHiddenThinking bool,
	calls []conv.ToolCall,
	results []conv.ToolResult,
	plans []conv.Plan,
	text string,
) bool {
	return text == "" &&
		thinking == "" &&
		!hasHiddenThinking &&
		len(calls) == 0 &&
		len(results) == 0 &&
		len(plans) == 0
}

func newParsedAssistantMessage(
	thinking string,
	hasHiddenThinking bool,
	calls []conv.ToolCall,
	results []conv.ToolResult,
	plans []conv.Plan,
	text string,
	timestamp time.Time,
) parsedMessage {
	return parsedMessage{
		role:              conv.RoleAssistant,
		timestamp:         timestamp,
		text:              text,
		thinking:          thinking,
		hasHiddenThinking: hasHiddenThinking,
		toolCalls:         append([]conv.ToolCall(nil), calls...),
		toolResults:       append([]conv.ToolResult(nil), results...),
		plans:             append([]conv.Plan(nil), plans...),
	}
}

func mergeAdjacentAssistantMessage(messages []parsedMessage, msg parsedMessage) []parsedMessage {
	last := &messages[len(messages)-1]
	if last.role != conv.RoleAssistant || last.text != msg.text || last.isAgentDivider {
		return nil
	}

	last.thinking = joinUniqueText(last.thinking, msg.thinking)
	last.hasHiddenThinking = strings.TrimSpace(last.thinking) == "" &&
		(last.hasHiddenThinking || msg.hasHiddenThinking)
	last.toolCalls = append(last.toolCalls, msg.toolCalls...)
	last.toolResults = append(last.toolResults, msg.toolResults...)
	last.plans = appendUniquePlans(last.plans, msg.plans)
	if msg.timestamp.After(last.timestamp) {
		last.timestamp = msg.timestamp
	}
	return messages
}

func appendParsedUserMessage(messages []parsedMessage, text string, timestamp time.Time) []parsedMessage {
	if text == "" {
		return messages
	}
	if len(messages) > 0 {
		last := messages[len(messages)-1]
		if last.role == conv.RoleUser && !last.isAgentDivider && last.text == text {
			return messages
		}
	}
	return append(messages, parsedMessage{role: conv.RoleUser, text: text, timestamp: timestamp})
}

func appendParsedSystemMessage(
	messages []parsedMessage,
	text string,
	visibility conv.MessageVisibility,
	timestamp time.Time,
) []parsedMessage {
	if text == "" {
		return messages
	}
	if len(messages) > 0 {
		last := messages[len(messages)-1]
		if last.role == conv.RoleSystem && last.text == text && last.visibility == visibility {
			return messages
		}
	}
	return append(messages, parsedMessage{
		role:       conv.RoleSystem,
		text:       text,
		timestamp:  timestamp,
		visibility: visibility,
	})
}

func appendParsedDividerMessage(messages []parsedMessage, text string, timestamp time.Time) []parsedMessage {
	if text == "" {
		return messages
	}
	if len(messages) > 0 {
		last := messages[len(messages)-1]
		if last.isAgentDivider && last.text == text {
			return messages
		}
	}
	return append(messages, parsedMessage{
		role:           conv.RoleUser,
		text:           text,
		timestamp:      timestamp,
		isAgentDivider: true,
	})
}

func projectParsedMessages(messages []parsedMessage) []conv.Message {
	projected := make([]conv.Message, 0, len(messages))
	for _, msg := range messages {
		projected = append(projected, conv.Message{
			Role:              msg.role,
			Text:              msg.text,
			Thinking:          msg.thinking,
			HasHiddenThinking: msg.hasHiddenThinking,
			ToolCalls:         append([]conv.ToolCall(nil), msg.toolCalls...),
			ToolResults:       append([]conv.ToolResult(nil), msg.toolResults...),
			Plans:             append([]conv.Plan(nil), msg.plans...),
			Visibility:        msg.visibility,
			IsAgentDivider:    msg.isAgentDivider,
		})
	}
	return projected
}
