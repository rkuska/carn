package codex

import (
	"bytes"
	"encoding/json"

	"github.com/buger/jsonparser"

	src "github.com/rkuska/carn/internal/source"
)

var knownEnvelopeFields = map[string]struct{}{
	"timestamp": {},
	"type":      {},
	"payload":   {},
}

var knownRecordTypes = map[string]struct{}{
	recordTypeSessionMeta:  {},
	recordTypeTurnContext:  {},
	recordTypeResponseItem: {},
	recordTypeEventMsg:     {},
}

var knownSessionMetaFields = map[string]struct{}{
	"id":             {},
	"timestamp":      {},
	"cwd":            {},
	"originator":     {},
	"cli_version":    {},
	"source":         {},
	"model_provider": {},
	"git":            {},
}

var knownGitFields = map[string]struct{}{
	"branch":      {},
	"commit_hash": {},
}

var knownTurnContextFields = map[string]struct{}{
	"cwd":             {},
	"model":           {},
	"approval_policy": {},
	"sandbox_policy":  {},
}

var knownResponseItemTypes = map[string]struct{}{
	responseTypeMessage:              {},
	responseTypeReasoning:            {},
	responseTypeFunctionCall:         {},
	responseTypeCustomToolCall:       {},
	responseTypeWebSearchCall:        {},
	responseTypeFunctionCallOutput:   {},
	responseTypeCustomToolCallOutput: {},
}

var knownRoles = map[string]struct{}{
	responseRoleUser:      {},
	responseRoleAssistant: {},
	responseRoleDeveloper: {},
}

var knownResponseMessageFields = map[string]struct{}{
	"type":    {},
	"role":    {},
	"content": {},
}

var knownContentBlockTypes = map[string]struct{}{
	"input_text":  {},
	"output_text": {},
}

var knownReasoningFields = map[string]struct{}{
	"type":              {},
	"summary":           {},
	"content":           {},
	"encrypted_content": {},
}

var knownReasoningSummaryBlockTypes = map[string]struct{}{
	"summary_text": {},
}

var knownFunctionCallFields = map[string]struct{}{
	"type":      {},
	"name":      {},
	"arguments": {},
	"call_id":   {},
}

var knownCustomToolCallFields = map[string]struct{}{
	"type":    {},
	"status":  {},
	"call_id": {},
	"name":    {},
	"input":   {},
}

var knownToolCallOutputFields = map[string]struct{}{
	"type":    {},
	"call_id": {},
	"output":  {},
	"status":  {},
}

var knownWebSearchCallFields = map[string]struct{}{
	"type":    {},
	"action":  {},
	"call_id": {},
	"status":  {},
}

var (
	originatorFieldMarker     = []byte(`"originator"`)
	commitHashFieldMarker     = []byte(`"commit_hash"`)
	approvalPolicyFieldMarker = []byte(`"approval_policy"`)
	sandboxPolicyFieldMarker  = []byte(`"sandbox_policy"`)
)

func detectRecordTypeDrift(recordType string, report *src.DriftReport) {
	if recordType == "" {
		return
	}
	if _, ok := knownRecordTypes[recordType]; ok {
		return
	}
	report.Record("record_type", recordType)
}

func detectResponseItemTypeDrift(itemType string, report *src.DriftReport) {
	if itemType == "" {
		return
	}
	if _, ok := knownResponseItemTypes[itemType]; ok {
		return
	}
	report.Record("response_item_type", itemType)
}

func detectEventTypeDrift(eventType string, report *src.DriftReport) {
	if eventType == "" {
		return
	}
	if _, ok := knownEventTypes[eventType]; ok {
		return
	}
	report.Record("event_type", eventType)
}

