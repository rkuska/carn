package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

var readerPool = sync.Pool{
	New: func() any { return bufio.NewReaderSize(nil, codexScanBufferSize) },
}

// codexRecord is a flat struct that merges the outer JSONL envelope with the
// nested payload object. A single json.Decode fills both levels, eliminating
// the intermediate json.RawMessage copy that the old recordEnvelope required.
type codexRecord struct {
	Timestamp string       `json:"timestamp"`
	Type      string       `json:"type"`
	Payload   codexPayload `json:"payload"`
}

// codexPayload merges all payload types into a single struct. The decoder
// populates only the fields present in each JSON record; the rest remain zero.
// The envelope-level Type discriminator selects which fields are meaningful.
type codexPayload struct {
	// Discriminator for response_item and event_msg subtypes.
	ItemType string `json:"type"`

	// session_meta fields.
	ID            string          `json:"id"`
	PayloadTS     string          `json:"timestamp"`
	CWD           string          `json:"cwd"`
	CLIVersion    string          `json:"cli_version"`
	ModelProvider string          `json:"model_provider"`
	Source        json.RawMessage `json:"source"`
	Git           struct {
		Branch string `json:"branch"`
	} `json:"git"`

	// turn_context (CWD reused from session_meta).
	Model string `json:"model"`

	// response_item fields.
	Role             string         `json:"role"`
	Name             string         `json:"name"`
	Arguments        string         `json:"arguments"`
	CallID           string         `json:"call_id"`
	Output           string         `json:"output"`
	Input            string         `json:"input"`
	Status           string         `json:"status"`
	EncryptedContent string         `json:"encrypted_content"`
	Content          []contentBlock `json:"content"`
	Summary          []summaryBlock `json:"summary"`
	Action           struct {
		Query   string   `json:"query"`
		Queries []string `json:"queries"`
	} `json:"action"`

	// event_msg fields.
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

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type summaryBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
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

// parseContext holds a reusable codexRecord to avoid per-line heap allocations.
// The rec struct is zeroed before each decode via reset().
type parseContext struct {
	rec codexRecord
}

func (pc *parseContext) reset() {
	pc.rec = codexRecord{}
}

func openReader(path string) (*os.File, *bufio.Reader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("os.Open: %w", err)
	}
	br := readerPool.Get().(*bufio.Reader)
	br.Reset(file)
	return file, br, nil
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

func usageFromPayload(p *codexPayload) conv.TokenUsage {
	usage := p.Info.TotalTokenUsage
	return conv.TokenUsage{
		InputTokens:          usage.InputTokens,
		CacheReadInputTokens: usage.CachedInputTokens,
		OutputTokens:         usage.OutputTokens + usage.ReasoningOutputTokens,
	}
}

func buildToolCall(p *codexPayload) conv.ToolCall {
	name := p.Name
	if p.ItemType == responseTypeWebSearchCall {
		name = "web_search"
	}
	return conv.ToolCall{
		Name:    name,
		Summary: buildToolSummary(p),
	}
}

func buildToolSummary(p *codexPayload) string {
	if p.ItemType == responseTypeWebSearchCall {
		if p.Action.Query != "" {
			return p.Action.Query
		}
		if len(p.Action.Queries) > 0 {
			return p.Action.Queries[0]
		}
		return ""
	}

	if p.Arguments != "" {
		if cmd, ok := extractJSONStringField(p.Arguments, "cmd"); ok {
			return cmd
		}
	}

	if p.Name == toolNameApplyPatch {
		return "apply patch"
	}
	return ""
}

func buildToolResult(p *codexPayload, meta toolEventMeta) conv.ToolResult {
	return conv.ToolResult{
		ToolName:        meta.call.Name,
		ToolSummary:     meta.call.Summary,
		Content:         p.Output,
		IsError:         p.Status == "failed" || p.Status == "error" || isCodexToolError(p.Output),
		StructuredPatch: parseStructuredPatch(meta.input),
	}
}

func extractJSONStringField(jsonStr, field string) (string, bool) {
	marker := `"` + field + `":"`
	idx := strings.Index(jsonStr, marker)
	if idx == -1 {
		return "", false
	}
	start := idx + len(marker)
	for i := start; i < len(jsonStr); i++ {
		if jsonStr[i] == '\\' {
			i++ // skip escaped character
			continue
		}
		if jsonStr[i] == '"' {
			return jsonStr[start:i], true
		}
	}
	return "", false
}

func isCodexToolError(output string) bool {
	check := output
	if len(check) > 200 {
		check = check[:200]
	}
	lower := strings.ToLower(check)
	return strings.Contains(lower, "aborted by user") ||
		strings.Contains(lower, "patch rejected") ||
		strings.Contains(lower, "verification failed")
}

func isJSONLExt(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".jsonl")
}
