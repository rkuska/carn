package claude

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
		Role:           msg.role,
		Text:           msg.text,
		Thinking:       msg.thinking,
		ToolCalls:      projectParsedToolCalls(msg.toolCalls),
		ToolResults:    projectParsedToolResults(msg.toolResults),
		Plans:          msg.plans,
		IsSidechain:    msg.isSidechain,
		IsAgentDivider: msg.isAgentDivider,
	}
}

func projectParsedToolCalls(calls []parsedToolCall) []toolCall {
	projected := make([]toolCall, 0, len(calls))
	for _, call := range calls {
		projected = append(projected, toolCall{
			Name:    call.name,
			Summary: call.summary,
		})
	}
	return projected
}

func projectParsedToolResults(results []parsedToolResult) []toolResult {
	projected := make([]toolResult, 0, len(results))
	for _, result := range results {
		projected = append(projected, toolResult{
			ToolName:        result.toolName,
			ToolSummary:     result.toolSummary,
			Content:         result.content,
			IsError:         result.isError,
			StructuredPatch: result.structuredPatch,
		})
	}
	return projected
}
