package main

import (
	"context"
	"encoding/json"
)

func userRecordHasConversationContent(line []byte) bool {
	var rec jsonRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return false
	}
	if rec.IsMeta {
		return false
	}

	var msg jsonMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return false
	}

	content, toolResults := extractUserContent(msg.Content)
	if len(toolResults) > 0 {
		return true
	}

	return content != "" && !isSystemInterrupt(content)
}

func assistantRecordHasConversationContent(ctx context.Context, line []byte) bool {
	_, ok := parseAssistantMessage(ctx, line)
	return ok
}
