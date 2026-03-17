package codex

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/buger/jsonparser"

	conv "github.com/rkuska/carn/internal/conversation"
)

var readerPool = sync.Pool{
	New: func() any { return bufio.NewReaderSize(nil, codexScanBufferSize) },
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

func openReader(path string) (*os.File, *bufio.Reader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("os.Open: %w", err)
	}
	br, ok := readerPool.Get().(*bufio.Reader)
	if !ok {
		br = bufio.NewReaderSize(nil, codexScanBufferSize)
	}
	br.Reset(file)
	return file, br, nil
}

func isJSONLExt(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".jsonl")
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

func extractReasoningText(raw []byte) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return ""
	}

	switch raw[0] {
	case '"':
		return extractReasoningStringText(raw)
	case '{':
		return extractReasoningBlockText(raw)
	case '[':
		return extractReasoningArrayText(raw)
	default:
		return ""
	}
}

func extractReasoningStringText(raw []byte) string {
	text, ok := decodeRawJSONString(raw)
	if !ok || strings.TrimSpace(text) == "" {
		return ""
	}
	return text
}

func extractReasoningArrayText(raw []byte) string {
	parts := make([]string, 0, 2)
	parseOK := true
	_, err := jsonparser.ArrayEach(raw, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if !parseOK {
			return
		}
		if err != nil || dataType != jsonparser.Object {
			parseOK = false
			return
		}
		if text := extractReasoningBlockText(value); text != "" {
			parts = append(parts, text)
		}
	})
	if err != nil || !parseOK {
		return ""
	}
	return strings.Join(parts, "\n")
}

func extractReasoningBlockText(raw []byte) string {
	text, err := jsonparser.GetString(raw, "text")
	if err != nil || strings.TrimSpace(text) == "" {
		return ""
	}
	return text
}

func buildToolCall(itemType string, payload []byte) conv.ToolCall {
	name, _ := extractTopLevelRawJSONStringFieldByMarker(payload, nameFieldMarker)
	if itemType == responseTypeWebSearchCall {
		name = "web_search"
	}
	return conv.ToolCall{
		Name:    name,
		Summary: buildToolSummary(itemType, name, payload),
	}
}

func buildToolSummary(itemType, name string, payload []byte) string {
	if itemType == responseTypeWebSearchCall {
		action, ok := extractTopLevelRawJSONFieldByMarker(payload, actionFieldMarker)
		if !ok {
			return ""
		}
		if query, ok := extractTopLevelRawJSONStringFieldByMarker(action, queryFieldMarker); ok {
			return query
		}
		if queries, ok := extractTopLevelRawJSONFieldByMarker(action, queriesFieldMarker); ok {
			return extractFirstJSONString(queries)
		}
		return ""
	}

	if arguments, ok := extractTopLevelRawJSONStringFieldByMarker(payload, argumentsFieldMarker); ok && arguments != "" {
		if cmd, ok := extractJSONStringField(arguments, "cmd"); ok {
			return cmd
		}
	}

	if name == toolNameApplyPatch {
		return "apply patch"
	}
	return ""
}

func buildToolResult(payload []byte, meta toolEventMeta) conv.ToolResult {
	output, _ := extractTopLevelRawJSONStringFieldByMarker(payload, outputFieldMarker)
	status, _ := extractTopLevelRawJSONStringFieldByMarker(payload, statusFieldMarker)
	return conv.ToolResult{
		ToolName:        meta.call.Name,
		ToolSummary:     meta.call.Summary,
		Content:         output,
		IsError:         status == "failed" || status == "error" || isCodexToolError(output),
		StructuredPatch: parseStructuredPatch(meta.input),
	}
}

func extractFirstJSONString(raw []byte) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return ""
	}

	first := ""
	parseOK := true
	_, err := jsonparser.ArrayEach(raw, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if first != "" || !parseOK {
			return
		}
		if err != nil || dataType != jsonparser.String {
			parseOK = false
			return
		}
		first, parseOK = decodeJSONParserStringValue(value)
	})
	if err != nil || !parseOK {
		return ""
	}
	return first
}

func decodeRawJSONString(raw []byte) (string, bool) {
	if len(raw) < 2 || raw[0] != '"' {
		return "", false
	}
	value, err := strconv.Unquote(string(raw))
	if err != nil {
		return "", false
	}
	return value, true
}

func decodeJSONParserStringValue(raw []byte) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	if raw[0] == '"' {
		return decodeRawJSONString(raw)
	}
	value, err := strconv.Unquote(`"` + string(raw) + `"`)
	if err != nil {
		return "", false
	}
	return value, true
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
			i++
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
