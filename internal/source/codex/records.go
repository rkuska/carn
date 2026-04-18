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
	src "github.com/rkuska/carn/internal/source"
)

var readerPool = sync.Pool{
	New: func() any { return bufio.NewReaderSize(nil, codexScanBufferSize) },
}

var (
	statusFailedRaw        = []byte(`"failed"`)
	statusErrorRaw         = []byte(`"error"`)
	toolNameExecCommandRaw = []byte(`"exec_command"`)
	toolNameApplyPatchRaw  = []byte(`"apply_patch"`)
)

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
		return nil, nil, src.MarkMalformedRawData(fmt.Errorf("os.Open: %w", err))
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

func buildToolCall(itemType string, payload []byte, readEvidence map[string]struct{}) conv.ToolCall {
	name, _ := extractTopLevelRawJSONStringFieldByMarker(payload, nameFieldMarker)
	if itemType == responseTypeWebSearchCall {
		action, _ := extractTopLevelRawJSONFieldByMarker(payload, actionFieldMarker)
		return conv.ToolCall{
			Name:    toolNameWebSearch,
			Summary: buildWebSearchSummaryFromActionRaw(action),
			Action: conv.NormalizedAction{
				Type:    conv.NormalizedActionWeb,
				Targets: webSearchTargetsFromActionRaw(action),
			},
		}
	}

	input := extractToolInputString(payload)
	return conv.ToolCall{
		Name:    name,
		Summary: buildToolSummaryFromInput(name, input),
		Action:  classifyCodexToolActionFromInput(name, input, readEvidence),
	}
}

func buildWebSearchSummaryFromActionRaw(action []byte) string {
	if query, ok := extractTopLevelRawJSONStringFieldByMarker(action, queryFieldMarker); ok {
		return query
	}
	if queries, ok := extractTopLevelRawJSONFieldByMarker(action, queriesFieldMarker); ok {
		return extractFirstJSONString(queries)
	}
	return ""
}

func buildToolSummaryFromInput(name, toolInput string) string {
	if summary, ok := buildCommandSummary(name, toolInput); ok {
		return summary
	}
	if name == toolNameApplyPatch {
		return "apply patch"
	}
	return ""
}

func buildCommandSummary(name, toolInput string) (string, bool) {
	if toolInput == "" {
		return "", false
	}
	if cmd, ok := extractJSONStringField(toolInput, "cmd"); ok {
		return cmd, true
	}
	if isCommandToolName(name) {
		return unwrapCommand(toolInput), true
	}
	return "", false
}

func isCommandToolName(name string) bool {
	switch name {
	case toolNameExecCommand, toolNameShellCommand, toolNameWriteStdin:
		return true
	default:
		return false
	}
}

func buildToolResult(payload []byte, meta toolEventMeta) conv.ToolResult {
	output, _ := extractTopLevelRawJSONStringFieldByMarker(payload, outputFieldMarker)
	status, _ := extractTopLevelRawJSONStringFieldByMarker(payload, statusFieldMarker)
	return conv.ToolResult{
		ToolName:        meta.call.Name,
		ToolSummary:     meta.call.Summary,
		Content:         output,
		IsError:         status == "failed" || status == "error" || isCodexToolError(meta.call.Name, output),
		StructuredPatch: parseStructuredPatch(meta.input),
		Action:          meta.call.Action,
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

func isCodexToolError(toolName, output string) bool {
	if toolName != toolNameApplyPatch {
		return false
	}

	check := output
	if len(check) > 200 {
		check = check[:200]
	}
	lower := strings.ToLower(strings.TrimSpace(check))
	return strings.HasPrefix(lower, "apply_patch verification failed:")
}

func scanToolName(raw []byte) (string, bool) {
	switch {
	case bytes.Equal(raw, toolNameExecCommandRaw):
		return "exec_command", true
	case bytes.Equal(raw, toolNameApplyPatchRaw):
		return toolNameApplyPatch, true
	default:
		return readRawJSONString(raw)
	}
}

func hasCodexToolError(toolName string, outputRaw, statusRaw []byte) bool {
	return bytes.Equal(bytes.TrimSpace(statusRaw), statusFailedRaw) ||
		bytes.Equal(bytes.TrimSpace(statusRaw), statusErrorRaw) ||
		isCodexToolErrorRaw(toolName, outputRaw)
}

func isCodexToolRejectRaw(outputRaw []byte) bool {
	output, ok := readRawJSONString(outputRaw)
	if !ok {
		return false
	}
	return conv.IsRejectedToolResultContent(output)
}

func isCodexToolErrorRaw(toolName string, outputRaw []byte) bool {
	if toolName != toolNameApplyPatch {
		return false
	}

	outputRaw = bytes.TrimSpace(outputRaw)
	if len(outputRaw) < 2 || outputRaw[0] != '"' {
		return false
	}

	check := outputRaw[1:]
	if len(check) > 200 {
		check = check[:200]
	}
	return hasPrefixFoldASCII(check, "apply_patch verification failed:")
}

func hasPrefixFoldASCII(raw []byte, prefix string) bool {
	if len(prefix) == 0 || len(raw) < len(prefix) {
		return false
	}
	return equalFoldASCII(raw[:len(prefix)], prefix)
}

func equalFoldASCII(raw []byte, needle string) bool {
	if len(raw) != len(needle) {
		return false
	}
	for i := range raw {
		if toLowerASCII(raw[i]) != toLowerASCII(needle[i]) {
			return false
		}
	}
	return true
}

func toLowerASCII(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}
	return value
}
