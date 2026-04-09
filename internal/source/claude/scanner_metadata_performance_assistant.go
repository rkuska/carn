package claude

import (
	"bytes"
	"encoding/json"

	"github.com/buger/jsonparser"
)

func accumulateAssistantPerformanceStats(line []byte, stats *scanStats) {
	contentRaw, ok := extractFirstContentValue(line)
	if ok {
		appendAssistantPerformanceContent(contentRaw, stats)
	}
	if stopReason := extractAssistantStopReason(line); stopReason != "" {
		addCount(&stats.performance.StopReasonCounts, stopReason, 1)
	}

	usageRaw, usageOK, err := jsonRawField(line, "message", "usage")
	if err != nil || !usageOK {
		return
	}
	accumulateAssistantUsagePerformance(usageRaw, &stats.performance)
}

func appendAssistantPerformanceContent(contentRaw []byte, stats *scanStats) {
	_ = scanAssistantContentPerformance(contentRaw, stats)
}

func extractAssistantStopReason(line []byte) string {
	stopReason, _, err := jsonStringField(line, "message", "stop_reason")
	if err != nil {
		return ""
	}
	return stopReason
}

func scanAssistantContentPerformance(raw json.RawMessage, stats *scanStats) bool {
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
		if scanErr := scanAssistantPerformanceBlock(value, stats); scanErr != nil {
			parseOK = false
		}
	})
	if err != nil || !parseOK {
		return false
	}
	return true
}

func scanAssistantPerformanceBlock(
	value []byte,
	stats *scanStats,
) error {
	blockTypeRaw, ok := extractObjectStringFieldRaw(value, envelopeTypeFieldKey)
	if !ok {
		return errMissingJSONField
	}

	switch {
	case bytes.Equal(blockTypeRaw, assistantBlockTypeThinkingRaw):
		stats.performance.ReasoningBlockCount++
		if isHiddenAssistantThinkingBlock(value) {
			stats.performance.ReasoningRedactionCount++
		}
	case bytes.Equal(blockTypeRaw, assistantBlockTypeToolUseRaw):
		call, toolUseID, ok := scanAssistantToolUsePerformance(value)
		if !ok {
			return errMissingJSONField
		}
		stats.recordToolCall(call, toolUseID)
	}
	return nil
}

func isHiddenAssistantThinkingBlock(value []byte) bool {
	thinkingRaw, ok := extractObjectStringFieldRaw(value, assistantFieldThinkingKey)
	if !ok {
		return false
	}
	if inner := rawJSONStringInnerValue(thinkingRaw); inner != nil {
		if len(inner) > 0 {
			return false
		}
		_, ok = extractObjectStringFieldRaw(value, assistantFieldSignatureKey)
		return ok
	}
	thinking, ok := decodeJSONStringFast(thinkingRaw)
	if ok && thinking != "" {
		return false
	}
	_, ok = extractObjectStringFieldRaw(value, assistantFieldSignatureKey)
	return ok
}

func scanAssistantToolUsePerformance(value []byte) (toolCall, string, bool) {
	nameRaw, ok := extractObjectStringFieldRaw(value, assistantFieldNameKey)
	if !ok {
		return toolCall{}, "", false
	}
	name := internClaudeToolNameRaw(nameRaw)
	if name == "" {
		return toolCall{}, "", false
	}
	idRaw, ok := extractObjectStringFieldRaw(value, assistantFieldIDKey)
	if !ok {
		return toolCall{}, "", false
	}
	toolUseID, ok := decodeJSONStringFast(idRaw)
	if !ok {
		return toolCall{}, "", false
	}
	input, _ := extractObjectFieldRaw(value, assistantFieldInputKey)
	return toolCall{
		Name:   name,
		Action: classifyClaudeToolActionMetadata(name, input),
	}, toolUseID, true
}

func accumulateAssistantUsagePerformance(raw json.RawMessage, performance *sessionPerformanceMeta) {
	serverToolUseRaw, ok, err := jsonRawField(raw, "server_tool_use")
	if err == nil && ok {
		accumulateServerToolUseCounts(serverToolUseRaw, performance)
	}

	serviceTier, _, err := jsonStringField(raw, "service_tier")
	if err == nil && serviceTier != "" {
		addCount(&performance.ServiceTierCounts, serviceTier, 1)
	}

	speed, _, err := jsonStringField(raw, "speed")
	if err == nil && speed != "" {
		addCount(&performance.SpeedCounts, speed, 1)
	}
}

func accumulateServerToolUseCounts(raw json.RawMessage, performance *sessionPerformanceMeta) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return
	}

	if err := jsonparser.ObjectEach(
		raw,
		func(keyData []byte, valueData []byte, dataType jsonparser.ValueType, _ int) error {
			if dataType != jsonparser.Number {
				return nil
			}
			value, err := jsonparser.ParseInt(valueData)
			if err != nil || value == 0 {
				return nil
			}
			addCount(&performance.ServerToolUseCounts, string(keyData), int(value))
			return nil
		},
	); err != nil {
		return
	}
}
