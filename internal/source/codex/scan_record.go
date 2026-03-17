package codex

import (
	"encoding/json"

	conv "github.com/rkuska/carn/internal/conversation"
)

type scanRecord struct {
	Timestamp string      `json:"timestamp"`
	Type      string      `json:"type"`
	Payload   scanPayload `json:"payload"`
}

type scanPayload struct {
	ItemType string `json:"type"`

	ID            string          `json:"id"`
	PayloadTS     string          `json:"timestamp"`
	CWD           string          `json:"cwd"`
	CLIVersion    string          `json:"cli_version"`
	ModelProvider string          `json:"model_provider"`
	Source        json.RawMessage `json:"source"`
	Git           struct {
		Branch string `json:"branch"`
	} `json:"git"`

	Model string `json:"model"`

	Role    string         `json:"role"`
	Name    string         `json:"name"`
	Content []contentBlock `json:"content"`

	Message          string `json:"message"`
	LastAgentMessage string `json:"last_agent_message"`

	Info struct {
		TotalTokenUsage struct {
			InputTokens           int `json:"input_tokens"`
			CachedInputTokens     int `json:"cached_input_tokens"`
			OutputTokens          int `json:"output_tokens"`
			ReasoningOutputTokens int `json:"reasoning_output_tokens"`
		} `json:"total_token_usage"`
	} `json:"info"`
}

type scanContext struct {
	rec scanRecord
}

func (pc *scanContext) reset() {
	pc.rec = scanRecord{}
}

func usageFromScanPayload(p *scanPayload) conv.TokenUsage {
	usage := p.Info.TotalTokenUsage
	return conv.TokenUsage{
		InputTokens:          usage.InputTokens,
		CacheReadInputTokens: usage.CachedInputTokens,
		OutputTokens:         usage.OutputTokens + usage.ReasoningOutputTokens,
	}
}

func scanToolName(p *scanPayload) string {
	if p.ItemType == responseTypeWebSearchCall {
		return "web_search"
	}
	return p.Name
}
