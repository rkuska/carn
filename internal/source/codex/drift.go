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

func detectRecordTypeDrift(recordTypeRaw []byte, report *src.DriftReport) {
	recordUnknownValue(report, "record_type", recordTypeRaw, isKnownRecordTypeRaw)
}

func detectResponseItemTypeDrift(itemTypeRaw []byte, report *src.DriftReport) {
	recordUnknownValue(report, "response_item_type", itemTypeRaw, isKnownResponseItemTypeRaw)
}

func detectEventTypeDrift(eventTypeRaw []byte, report *src.DriftReport) {
	recordUnknownValue(report, "event_type", eventTypeRaw, isKnownEventTypeRaw)
}

func detectPayloadFieldDrift(recordTypeRaw []byte, payload []byte, report *src.DriftReport) {
	switch {
	case bytes.Equal(recordTypeRaw, recordTypeSessionMetaRaw):
		recordUnknownTopLevelFields(report, "session_meta_field", payload, isKnownSessionMetaField)
		if git, ok := extractTopLevelRawJSONFieldByMarker(payload, gitFieldMarker); ok {
			recordUnknownTopLevelFields(report, "git_field", git, isKnownGitField)
		}
	case bytes.Equal(recordTypeRaw, recordTypeTurnContextRaw):
		recordUnknownTopLevelFields(report, "turn_context_field", payload, isKnownTurnContextField)
	case bytes.Equal(recordTypeRaw, recordTypeResponseItemRaw):
		detectResponseItemPayloadDrift(payload, report)
	case bytes.Equal(recordTypeRaw, recordTypeEventMsgRaw):
		detectEventPayloadDrift(payload, report)
	}
}

func detectResponseItemPayloadDrift(payload []byte, report *src.DriftReport) {
	itemTypeRaw, _ := extractTopLevelRawJSONStringByMarker(payload, typeFieldMarker)
	detectResponseItemTypeDrift(itemTypeRaw, report)

	switch {
	case bytes.Equal(itemTypeRaw, responseTypeMessageRaw):
		detectResponseMessagePayloadDrift(payload, report)
	case bytes.Equal(itemTypeRaw, responseTypeReasoningRaw):
		detectReasoningPayloadDrift(payload, report)
	case bytes.Equal(itemTypeRaw, responseTypeFunctionCallRaw):
		recordUnknownTopLevelFields(report, "function_call_field", payload, isKnownFunctionCallField)
	case bytes.Equal(itemTypeRaw, responseTypeCustomToolCallRaw):
		recordUnknownTopLevelFields(report, "custom_tool_call_field", payload, isKnownCustomToolCallField)
	case bytes.Equal(itemTypeRaw, responseTypeFunctionCallOutputRaw),
		bytes.Equal(itemTypeRaw, responseTypeCustomToolCallOutputRaw):
		recordUnknownTopLevelFields(report, "tool_call_output_field", payload, isKnownToolCallOutputField)
	case bytes.Equal(itemTypeRaw, responseTypeWebSearchCallRaw):
		recordUnknownTopLevelFields(report, "web_search_call_field", payload, isKnownWebSearchCallField)
	}
}

func detectResponseMessagePayloadDrift(payload []byte, report *src.DriftReport) {
	recordUnknownTopLevelFields(report, "response_message_field", payload, isKnownResponseMessageField)
	if roleRaw, ok := extractTopLevelRawJSONStringByMarker(payload, roleFieldMarker); ok {
		detectRoleDrift(roleRaw, report)
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

func detectRoleDrift(roleRaw []byte, report *src.DriftReport) {
	recordUnknownValue(report, "role", roleRaw, isKnownRoleRaw)
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
		blockTypeRaw, ok := extractTopLevelRawJSONStringByMarker(value, typeFieldMarker)
		if !ok || isKnownContentBlockTypeRaw(blockTypeRaw) {
			return
		}
		recordUnknownValue(report, "content_block_type", blockTypeRaw, isKnownContentBlockTypeRaw)
	})
	if err != nil {
		return
	}
}

type driftEnvelope struct {
	timestampRaw  []byte
	recordTypeRaw []byte
	payload       []byte
	hasPayload    bool
}

func detectLineDrift(line []byte, report *src.DriftReport) driftEnvelope {
	envelope := scanDriftEnvelope(line, report)
	detectRecordTypeDrift(envelope.recordTypeRaw, report)
	if !envelope.hasPayload {
		return envelope
	}
	detectPayloadFieldDrift(envelope.recordTypeRaw, envelope.payload, report)
	return envelope
}

func scanDriftEnvelope(line []byte, report *src.DriftReport) driftEnvelope {
	pos, ok := topLevelObjectStart(line)
	if !ok {
		return driftEnvelope{}
	}

	var envelope driftEnvelope
	for {
		field, value, next, done, ok := nextTopLevelFieldValue(line, pos)
		if !ok || done {
			return envelope
		}

		switch {
		case bytes.Equal(field, timestampFieldMarker):
			envelope.timestampRaw = value
		case bytes.Equal(field, typeFieldMarker):
			envelope.recordTypeRaw = value
		case bytes.Equal(field, payloadFieldMarker):
			envelope.payload = value
			envelope.hasPayload = true
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

func recordUnknownValue(
	report *src.DriftReport,
	category string,
	rawValue []byte,
	known func([]byte) bool,
) {
	if report == nil || len(rawValue) == 0 || known(rawValue) {
		return
	}
	value, ok := decodeRawJSONString(rawValue)
	if !ok || value == "" {
		return
	}
	report.Record(category, value)
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
		bytes.Equal(field, gitFieldMarker) ||
		codexKnownSchemaExtras.HasRaw("session_meta_field", field)
}

func isKnownGitField(field []byte) bool {
	return bytes.Equal(field, branchFieldMarker) ||
		bytes.Equal(field, commitHashFieldMarker) ||
		codexKnownSchemaExtras.HasRaw("git_field", field)
}

func isKnownTurnContextField(field []byte) bool {
	return bytes.Equal(field, cwdFieldMarker) ||
		bytes.Equal(field, modelFieldMarker) ||
		bytes.Equal(field, approvalPolicyFieldMarker) ||
		bytes.Equal(field, sandboxPolicyFieldMarker) ||
		codexKnownSchemaExtras.HasRaw("turn_context_field", field)
}

func isKnownResponseMessageField(field []byte) bool {
	return bytes.Equal(field, typeFieldMarker) ||
		bytes.Equal(field, roleFieldMarker) ||
		bytes.Equal(field, contentFieldMarker) ||
		codexKnownSchemaExtras.HasRaw("response_message_field", field)
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
