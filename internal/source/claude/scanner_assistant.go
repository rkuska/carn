package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rs/zerolog"
)

var toolParamKey = map[string]string{
	"Read":          "file_path",
	"Write":         "file_path",
	"Edit":          "file_path",
	"Glob":          "pattern",
	"Grep":          "pattern",
	"WebFetch":      "url",
	"WebSearch":     "query",
	"Skill":         "skill",
	"TaskCreate":    "subject",
	"TaskUpdate":    "taskId",
	"TaskGet":       "taskId",
	"NotebookEdit":  "notebook_path",
	"EnterWorktree": "name",
	"TaskOutput":    "task_id",
}

var toolTruncateKey = map[string]string{
	"Bash":            "command",
	"Agent":           "prompt",
	"AskUserQuestion": "question",
	"Task":            "description",
}

var toolConstant = map[string]string{
	"EnterPlanMode": "enter plan mode",
	"ExitPlanMode":  "exit plan mode",
	"TaskList":      "list tasks",
}

// blockJoiner concatenates multiple block strings with newline separators.
// For the common single-block case it returns the string directly (zero alloc).
type blockJoiner struct {
	first string
	b     strings.Builder
	n     int
}

func (j *blockJoiner) add(s string) {
	if j.n == 0 {
		j.first = s
	} else {
		if j.n == 1 {
			j.b.WriteString(j.first)
		}
		j.b.WriteByte('\n')
		j.b.WriteString(s)
	}
	j.n++
}

func (j *blockJoiner) result() string {
	if j.n <= 1 {
		return j.first
	}
	return j.b.String()
}

func extractAssistantContent(
	raw json.RawMessage,
) (text, thinking string, toolCalls []toolCall, toolCallIDs []string, ok bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return "", "", nil, nil, false
	}

	var textJ, thinkJ blockJoiner
	parseOK := true
	_, err := jsonparser.ArrayEach(raw, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if !parseOK {
			return
		}
		if err != nil || dataType != jsonparser.Object {
			parseOK = false
			return
		}
		if err := appendAssistantContentBlock(
			value,
			&textJ,
			&thinkJ,
			&toolCalls,
			&toolCallIDs,
		); err != nil {
			parseOK = false
		}
	})
	if err != nil || !parseOK {
		return "", "", nil, nil, false
	}
	return textJ.result(), thinkJ.result(), toolCalls, toolCallIDs, true
}

func appendAssistantContentBlock(
	value []byte,
	textJ *blockJoiner,
	thinkJ *blockJoiner,
	toolCalls *[]toolCall,
	toolCallIDs *[]string,
) error {
	blockType, _, err := jsonStringField(value, "type")
	if err != nil {
		return fmt.Errorf("appendAssistantContentBlock_type: %w", err)
	}

	switch blockType {
	case blockTypeText:
		return appendAssistantTextBlock(value, textJ)
	case blockTypeThinking:
		return appendAssistantThinkingBlock(value, thinkJ)
	case blockTypeToolUse:
		return appendAssistantToolUseBlock(value, toolCalls, toolCallIDs)
	default:
		return nil
	}
}

func appendAssistantTextBlock(value []byte, textJ *blockJoiner) error {
	blockText, _, err := jsonStringField(value, "text")
	if err != nil {
		return fmt.Errorf("appendAssistantTextBlock_text: %w", err)
	}
	textJ.add(blockText)
	return nil
}

func appendAssistantThinkingBlock(value []byte, thinkJ *blockJoiner) error {
	blockThinking, _, err := jsonStringField(value, "thinking")
	if err != nil {
		return fmt.Errorf("appendAssistantThinkingBlock_thinking: %w", err)
	}
	thinkJ.add(blockThinking)
	return nil
}

func appendAssistantToolUseBlock(
	value []byte,
	toolCalls *[]toolCall,
	toolCallIDs *[]string,
) error {
	name, _, err := jsonStringField(value, "name")
	if err != nil {
		return fmt.Errorf("appendAssistantToolUseBlock_name: %w", err)
	}
	id, _, err := jsonStringField(value, "id")
	if err != nil {
		return fmt.Errorf("appendAssistantToolUseBlock_id: %w", err)
	}
	input, _, err := jsonRawField(value, "input")
	if err != nil {
		return fmt.Errorf("appendAssistantToolUseBlock_input: %w", err)
	}

	*toolCalls = append(*toolCalls, toolCall{
		Name:    name,
		Summary: summarizeToolCallFast(name, input),
	})
	*toolCallIDs = append(*toolCallIDs, id)
	return nil
}

func parseParsedAssistantMessage(ctx context.Context, pc *parseContext) (parsedMessage, []string, bool) {
	text, thinking, toolCalls, toolCallIDs, ok := extractAssistantContent(pc.rec.Message.Content)
	if !ok {
		zerolog.Ctx(ctx).Debug().Msg("failed to parse assistant content blocks")
		return parsedMessage{}, nil, false
	}
	if text == "" && thinking == "" && len(toolCalls) == 0 {
		return parsedMessage{}, nil, false
	}

	var usage tokenUsage
	if pc.rec.Message.Usage != nil {
		usage = tokenUsage{
			InputTokens:              pc.rec.Message.Usage.InputTokens,
			CacheCreationInputTokens: pc.rec.Message.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     pc.rec.Message.Usage.CacheReadInputTokens,
			OutputTokens:             pc.rec.Message.Usage.OutputTokens,
		}
	}

	return parsedMessage{
		message: message{
			Role:        roleAssistant,
			Text:        text,
			Thinking:    thinking,
			ToolCalls:   toolCalls,
			IsSidechain: pc.rec.IsSidechain,
		},
		timestamp: parseRecordTimestamp(pc.rec.Timestamp),
		usage:     usage,
	}, toolCallIDs, true
}

func parseRecordTimestamp(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	timestamp, _ := time.Parse(time.RFC3339Nano, value)
	return timestamp
}

func summarizeToolCallFast(name string, input json.RawMessage) string {
	if paramKey, ok := toolParamKey[name]; ok {
		if value, ok := extractTopLevelJSONStringFieldFast(input, paramKey); ok {
			return value
		}
	}
	if paramKey, ok := toolTruncateKey[name]; ok {
		if value, ok := extractTopLevelJSONStringFieldFast(input, paramKey); ok {
			return conv.Truncate(value, 80)
		}
	}
	if constant, ok := toolConstant[name]; ok {
		return constant
	}
	if strings.HasPrefix(name, "mcp__") {
		if value := summarizeMCPToolFast(input); value != "" {
			return value
		}
	}
	return name
}

func summarizeMCPToolFast(raw json.RawMessage) string {
	for _, key := range []string{"query", "libraryName"} {
		if value, ok := extractTopLevelJSONStringFieldFast(raw, key); ok && value != "" {
			return conv.Truncate(value, 80)
		}
	}
	if value, ok := firstTopLevelJSONStringFieldFast(raw); ok && value != "" {
		return conv.Truncate(value, 80)
	}
	return ""
}
