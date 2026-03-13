package claude

import (
	"encoding/json"
	"fmt"
	"time"
)

func initSessionMeta(meta *sessionMeta, rec jsonRecord) {
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
		meta.Project.DisplayName = displayNameFromCWD(rec.CWD)
	}
}

func applyUserMetadata(meta *sessionMeta, rec jsonRecord) {
	if meta.ID == "" {
		initSessionMeta(meta, rec)
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
		return false, nil
	}

	var rec jsonRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return false, fmt.Errorf("json.Unmarshal: %w", err)
	}

	applyUserMetadata(meta, rec)
	if rec.IsMeta {
		return false, nil
	}

	var msg jsonMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return false, fmt.Errorf("json.Unmarshal message: %w", err)
	}

	content, toolResults := extractUserContent(msg.Content)
	hasContent := len(toolResults) > 0 || isUserContentText(content)
	if !*found && isUserContentText(content) {
		meta.FirstMessage = truncate(content, maxFirstMessage)
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

	var rec jsonRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return false, fmt.Errorf("json.Unmarshal: %w", err)
	}

	var msg jsonMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return false, fmt.Errorf("json.Unmarshal message: %w", err)
	}

	if !*found && msg.Model != "" {
		meta.Model = msg.Model
		*found = true
	}

	return assistantContentHasConversationContent(msg.Content), nil
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
		case "thinking":
			if block.Thinking != "" {
				return true
			}
		case "tool_use":
			return true
		}
	}

	return false
}