func detectPayloadFieldDrift(recordType string, payload []byte, report *src.DriftReport) {
	switch recordType {
	case recordTypeSessionMeta:
		recordUnknownTopLevelFields(report, "session_meta_field", payload, isKnownSessionMetaField)
		if git, ok := extractTopLevelRawJSONFieldByMarker(payload, gitFieldMarker); ok {
			recordUnknownTopLevelFields(report, "git_field", git, isKnownGitField)
		}
	case recordTypeTurnContext:
		recordUnknownTopLevelFields(report, "turn_context_field", payload, isKnownTurnContextField)
	case recordTypeResponseItem:
		detectResponseItemPayloadDrift(payload, report)
	case recordTypeEventMsg:
		detectEventPayloadDrift(payload, report)
	}
}

func detectResponseItemPayloadDrift(payload []byte, report *src.DriftReport) {
	itemType, ok := extractTopLevelRawJSONStringFieldByMarker(payload, typeFieldMarker)
	if ok {
		detectResponseItemTypeDrift(itemType, report)
	}

	switch itemType {
	case responseTypeMessage:
		detectResponseMessagePayloadDrift(payload, report)
	case responseTypeReasoning:
		detectReasoningPayloadDrift(payload, report)
	case responseTypeFunctionCall:
		recordUnknownTopLevelFields(report, "function_call_field", payload, isKnownFunctionCallField)
	case responseTypeCustomToolCall:
		recordUnknownTopLevelFields(report, "custom_tool_call_field", payload, isKnownCustomToolCallField)
	case responseTypeFunctionCallOutput, responseTypeCustomToolCallOutput:
		recordUnknownTopLevelFields(report, "tool_call_output_field", payload, isKnownToolCallOutputField)
	case responseTypeWebSearchCall:
		recordUnknownTopLevelFields(report, "web_search_call_field", payload, isKnownWebSearchCallField)
	}
}

func detectResponseMessagePayloadDrift(payload []byte, report *src.DriftReport) {
	recordUnknownTopLevelFields(report, "response_message_field", payload, isKnownResponseMessageField)
	if role, ok := extractTopLevelRawJSONStringFieldByMarker(payload, roleFieldMarker); ok {
		detectRoleDrift(role, report)
	}
	if content, ok := extractTopLevelRawJSONFieldByMarker(payload, contentFieldMarker); ok {
		detectContentBlockDrift(content, report)
	}
}

func detectReasoningPayloadDrift(payload []byte, report *src.DriftReport) {
	recordUnknownTopLevelFields(report, "reasoning_field", payload, isKnownReasoningField)
	if summary, ok := extractTopLevelRawJSONFieldByMarker(payload, summaryFieldMarker); ok {
		detectReasoningSummaryBlockDrift(summary, report)
	}
}

func detectRoleDrift(role string, report *src.DriftReport) {
	if role == "" {
		return
	}
	if _, ok := knownRoles[role]; ok {
		return
	}
	report.Record("role", role)
}

func detectContentBlockDrift(content json.RawMessage, report *src.DriftReport) {
	content = bytes.TrimSpace(content)
	if len(content) == 0 || content[0] != '[' {
		return
	}

	_, err := jsonparser.ArrayEach(content, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if err != nil || dataType != jsonparser.Object {
			return
		}
		blockType, ok := extractTopLevelRawJSONStringFieldByMarker(value, typeFieldMarker)
		if !ok {
			return
		}
		if _, known := knownContentBlockTypes[blockType]; known {
			return
		}
		report.Record("content_block_type", blockType)
	})
	if err != nil {
		return
	}
}

type driftEnvelope struct {
	timestamp  string
	recordType string
	payload    []byte
	hasPayload bool
}

func detectLineDrift(line []byte, report *src.DriftReport) driftEnvelope {
	envelope := scanDriftEnvelope(line, report)
	detectRecordTypeDrift(envelope.recordType, report)
	if !envelope.hasPayload {
		return envelope
	}
	detectPayloadFieldDrift(envelope.recordType, envelope.payload, report)
	return envelope
}

