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
	usage             conv.TokenUsage
	toolCalls         []conv.ToolCall
	toolResults       []conv.ToolResult
	plans             []conv.Plan
	visibility        conv.MessageVisibility
	isAgentDivider    bool
	performance       conv.MessagePerformanceMeta
}

type assistantContent struct {
	thinking          string
	hasHiddenThinking bool
	calls             []conv.ToolCall
	results           []conv.ToolResult
	plans             []conv.Plan
	usage             conv.TokenUsage
	text              string
	timestamp         time.Time
	performance       conv.MessagePerformanceMeta
}

func (c assistantContent) empty() bool {
	return c.text == "" &&
		c.thinking == "" &&
		!c.hasHiddenThinking &&
		len(c.calls) == 0 &&
		len(c.results) == 0 &&
		len(c.plans) == 0 &&
		c.usage.TotalTokens() == 0
}

func appendParsedAssistantMessage(messages []parsedMessage, c assistantContent) ([]parsedMessage, int, bool) {
	if c.empty() {
		return messages, -1, false
	}

	msg := newParsedAssistantMessage(c)

	if len(messages) == 0 {
		return append(messages, msg), 0, true
	}

	if merged := mergeAdjacentAssistantMessage(messages, msg); merged != nil {
		return merged, len(merged) - 1, true
	}

	messages = append(messages, msg)
	return messages, len(messages) - 1, true
}

func newParsedAssistantMessage(c assistantContent) parsedMessage {
	return parsedMessage{
		role:              conv.RoleAssistant,
		timestamp:         c.timestamp,
		text:              c.text,
		thinking:          c.thinking,
		hasHiddenThinking: c.hasHiddenThinking,
		usage:             c.usage,
		toolCalls:         append([]conv.ToolCall(nil), c.calls...),
		toolResults:       append([]conv.ToolResult(nil), c.results...),
		plans:             append([]conv.Plan(nil), c.plans...),
		performance:       c.performance,
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
	last.usage = preferTokenUsage(last.usage, msg.usage)
	last.toolCalls = append(last.toolCalls, msg.toolCalls...)
	last.toolResults = append(last.toolResults, msg.toolResults...)
	last.plans = appendUniquePlans(last.plans, msg.plans)
	last.performance = mergeMessagePerformance(last.performance, msg.performance)
	if msg.timestamp.After(last.timestamp) {
		last.timestamp = msg.timestamp
	}
	return messages
}

func mergeMessagePerformance(
	current conv.MessagePerformanceMeta,
	next conv.MessagePerformanceMeta,
) conv.MessagePerformanceMeta {
	current.ReasoningBlockCount += next.ReasoningBlockCount
	current.ReasoningRedactionCount += next.ReasoningRedactionCount
	if current.StopReason == "" {
		current.StopReason = next.StopReason
	}
	if current.Phase == "" {
		current.Phase = next.Phase
	}
	if current.Effort == "" {
		current.Effort = next.Effort
	}
	return current
}

func preferTokenUsage(current, next conv.TokenUsage) conv.TokenUsage {
	if next.TotalTokens() > current.TotalTokens() {
		return next
	}
	return current
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
			Usage:             msg.usage,
			ToolCalls:         msg.toolCalls,
			ToolResults:       msg.toolResults,
			Plans:             msg.plans,
			Visibility:        msg.visibility,
			IsAgentDivider:    msg.isAgentDivider,
			Performance:       msg.performance,
			Timestamp:         msg.timestamp,
		})
	}
	return projected
}
