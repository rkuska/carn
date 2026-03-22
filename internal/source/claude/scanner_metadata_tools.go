package claude

import (
	"bytes"
	"encoding/json"

	"github.com/buger/jsonparser"
)

var (
	assistantToolUseMarker    = []byte(`"tool_use"`)
	userToolResultMarker      = []byte(`"tool_result"`)
	userToolResultErrorMarker = []byte(`"is_error"`)
)

func visitAssistantToolUses(raw json.RawMessage, yield func(name, id string) bool) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return false
	}

	parseOK := true
	_, err := jsonparser.ArrayEach(raw, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if !parseOK {
			return
		}
		if err != nil || dataType != jsonparser.Object {
			parseOK = false
			return
		}
		blockType, ok, err := jsonStringField(value, "type")
		if err != nil {
			parseOK = false
			return
		}
		if !ok {
			return
		}
		if blockType != blockTypeToolUse {
			return
		}

		name, ok, err := jsonStringField(value, "name")
		if err != nil {
			parseOK = false
			return
		}
		if !ok || name == "" {
			return
		}
		id, _, err := jsonStringField(value, "id")
		if err != nil {
			parseOK = false
			return
		}
		if !yield(name, id) {
			parseOK = false
		}
	})
	return err == nil && parseOK
}

func visitUserToolErrors(raw json.RawMessage, yield func(toolUseID string) bool) bool {
	if !bytes.Contains(raw, userToolResultMarker) || !bytes.Contains(raw, userToolResultErrorMarker) {
		return true
	}

	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return false
	}

	parseOK := true
	_, err := jsonparser.ArrayEach(raw, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if !parseOK {
			return
		}
		if err != nil || dataType != jsonparser.Object {
			parseOK = false
			return
		}
		blockType, ok, err := jsonStringField(value, "type")
		if err != nil {
			parseOK = false
			return
		}
		if !ok {
			return
		}
		if blockType != contentTypeToolResult {
			return
		}
		isError, ok, err := jsonBoolField(value, "is_error")
		if err != nil {
			parseOK = false
			return
		}
		if !ok || !isError {
			return
		}

		toolUseID, ok, err := jsonStringField(value, "tool_use_id")
		if err != nil {
			parseOK = false
			return
		}
		if !ok || toolUseID == "" {
			return
		}
		if !yield(toolUseID) {
			parseOK = false
		}
	})
	return err == nil && parseOK
}