func scanDriftEnvelope(line []byte, report *src.DriftReport) driftEnvelope {
	pos, ok := topLevelObjectStart(line)
	if !ok {
		return driftEnvelope{}
	}

	var envelope driftEnvelope
	for {
		field, valueStart, next, done, ok := nextTopLevelField(line, pos)
		if !ok || done {
			return envelope
		}

		switch {
		case bytes.Equal(field, timestampFieldMarker):
			envelope.timestamp, _ = readDriftString(line, valueStart)
		case bytes.Equal(field, typeFieldMarker):
			envelope.recordType, _ = readDriftString(line, valueStart)
		case bytes.Equal(field, payloadFieldMarker):
			envelope.payload, envelope.hasPayload = sliceRawJSONValue(line, valueStart)
		case isKnownEnvelopeField(field):
		default:
			recordUnknownField(report, "envelope_field", field)
		}

		pos = next
	}
}

func recordUnknownTopLevelFields(
	report *src.DriftReport,
	category string,
	raw []byte,
	known func([]byte) bool,
) {
	pos, ok := topLevelObjectStart(raw)
	if !ok {
		return
	}

	for {
		field, _, next, done, ok := nextTopLevelField(raw, pos)
		if !ok || done {
			return
		}
		if !known(field) {
			recordUnknownField(report, category, field)
		}
		pos = next
	}
}

func readDriftString(raw []byte, start int) (string, bool) {
	if !startsJSONString(raw, start) {
		return "", false
	}
	return readRawJSONStringValue(raw, start)
}

func recordUnknownField(report *src.DriftReport, category string, rawField []byte) {
	if report == nil {
		return
	}
	field, ok := decodeRawJSONString(rawField)
	if !ok {
		return
	}
	report.Record(category, field)
}

func isKnownEnvelopeField(field []byte) bool {
	return bytes.Equal(field, timestampFieldMarker) ||
		bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, payloadFieldMarker)
}

func isKnownSessionMetaField(field []byte) bool {
	return bytes.Equal(field, idFieldMarker) ||
		bytes.Equal(field, timestampFieldMarker) ||
		bytes.Equal(field, cwdFieldMarker) ||
		bytes.Equal(field, originatorFieldMarker) ||
		bytes.Equal(field, cliVersionFieldMarker) ||
		bytes.Equal(field, sourceFieldMarker) ||
		bytes.Equal(field, modelProviderFieldMarker) ||
		bytes.Equal(field, gitFieldMarker)
}

func isKnownGitField(field []byte) bool {
	return bytes.Equal(field, branchFieldMarker) || bytes.Equal(field, commitHashFieldMarker)
}

func isKnownTurnContextField(field []byte) bool {
	return bytes.Equal(field, cwdFieldMarker) ||
		bytes.Equal(field, modelFieldMarker) ||
		bytes.Equal(field, approvalPolicyFieldMarker) ||
		bytes.Equal(field, sandboxPolicyFieldMarker)
}

func isKnownResponseMessageField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, roleFieldMarker) ||
		bytes.Equal(field, contentFieldMarker)
}

func isKnownReasoningField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, summaryFieldMarker) ||
		bytes.Equal(field, contentFieldMarker) ||
		bytes.Equal(field, encryptedContentFieldMarker)
}

func isKnownFunctionCallField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, nameFieldMarker) ||
		bytes.Equal(field, argumentsFieldMarker) ||
		bytes.Equal(field, callIDFieldMarker)
}

func isKnownCustomToolCallField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, statusFieldMarker) ||
		bytes.Equal(field, callIDFieldMarker) ||
		bytes.Equal(field, nameFieldMarker) ||
		bytes.Equal(field, inputFieldMarker)
}

func isKnownToolCallOutputField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, callIDFieldMarker) ||
		bytes.Equal(field, outputFieldMarker) ||
		bytes.Equal(field, statusFieldMarker)
}

func isKnownWebSearchCallField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, actionFieldMarker) ||
		bytes.Equal(field, callIDFieldMarker) ||
		bytes.Equal(field, statusFieldMarker)
}
