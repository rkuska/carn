package app

import "time"

type scannedSession struct {
	meta                   sessionMeta
	groupKey               groupKey
	hasConversationContent bool
}

type scannedProject struct {
	dirName     string
	displayName string
}

type parsedMessage struct {
	role           role
	timestamp      time.Time
	text           string
	thinking       string
	toolCalls      []parsedToolCall
	toolResults    []parsedToolResult
	plans          []plan
	usage          tokenUsage
	isSidechain    bool
	isAgentDivider bool
}

type parsedToolCall struct {
	id      string
	name    string
	summary string
}

type parsedToolResult struct {
	toolUseID       string
	toolName        string
	toolSummary     string
	content         string
	isError         bool
	structuredPatch []diffHunk
}

type parsedLinkedTranscript struct {
	kind     linkedTranscriptKind
	title    string
	anchor   time.Time
	messages []parsedMessage
}

func projectParsedMessages(messages []parsedMessage) []message {
	projected := make([]message, 0, len(messages))
	for _, msg := range messages {
		projected = append(projected, projectParsedMessage(msg))
	}
	return projected
}

func projectParsedMessage(msg parsedMessage) message {
	return message{
		role:           msg.role,
		text:           msg.text,
		thinking:       msg.thinking,
		toolCalls:      projectParsedToolCalls(msg.toolCalls),
		toolResults:    projectParsedToolResults(msg.toolResults),
		plans:          msg.plans,
		isSidechain:    msg.isSidechain,
		isAgentDivider: msg.isAgentDivider,
	}
}

func projectParsedToolCalls(calls []parsedToolCall) []toolCall {
	projected := make([]toolCall, 0, len(calls))
	for _, call := range calls {
		projected = append(projected, toolCall{
			name:    call.name,
			summary: call.summary,
		})
	}
	return projected
}

func projectParsedToolResults(results []parsedToolResult) []toolResult {
	projected := make([]toolResult, 0, len(results))
	for _, result := range results {
		projected = append(projected, toolResult{
			toolName:        result.toolName,
			toolSummary:     result.toolSummary,
			content:         result.content,
			isError:         result.isError,
			structuredPatch: result.structuredPatch,
		})
	}
	return projected
}
