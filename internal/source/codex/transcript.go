package codex

import (
	"sort"
	"strings"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type parsedMessage struct {
	role           conv.Role
	timestamp      time.Time
	text           string
	thinking       string
	toolCalls      []conv.ToolCall
	toolResults    []conv.ToolResult
	plans          []conv.Plan
	isAgentDivider bool
}

type linkedTranscript struct {
	title    string
	anchor   time.Time
	messages []parsedMessage
}

type rolloutTranscript struct {
	meta     conv.SessionMeta
	link     subagentLink
	messages []parsedMessage
}

func appendParsedAssistantMessage(
	messages []parsedMessage,
	thinking string,
	calls []conv.ToolCall,
	results []conv.ToolResult,
	plans []conv.Plan,
	text string,
	timestamp time.Time,
) []parsedMessage {
	if assistantMessageEmpty(thinking, calls, results, plans, text) {
		return messages
	}

	msg := newParsedAssistantMessage(thinking, calls, results, plans, text, timestamp)

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
	calls []conv.ToolCall,
	results []conv.ToolResult,
	plans []conv.Plan,
	text string,
) bool {
	return text == "" && thinking == "" && len(calls) == 0 && len(results) == 0 && len(plans) == 0
}

func newParsedAssistantMessage(
	thinking string,
	calls []conv.ToolCall,
	results []conv.ToolResult,
	plans []conv.Plan,
	text string,
	timestamp time.Time,
) parsedMessage {
	return parsedMessage{
		role:        conv.RoleAssistant,
		timestamp:   timestamp,
		text:        text,
		thinking:    thinking,
		toolCalls:   append([]conv.ToolCall(nil), calls...),
		toolResults: append([]conv.ToolResult(nil), results...),
		plans:       append([]conv.Plan(nil), plans...),
	}
}

func mergeAdjacentAssistantMessage(messages []parsedMessage, msg parsedMessage) []parsedMessage {
	last := &messages[len(messages)-1]
	if last.role != conv.RoleAssistant || last.text != msg.text || last.isAgentDivider {
		return nil
	}

	last.thinking = joinUniqueText(last.thinking, msg.thinking)
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
			Role:           msg.role,
			Text:           msg.text,
			Thinking:       msg.thinking,
			ToolCalls:      append([]conv.ToolCall(nil), msg.toolCalls...),
			ToolResults:    append([]conv.ToolResult(nil), msg.toolResults...),
			Plans:          append([]conv.Plan(nil), msg.plans...),
			IsAgentDivider: msg.isAgentDivider,
		})
	}
	return projected
}

func mergeLinkedTranscripts(base []parsedMessage, linked []linkedTranscript) []parsedMessage {
	if len(linked) == 0 {
		return base
	}

	projected := append(make([]parsedMessage, 0, len(base)+len(linked)*2), base...)
	for _, transcript := range linked {
		if len(transcript.messages) == 0 {
			continue
		}

		divider := parsedMessage{
			role:           conv.RoleUser,
			text:           transcript.title,
			timestamp:      transcript.anchor,
			isAgentDivider: true,
		}
		pos := findParsedInsertPosition(projected, transcript.anchor)
		projected = insertParsedMessage(projected, pos, divider)
		projected = insertParsedMessages(projected, pos+1, transcript.messages)
	}
	return projected
}

func findParsedInsertPosition(messages []parsedMessage, anchor time.Time) int {
	if anchor.IsZero() {
		return len(messages)
	}

	pos := 0
	for i, msg := range messages {
		if !msg.timestamp.IsZero() && !msg.timestamp.After(anchor) {
			pos = i + 1
		}
	}
	return pos
}

func insertParsedMessage(items []parsedMessage, index int, item parsedMessage) []parsedMessage {
	items = append(items, parsedMessage{})
	copy(items[index+1:], items[index:])
	items[index] = item
	return items
}

func insertParsedMessages(items []parsedMessage, index int, inserted []parsedMessage) []parsedMessage {
	if len(inserted) == 0 {
		return items
	}

	oldLen := len(items)
	items = append(items, make([]parsedMessage, len(inserted))...)
	copy(items[index+len(inserted):], items[index:oldLen])
	copy(items[index:], inserted)
	return items
}

func mergeSubagentTranscripts(parent rolloutTranscript, children []rolloutTranscript) []parsedMessage {
	if len(children) == 0 {
		return parent.messages
	}

	sortRolloutTranscripts(children)
	dividerIndexes := collectDividerIndexes(parent.messages)
	consumed := make(map[int]struct{}, min(len(dividerIndexes), len(children)))
	linked := make([]linkedTranscript, 0, len(children))

	for i, child := range children {
		title := linkedTranscriptTitle(child)
		if i < len(dividerIndexes) {
			index := dividerIndexes[i]
			consumed[index] = struct{}{}
			if parent.messages[index].text != "" {
				title = parent.messages[index].text
			}
		}
		linked = append(linked, linkedTranscript{
			title:    title,
			anchor:   linkedTranscriptAnchor(child),
			messages: child.messages,
		})
	}

	base := filterConsumedDividers(parent.messages, consumed)
	return mergeLinkedTranscripts(base, linked)
}

func collectDividerIndexes(messages []parsedMessage) []int {
	indexes := make([]int, 0)
	for i, msg := range messages {
		if msg.isAgentDivider {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func filterConsumedDividers(messages []parsedMessage, consumed map[int]struct{}) []parsedMessage {
	if len(consumed) == 0 {
		return messages
	}

	filtered := make([]parsedMessage, 0, len(messages)-len(consumed))
	for i, msg := range messages {
		if _, ok := consumed[i]; ok {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func sortRolloutTranscripts(items []rolloutTranscript) {
	slicesSortStableFunc(items, func(a, b rolloutTranscript) int {
		switch {
		case a.meta.Timestamp.IsZero() && b.meta.Timestamp.IsZero():
			return 0
		case a.meta.Timestamp.IsZero():
			return 1
		case b.meta.Timestamp.IsZero():
			return -1
		case a.meta.Timestamp.Before(b.meta.Timestamp):
			return -1
		case a.meta.Timestamp.After(b.meta.Timestamp):
			return 1
		default:
			return 0
		}
	})
}

func linkedTranscriptAnchor(transcript rolloutTranscript) time.Time {
	if !transcript.meta.Timestamp.IsZero() {
		return transcript.meta.Timestamp
	}
	for _, msg := range transcript.messages {
		if !msg.timestamp.IsZero() {
			return msg.timestamp
		}
	}
	return time.Time{}
}

func linkedTranscriptTitle(transcript rolloutTranscript) string {
	if prompt := firstUserPrompt(transcript.messages); prompt != "" {
		return prompt
	}
	if transcript.link.agentNickname != "" {
		return transcript.link.agentNickname
	}
	if transcript.link.agentRole != "" {
		return transcript.link.agentRole
	}
	return "Subagent"
}

func firstUserPrompt(messages []parsedMessage) string {
	for _, msg := range messages {
		if msg.role != conv.RoleUser || msg.isAgentDivider || strings.TrimSpace(msg.text) == "" {
			continue
		}
		return msg.text
	}
	return ""
}

func slicesSortStableFunc[T any](items []T, cmp func(T, T) int) {
	sort.SliceStable(items, func(i, j int) bool {
		return cmp(items[i], items[j]) < 0
	})
}
