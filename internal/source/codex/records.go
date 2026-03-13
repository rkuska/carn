package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type recordEnvelope struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type sessionMetaPayload struct {
	ID            string          `json:"id"`
	Timestamp     string          `json:"timestamp"`
	CWD           string          `json:"cwd"`
	CLIVersion    string          `json:"cli_version"`
	ModelProvider string          `json:"model_provider"`
	Source        json.RawMessage `json:"source"`
	Git           struct {
		Branch string `json:"branch"`
	} `json:"git"`
}

type turnContextPayload struct {
	CWD   string `json:"cwd"`
	Model string `json:"model"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type summaryBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responseItemPayload struct {
	Type      string         `json:"type"`
	Role      string         `json:"role"`
	Name      string         `json:"name"`
	Arguments string         `json:"arguments"`
	CallID    string         `json:"call_id"`
	Output    string         `json:"output"`
	Input     string         `json:"input"`
	Status    string         `json:"status"`
	Content   []contentBlock `json:"content"`
	Summary   []summaryBlock `json:"summary"`
	Action    struct {
		Query   string   `json:"query"`
		Queries []string `json:"queries"`
	} `json:"action"`
}

type eventPayload struct {
	Type             string          `json:"type"`
	Message          string          `json:"message"`
	Text             string          `json:"text"`
	LastAgentMessage string          `json:"last_agent_message"`
	Item             json.RawMessage `json:"item"`
	Info             struct {
		TotalTokenUsage struct {
			InputTokens           int `json:"input_tokens"`
			CachedInputTokens     int `json:"cached_input_tokens"`
			OutputTokens          int `json:"output_tokens"`
			ReasoningOutputTokens int `json:"reasoning_output_tokens"`
		} `json:"total_token_usage"`
	} `json:"info"`
}

type toolEventMeta struct {
	call  conv.ToolCall
	input string
}

type completedItemPayload struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Text string `json:"text"`
}

func openScanner(path string) (*os.File, *bufio.Scanner, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("os.Open: %w", err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), codexScanBufferSize)
	return file, scanner, nil
}

func parseEnvelope(line []byte) (recordEnvelope, error) {
	var envelope recordEnvelope
	if err := json.Unmarshal(line, &envelope); err != nil {
		return recordEnvelope{}, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return envelope, nil
}

func parseTimestamp(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func projectNameFromCWD(cwd string) string {
	if cwd == "" {
		return ""
	}
	return filepath.Base(filepath.Clean(cwd))
}

func extractMessageText(blocks []contentBlock) string {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		switch block.Type {
		case "input_text", "output_text":
			if block.Text != "" {
				parts = append(parts, block.Text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func extractReasoningText(blocks []summaryBlock) string {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func usageFromEvent(payload eventPayload) conv.TokenUsage {
	usage := payload.Info.TotalTokenUsage
	return conv.TokenUsage{
		InputTokens:          usage.InputTokens,
		CacheReadInputTokens: usage.CachedInputTokens,
		OutputTokens:         usage.OutputTokens + usage.ReasoningOutputTokens,
	}
}

func sourceIsSubagent(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}

	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return false
	}

	var source map[string]any
	if err := json.Unmarshal(raw, &source); err != nil {
		return false
	}
	_, ok := source["subagent"]
	return ok
}

func buildToolCall(payload responseItemPayload) conv.ToolCall {
	name := payload.Name
	if payload.Type == responseTypeWebSearchCall {
		name = "web_search"
	}
	return conv.ToolCall{
		Name:    name,
		Summary: buildToolSummary(payload),
	}
}

func buildToolSummary(payload responseItemPayload) string {
	if payload.Type == responseTypeWebSearchCall {
		if payload.Action.Query != "" {
			return payload.Action.Query
		}
		if len(payload.Action.Queries) > 0 {
			return payload.Action.Queries[0]
		}
		return ""
	}

	if payload.Arguments != "" {
		var args map[string]any
		if err := json.Unmarshal([]byte(payload.Arguments), &args); err == nil {
			if cmd, ok := args["cmd"].(string); ok {
				return cmd
			}
		}
	}

	if payload.Name == toolNameApplyPatch {
		return "apply patch"
	}
	return ""
}

func buildToolResult(payload responseItemPayload, meta toolEventMeta) conv.ToolResult {
	return conv.ToolResult{
		ToolName:        meta.call.Name,
		ToolSummary:     meta.call.Summary,
		Content:         payload.Output,
		IsError:         payload.Status == "failed" || payload.Status == "error" || isCodexToolError(payload.Output),
		StructuredPatch: parseStructuredPatch(meta.input),
	}
}

func isCodexToolError(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "aborted by user") ||
		strings.Contains(lower, "patch rejected") ||
		strings.Contains(lower, "verification failed")
}

func firstSessionPath(conversation conv.Conversation) string {
	if len(conversation.Sessions) == 0 {
		return ""
	}
	return conversation.Sessions[0].FilePath
}

func isJSONLExt(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".jsonl")
}
