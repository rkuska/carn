package codex

import (
	"bytes"
	"encoding/json"

	"github.com/buger/jsonparser"

	src "github.com/rkuska/carn/internal/source"
)

var knownEventTypes = map[string]struct{}{
	eventTypeTokenCount:     {},
	eventTypeUserMessage:    {},
	eventTypeAgentMessage:   {},
	eventTypeAgentReasoning: {},
	eventTypeItemCompleted:  {},
	eventTypeTaskComplete:   {},
}

var knownUserMessageFields = map[string]struct{}{
	"type":    {},
	"message": {},
}

var knownAgentMessageFields = map[string]struct{}{
	"type":    {},
	"phase":   {},
	"message": {},
}

var knownAgentReasoningFields = map[string]struct{}{
	"type": {},
	"text": {},
}

var knownItemCompletedFields = map[string]struct{}{
	"type": {},
	"item": {},
}

var knownCompletedItemFields = map[string]struct{}{
	"type": {},
	"id":   {},
	"text": {},
}

var knownCompletedItemTypes = map[string]struct{}{
	eventItemTypePlan: {},
}

var knownTaskCompleteFields = map[string]struct{}{
	"type":               {},
	"last_agent_message": {},
}

var knownTokenCountFields = map[string]struct{}{
	"type": {},
	"info": {},
}

var knownTokenCountInfoFields = map[string]struct{}{
	"total_token_usage": {},
	"last_token_usage":  {},
}

var knownTokenUsageFields = map[string]struct{}{
	"input_tokens":            {},
	"cached_input_tokens":     {},
	"output_tokens":           {},
	"reasoning_output_tokens": {},
	"total_tokens":            {},
}

var phaseFieldMarker = []byte(`"phase"`)

func detectEventPayloadDrift(payload []byte, report *src.DriftReport) {
	eventTypeRaw, _ := extractTopLevelRawJSONStringByMarker(payload, typeFieldMarker)
	detectEventTypeDrift(eventTypeRaw, report)

	switch {
	case bytes.Equal(eventTypeRaw, eventTypeUserMessageRaw):
		recordUnknownTopLevelFields(report, "user_message_field", payload, isKnownUserMessageField)
	case bytes.Equal(eventTypeRaw, eventTypeAgentMessageRaw):
		recordUnknownTopLevelFields(report, "agent_message_field", payload, isKnownAgentMessageField)
	case bytes.Equal(eventTypeRaw, eventTypeAgentReasoningRaw):
		recordUnknownTopLevelFields(report, "agent_reasoning_field", payload, isKnownAgentReasoningField)
	case bytes.Equal(eventTypeRaw, eventTypeItemCompletedRaw):
		detectItemCompletedPayloadDrift(payload, report)
	case bytes.Equal(eventTypeRaw, eventTypeTaskCompleteRaw):
		recordUnknownTopLevelFields(report, "task_complete_field", payload, isKnownTaskCompleteField)
	case bytes.Equal(eventTypeRaw, eventTypeTokenCountRaw):
		detectTokenCountPayloadDrift(payload, report)
	}
}

func detectItemCompletedPayloadDrift(payload []byte, report *src.DriftReport) {
	recordUnknownTopLevelFields(report, "item_completed_field", payload, isKnownItemCompletedField)
	if item, ok := extractTopLevelRawJSONFieldByMarker(payload, itemFieldMarker); ok {
		recordUnknownTopLevelFields(report, "completed_item_field", item, isKnownCompletedItemField)
		itemTypeRaw, _ := extractTopLevelRawJSONStringByMarker(item, typeFieldMarker)
		recordUnknownValue(report, "completed_item_type", itemTypeRaw, isKnownCompletedItemTypeRaw)
	}
}

func detectTokenCountPayloadDrift(payload []byte, report *src.DriftReport) {
	recordUnknownTopLevelFields(report, "token_count_field", payload, isKnownTokenCountField)
	if info, ok := extractTopLevelRawJSONFieldByMarker(payload, infoFieldMarker); ok {
		recordUnknownTopLevelFields(report, "token_count_info_field", info, isKnownTokenCountInfoField)
		detectTokenUsageFieldDrift(info, totalTokenUsageFieldMarker, report)
		detectTokenUsageFieldDrift(info, lastTokenUsageFieldMarker, report)
	}
}

func detectTokenUsageFieldDrift(info []byte, fieldMarker []byte, report *src.DriftReport) {
	usage, ok := extractTopLevelRawJSONFieldByMarker(info, fieldMarker)
	if !ok {
		return
	}
	recordUnknownTopLevelFields(report, "token_usage_field", usage, isKnownTokenUsageField)
}

func detectReasoningSummaryBlockDrift(summary json.RawMessage, report *src.DriftReport) {
	summary = bytes.TrimSpace(summary)
	if len(summary) == 0 {
		return
	}

	switch summary[0] {
	case '[':
		_, err := jsonparser.ArrayEach(summary, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
			if err != nil || dataType != jsonparser.Object {
				return
			}
			detectReasoningSummaryObjectDrift(value, report)
		})
		if err != nil {
			return
		}
	case '{':
		detectReasoningSummaryObjectDrift(summary, report)
	}
}

func detectReasoningSummaryObjectDrift(raw []byte, report *src.DriftReport) {
	blockTypeRaw, _ := extractTopLevelRawJSONStringByMarker(raw, typeFieldMarker)
	recordUnknownValue(
		report,
		"reasoning_summary_block_type",
		blockTypeRaw,
		isKnownReasoningSummaryBlockTypeRaw,
	)
}

func isKnownUserMessageField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) || bytes.Equal(field, messageFieldMarker)
}

func isKnownAgentMessageField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, phaseFieldMarker) ||
		bytes.Equal(field, messageFieldMarker)
}

func isKnownAgentReasoningField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) || bytes.Equal(field, textFieldMarker)
}

func isKnownItemCompletedField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) || bytes.Equal(field, itemFieldMarker)
}

func isKnownCompletedItemField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, idFieldMarker) ||
		bytes.Equal(field, textFieldMarker)
}

func isKnownTaskCompleteField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) || bytes.Equal(field, lastAgentMessageFieldMarker)
}

func isKnownTokenCountField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) || bytes.Equal(field, infoFieldMarker)
}

func isKnownTokenCountInfoField(field []byte) bool {
	return bytes.Equal(field, totalTokenUsageFieldMarker) ||
		bytes.Equal(field, lastTokenUsageFieldMarker)
}

func isKnownTokenUsageField(field []byte) bool {
	return bytes.Equal(field, inputTokensFieldMarker) ||
		bytes.Equal(field, cachedInputTokensFieldMarker) ||
		bytes.Equal(field, outputTokensFieldMarker) ||
		bytes.Equal(field, reasoningTokensFieldMarker) ||
		bytes.Equal(field, totalTokensFieldMarker)
}
