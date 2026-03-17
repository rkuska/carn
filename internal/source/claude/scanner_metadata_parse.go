package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

// metadataScanRecord flattens the outer JSONL record and the nested message
// object into a single struct for one-pass JSON decoding. This avoids the
// double unmarshal that jsonRecord + jsonMessage required.
type metadataScanRecord struct {
	Type        string `json:"type"`
	SessionID   string `json:"sessionId"`
	Slug        string `json:"slug"`
	CWD         string `json:"cwd"`
	GitBranch   string `json:"gitBranch"`
	Version     string `json:"version"`
	Timestamp   string `json:"timestamp"`
	IsMeta      bool   `json:"isMeta"`
	IsSidechain bool   `json:"isSidechain"`
	Message     struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
		Model   string          `json:"model"`
		Usage   *jsonUsage      `json:"usage"`
	} `json:"message"`
}

func initSessionMetaFromRecord(meta *sessionMeta, rec metadataScanRecord) {
	meta.ID = rec.SessionID
	meta.Slug = rec.Slug
	meta.CWD = rec.CWD
	meta.GitBranch = rec.GitBranch
	meta.Version = rec.Version
	if rec.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339Nano, rec.Timestamp); err == nil {
			meta.Timestamp = t
		}
	}
	if rec.CWD != "" {
		meta.Project.DisplayName = conv.CompactCWD(rec.CWD)
	}
}

func applyUserMetadata(meta *sessionMeta, rec metadataScanRecord) {
	if meta.ID == "" {
		initSessionMetaFromRecord(meta, rec)
	}
	if meta.Slug == "" && rec.Slug != "" {
		meta.Slug = rec.Slug
	}
}

func isUserContentText(content string) bool {
	if content == "" {
		return false
	}
	_, visibility := classifyUserText(content)
	return visibility == ""
}

func parseUserRecord(line []byte, meta *sessionMeta, found *bool) (bool, error) {
	if *found && meta.Slug != "" {
		return userRecordHasConversationContent(line), nil
	}

	var rec metadataScanRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return false, fmt.Errorf("json.Unmarshal: %w", err)
	}

	applyUserMetadata(meta, rec)
	if rec.IsMeta {
		return false, nil
	}

	content, toolResults := extractUserContent(rec.Message.Content)
	hasContent := len(toolResults) > 0 || isUserContentText(content)
	if !*found && isUserContentText(content) {
		meta.FirstMessage = conv.Truncate(content, maxFirstMessage)
		*found = true
	}
	return hasContent, nil
}

func parseAssistantRecord(
	line []byte,
	meta *sessionMeta,
	found *bool,
	hasConversationContent bool,
) (bool, error) {
	if *found && hasConversationContent {
		return false, nil
	}

	var rec metadataScanRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return false, fmt.Errorf("json.Unmarshal: %w", err)
	}

	if !*found && rec.Message.Model != "" {
		meta.Model = rec.Message.Model
		*found = true
	}

	return assistantContentHasConversationContent(rec.Message.Content), nil
}

func assistantContentHasConversationContent(raw json.RawMessage) bool {
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return false
	}

	for _, block := range blocks {
		switch block.Type {
		case blockTypeText:
			if block.Text != "" {
				return true
			}
		case blockTypeThinking:
			if block.Thinking != "" {
				return true
			}
		case blockTypeToolUse:
			return true
		}
	}

	return false
}

func userRecordHasConversationContent(line []byte) bool {
	content, ok := extractFirstContentValue(line)
	if !ok {
		return false
	}
	content = bytes.TrimSpace(content)
	if len(content) == 0 {
		return false
	}
	if content[0] == '"' {
		text, ok := decodeJSONStringFast(content)
		if !ok {
			return false
		}
		return isUserContentText(text)
	}
	return bytes.Contains(content, []byte(`"type":"tool_result"`)) ||
		bytes.Contains(content, []byte(`"type":"text"`))
}

func extractFirstContentValue(line []byte) ([]byte, bool) {
	const marker = `"content":`
	start := bytes.Index(line, []byte(marker))
	if start == -1 {
		return nil, false
	}
	start += len(marker)
	for start < len(line) && line[start] == ' ' {
		start++
	}
	if start >= len(line) {
		return nil, false
	}
	end := jsonValueEnd(line, start)
	if end == -1 {
		return nil, false
	}
	return line[start:end], true
}
