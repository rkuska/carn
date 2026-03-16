package claude

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rs/zerolog"
)

type contentBlock struct {
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	Thinking string          `json:"thinking"`
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Input    json.RawMessage `json:"input"`
}

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

func extractAssistantContent(pc *parseContext) (text, thinking string, toolCalls []parsedToolCall) {
	var textJ, thinkJ blockJoiner
	toolUseCount := 0
	for _, block := range pc.blocks {
		if block.Type == blockTypeToolUse {
			toolUseCount++
		}
	}
	if toolUseCount > 0 {
		toolCalls = make([]parsedToolCall, 0, toolUseCount)
	}
	for _, block := range pc.blocks {
		switch block.Type {
		case blockTypeText:
			textJ.add(block.Text)
		case blockTypeThinking:
			thinkJ.add(block.Thinking)
		case blockTypeToolUse:
			toolCalls = append(toolCalls, parsedToolCall{
				id:      block.ID,
				name:    block.Name,
				summary: summarizeToolCallFast(block.Name, block.Input, pc),
			})
		}
	}
	return textJ.result(), thinkJ.result(), toolCalls
}

func parseParsedAssistantMessage(ctx context.Context, pc *parseContext) (parsedMessage, bool) {
	pc.blocks = pc.blocks[:0]
	if err := json.Unmarshal(pc.rec.Message.Content, &pc.blocks); err != nil {
		zerolog.Ctx(ctx).Debug().Err(err).Msg("failed to unmarshal assistant content blocks")
		return parsedMessage{}, false
	}

	text, thinking, toolCalls := extractAssistantContent(pc)
	if text == "" && thinking == "" && len(toolCalls) == 0 {
		return parsedMessage{}, false
	}

	var ts time.Time
	if pc.rec.Timestamp != "" {
		ts, _ = time.Parse(time.RFC3339Nano, pc.rec.Timestamp)
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
		role:        roleAssistant,
		timestamp:   ts,
		text:        text,
		thinking:    thinking,
		toolCalls:   toolCalls,
		usage:       usage,
		isSidechain: pc.rec.IsSidechain,
	}, true
}

func summarizeToolCallFast(name string, input json.RawMessage, pc *parseContext) string {
	pc.resetSummarizeParams()
	if err := json.Unmarshal(input, &pc.summarizeParam); err != nil {
		return name
	}
	return summarizeToolCallFromParams(name, pc.summarizeParam)
}

func summarizeToolCallFromParams(name string, params map[string]json.RawMessage) string {
	if paramKey, ok := toolParamKey[name]; ok {
		return extractStringParam(params, paramKey)
	}
	if paramKey, ok := toolTruncateKey[name]; ok {
		return conv.Truncate(extractStringParam(params, paramKey), 80)
	}
	if constant, ok := toolConstant[name]; ok {
		return constant
	}
	if strings.HasPrefix(name, "mcp__") {
		return summarizeMCPTool(params)
	}
	return ""
}

func extractStringParam(params map[string]json.RawMessage, key string) string {
	raw, ok := params[key]
	if !ok {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func summarizeMCPTool(params map[string]json.RawMessage) string {
	for _, key := range []string{"query", "libraryName"} {
		if value := extractStringParam(params, key); value != "" {
			return conv.Truncate(value, 80)
		}
	}
	for _, raw := range params {
		var value string
		if err := json.Unmarshal(raw, &value); err == nil && value != "" {
			return conv.Truncate(value, 80)
		}
	}
	return ""
}
