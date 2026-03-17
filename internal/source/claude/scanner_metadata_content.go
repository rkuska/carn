package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/buger/jsonparser"

	conv "github.com/rkuska/carn/internal/conversation"
)

func extractUserContentWithToolUseIDs(raw json.RawMessage) (string, []toolResult, []string) {
	plain, ok := extractJSONStringOrEmpty(raw)
	if ok {
		return plain, nil, nil
	}

	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return "", nil, nil
	}

	var textJ blockJoiner
	var results []toolResult
	var toolUseIDs []string
	parseOK := true
	_, err := jsonparser.ArrayEach(raw, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if !parseOK {
			return
		}
		if err != nil || dataType != jsonparser.Object {
			parseOK = false
			return
		}
		if err := appendUserContentBlock(value, &textJ, &results, &toolUseIDs); err != nil {
			parseOK = false
		}
	})
	if err != nil || !parseOK {
		return "", nil, nil
	}
	return textJ.result(), results, toolUseIDs
}

func extractToolResultContent(raw json.RawMessage) string {
	if plain, ok := extractJSONStringOrEmpty(raw); ok {
		return plain
	}

	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return ""
	}

	var parts blockJoiner
	parseOK := true
	_, err := jsonparser.ArrayEach(raw, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if !parseOK {
			return
		}
		if err != nil || dataType != jsonparser.Object {
			parseOK = false
			return
		}
		if err := appendToolResultTextBlock(value, &parts); err != nil {
			parseOK = false
		}
	})
	if err != nil || !parseOK {
		return ""
	}
	return parts.result()
}

func extractStructuredPatch(raw json.RawMessage) []diffHunk {
	if len(raw) == 0 {
		return nil
	}

	patchRaw, ok, err := jsonRawField(raw, "structuredPatch")
	if err != nil || !ok {
		return nil
	}
	patchRaw = bytes.TrimSpace(patchRaw)
	if len(patchRaw) == 0 || patchRaw[0] != '[' {
		return nil
	}
	return extractStructuredPatchHunks(patchRaw)
}

func extractStructuredPatchHunks(raw json.RawMessage) []diffHunk {
	hunks := make([]diffHunk, 0, 1)
	parseOK := true
	_, err := jsonparser.ArrayEach(raw, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if !parseOK {
			return
		}
		if err != nil || dataType != jsonparser.Object {
			parseOK = false
			return
		}
		hunk, err := extractStructuredPatchHunk(value)
		if err != nil {
			parseOK = false
			return
		}
		hunks = append(hunks, hunk)
	})
	if err != nil || !parseOK || len(hunks) == 0 {
		return nil
	}
	return hunks
}

func extractJSONStringOrEmpty(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '"' {
		return "", false
	}
	plain, ok := decodeJSONStringFast(raw)
	if !ok {
		return "", false
	}
	return plain, true
}

func appendUserContentBlock(
	value []byte,
	textJ *blockJoiner,
	results *[]toolResult,
	toolUseIDs *[]string,
) error {
	blockType, _, err := jsonStringField(value, "type")
	if err != nil {
		return fmt.Errorf("appendUserContentBlock_type: %w", err)
	}

	switch blockType {
	case blockTypeText:
		return appendUserTextBlock(value, textJ)
	case contentTypeToolResult:
		return appendUserToolResultBlock(value, results, toolUseIDs)
	default:
		return nil
	}
}

func appendUserTextBlock(value []byte, textJ *blockJoiner) error {
	blockText, _, err := jsonStringField(value, "text")
	if err != nil {
		return fmt.Errorf("appendUserTextBlock_text: %w", err)
	}
	if blockText != "" {
		textJ.add(blockText)
	}
	return nil
}

func appendUserToolResultBlock(
	value []byte,
	results *[]toolResult,
	toolUseIDs *[]string,
) error {
	contentRaw, _, err := jsonRawField(value, "content")
	if err != nil {
		return fmt.Errorf("appendUserToolResultBlock_content: %w", err)
	}
	content := extractToolResultContent(contentRaw)
	if content == "" {
		return nil
	}

	isError, _, err := jsonBoolField(value, "is_error")
	if err != nil {
		return fmt.Errorf("appendUserToolResultBlock_is_error: %w", err)
	}
	toolUseID, _, err := jsonStringField(value, "tool_use_id")
	if err != nil {
		return fmt.Errorf("appendUserToolResultBlock_tool_use_id: %w", err)
	}

	*results = append(*results, toolResult{
		Content: conv.TruncatePreserveNewlines(content, maxToolResultChars),
		IsError: isError,
	})
	*toolUseIDs = append(*toolUseIDs, toolUseID)
	return nil
}

func appendToolResultTextBlock(value []byte, parts *blockJoiner) error {
	blockType, _, err := jsonStringField(value, "type")
	if err != nil {
		return fmt.Errorf("appendToolResultTextBlock_type: %w", err)
	}
	if blockType != blockTypeText {
		return nil
	}

	blockText, _, err := jsonStringField(value, "text")
	if err != nil {
		return fmt.Errorf("appendToolResultTextBlock_text: %w", err)
	}
	if blockText != "" {
		parts.add(blockText)
	}
	return nil
}

func extractStructuredPatchHunk(value []byte) (diffHunk, error) {
	oldStart, _, err := jsonIntField(value, "oldStart")
	if err != nil {
		return diffHunk{}, fmt.Errorf("extractStructuredPatchHunk_oldStart: %w", err)
	}
	oldLines, _, err := jsonIntField(value, "oldLines")
	if err != nil {
		return diffHunk{}, fmt.Errorf("extractStructuredPatchHunk_oldLines: %w", err)
	}
	newStart, _, err := jsonIntField(value, "newStart")
	if err != nil {
		return diffHunk{}, fmt.Errorf("extractStructuredPatchHunk_newStart: %w", err)
	}
	newLines, _, err := jsonIntField(value, "newLines")
	if err != nil {
		return diffHunk{}, fmt.Errorf("extractStructuredPatchHunk_newLines: %w", err)
	}
	linesRaw, _, err := jsonRawField(value, "lines")
	if err != nil {
		return diffHunk{}, fmt.Errorf("extractStructuredPatchHunk_lines: %w", err)
	}

	return diffHunk{
		OldStart: oldStart,
		OldLines: oldLines,
		NewStart: newStart,
		NewLines: newLines,
		Lines:    extractJSONStringArray(linesRaw),
	}, nil
}

func extractJSONStringArray(raw json.RawMessage) []string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return nil
	}

	values := make([]string, 0, 4)
	parseOK := true
	_, err := jsonparser.ArrayEach(raw, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if !parseOK {
			return
		}
		if err != nil || dataType != jsonparser.String {
			parseOK = false
			return
		}
		text, ok := decodeJSONParserStringValue(value)
		if !ok {
			parseOK = false
			return
		}
		values = append(values, text)
	})
	if err != nil || !parseOK {
		return nil
	}
	return values
}

func decodeJSONParserStringValue(raw []byte) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	if raw[0] == '"' {
		return decodeJSONStringFast(raw)
	}
	value, err := strconv.Unquote(`"` + string(raw) + `"`)
	if err != nil {
		return "", false
	}
	return value, true
}
