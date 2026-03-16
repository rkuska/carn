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

type assistantContent struct {
	thinking          string
	hasHiddenThinking bool
	calls             []conv.ToolCall
	results           []conv.ToolResult
	plans             []conv.Plan
	text              string
	timestamp         time.Time
}

func (c assistantContent) empty() bool {
	return c.text == "" &&
		c.thinking == "" &&
		!c.hasHiddenThinking &&
		len(c.calls) == 0 &&
		len(c.results) == 0 &&
		len(c.plans) == 0
}

func appendParsedAssistantMessage(messages []parsedMessage, c assistantContent) []parsedMessage {
	if c.empty() {
		return messages
	}

	msg := newParsedAssistantMessage(c)

	if len(messages) == 0 {
		return append(messages, msg)
	}

	if merged := mergeAdjacentAssistantMessage(messages, msg); merged != nil {
		return merged
	}

	return append(messages, msg)
}

func newParsedAssistantMessage(c assistantContent) parsedMessage {
	return parsedMessage{
		role:              conv.RoleAssistant,
		timestamp:         c.timestamp,
		text:              c.text,
		thinking:          c.thinking,
		hasHiddenThinking: c.hasHiddenThinking,
		toolCalls:         append([]conv.ToolCall(nil), c.calls...),
		toolResults:       append([]conv.ToolResult(nil), c.results...),
		plans:             append([]conv.Plan(nil), c.plans...),
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
			ToolCalls:         msg.toolCalls,
			ToolResults:       msg.toolResults,
			Plans:             msg.plans,
			Visibility:        msg.visibility,
			IsAgentDivider:    msg.isAgentDivider,
		})
	}
	return projected
}
