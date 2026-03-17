package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/buger/jsonparser"

	conv "github.com/rkuska/carn/internal/conversation"
)

func initSessionMetaFromRecord(meta *sessionMeta, rec parseRecord) {
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

func applyUserMetadata(meta *sessionMeta, rec parseRecord) {
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

	var rec parseRecord
	if err := parseRecordLine(line, &rec); err != nil {
		return false, fmt.Errorf("parseRecordLine: %w", err)
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

	var rec parseRecord
	if err := parseRecordLine(line, &rec); err != nil {
		return false, fmt.Errorf("parseRecordLine: %w", err)
	}

	if !*found && rec.Message.Model != "" {
		meta.Model = rec.Message.Model
		*found = true
	}

	return assistantContentHasConversationContent(rec.Message.Content), nil
}

func assistantContentHasConversationContent(raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return false
	}

	hasConversationContent := false
	parseOK := true
	_, err := jsonparser.ArrayEach(raw, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if hasConversationContent || !parseOK {
			return
		}
		if err != nil || dataType != jsonparser.Object {
			parseOK = false
			return
		}
		content, err := assistantContentBlockHasConversationContent(value)
		if err != nil {
			parseOK = false
			return
		}
		hasConversationContent = content
	})
	if err != nil || !parseOK {
		return false
	}
	return hasConversationContent
}

func assistantContentBlockHasConversationContent(value []byte) (bool, error) {
	blockType, _, err := jsonStringField(value, "type")
	if err != nil {
		return false, fmt.Errorf("assistantContentBlockHasConversationContent_type: %w", err)
	}

	switch blockType {
	case blockTypeText:
		return assistantContentStringFieldHasValue(value, "text")
	case blockTypeThinking:
		hasText, err := assistantContentStringFieldHasValue(value, "thinking")
		if err != nil {
			return false, err
		}
		if hasText {
			return true, nil
		}
		// Signed thinking blocks without visible text still count as content.
		return assistantContentStringFieldHasValue(value, "signature")
	case blockTypeToolUse:
		return true, nil
	default:
		return false, nil
	}
}

func assistantContentStringFieldHasValue(value []byte, field string) (bool, error) {
	content, _, err := jsonStringField(value, field)
	if err != nil {
		return false, fmt.Errorf("assistantContentStringFieldHasValue_%s: %w", field, err)
	}
	return content != "", nil
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
