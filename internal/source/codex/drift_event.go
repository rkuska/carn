package codex

import (
	"bytes"
	"encoding/json"

	"github.com/buger/jsonparser"

	src "github.com/rkuska/carn/internal/source"
)

var knownEventTypes = map[string]struct{}{
	eventTypeTokenCount:       {},
	eventTypeUserMessage:      {},
	eventTypeAgentMessage:     {},
	eventTypeAgentReasoning:   {},
	eventTypeItemCompleted:    {},
	eventTypeTaskStarted:      {},
	eventTypeTaskComplete:     {},
	eventTypeTurnAborted:      {},
	eventTypeContextCompacted: {},
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

var knownTaskStartedFields = map[string]struct{}{
	"type":                 {},
	"turn_id":              {},
	"model_context_window": {},
}

var knownTurnAbortedFields = map[string]struct{}{
	"type":    {},
	"turn_id": {},
}

var knownContextCompactedFields = map[string]struct{}{
	"type": {},
}

var knownTokenCountFields = map[string]struct{}{
	"type":        {},
	"rate_limits": {},
	"info":        {},
}

var knownTokenCountInfoFields = map[string]struct{}{
	"total_token_usage":    {},
	"last_token_usage":     {},
	"model_context_window": {},
}

var knownTokenUsageFields = map[string]struct{}{
	"input_tokens":                {},
	"cached_input_tokens":         {},
	"cache_creation_input_tokens": {},
	"output_tokens":               {},
	"reasoning_output_tokens":     {},
	"total_tokens":                {},
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
	case bytes.Equal(eventTypeRaw, eventTypeTaskStartedRaw):
		recordUnknownTopLevelFields(report, "task_started_field", payload, isKnownTaskStartedField)
	case bytes.Equal(eventTypeRaw, eventTypeTaskCompleteRaw):
		recordUnknownTopLevelFields(report, "task_complete_field", payload, isKnownTaskCompleteField)
	case bytes.Equal(eventTypeRaw, eventTypeTurnAbortedRaw):
		recordUnknownTopLevelFields(report, "turn_aborted_field", payload, isKnownTurnAbortedField)
	case bytes.Equal(eventTypeRaw, eventTypeContextCompactedRaw):
		recordUnknownTopLevelFields(report, "context_compacted_field", payload, isKnownContextCompactedField)
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
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, messageFieldMarker) ||
		codexKnownSchemaExtras.HasRaw("user_message_field", field)
}

func isKnownAgentMessageField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, phaseFieldMarker) ||
		bytes.Equal(field, messageFieldMarker) ||
		codexKnownSchemaExtras.HasRaw("agent_message_field", field)
}

func isKnownAgentReasoningField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) || bytes.Equal(field, textFieldMarker)
}

func isKnownItemCompletedField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, itemFieldMarker) ||
		codexKnownSchemaExtras.HasRaw("item_completed_field", field)
}

func isKnownCompletedItemField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, idFieldMarker) ||
		bytes.Equal(field, textFieldMarker)
}

func isKnownTaskCompleteField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, lastAgentMessageFieldMarker) ||
		codexKnownSchemaExtras.HasRaw("task_complete_field", field)
}

func isKnownTaskStartedField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, modelContextWindowFieldMarker) ||
		bytes.Equal(field, []byte(`"turn_id"`)) ||
		codexKnownSchemaExtras.HasRaw("task_started_field", field)
}

func isKnownTurnAbortedField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, []byte(`"turn_id"`)) ||
		codexKnownSchemaExtras.HasRaw("turn_aborted_field", field)
}

func isKnownContextCompactedField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		codexKnownSchemaExtras.HasRaw("context_compacted_field", field)
}

func isKnownTokenCountField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, rateLimitsFieldMarker) ||
		bytes.Equal(field, infoFieldMarker) ||
		codexKnownSchemaExtras.HasRaw("token_count_field", field)
}

func isKnownTokenCountInfoField(field []byte) bool {
	return bytes.Equal(field, totalTokenUsageFieldMarker) ||
		bytes.Equal(field, lastTokenUsageFieldMarker) ||
		bytes.Equal(field, modelContextWindowFieldMarker) ||
		codexKnownSchemaExtras.HasRaw("token_count_info_field", field)
}

func isKnownTokenUsageField(field []byte) bool {
	return bytes.Equal(field, inputTokensFieldMarker) ||
		bytes.Equal(field, cachedInputTokensFieldMarker) ||
		bytes.Equal(field, cacheCreationInputTokensFieldMarker) ||
		bytes.Equal(field, outputTokensFieldMarker) ||
		bytes.Equal(field, reasoningTokensFieldMarker) ||
		bytes.Equal(field, totalTokensFieldMarker)
}
