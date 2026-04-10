package codex

import (
	"bytes"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

type scannedResponseItemPayload struct {
	itemTypeRaw  []byte
	roleRaw      []byte
	nameRaw      []byte
	callIDRaw    []byte
	argumentsRaw []byte
	inputRaw     []byte
	outputRaw    []byte
	statusRaw    []byte
	contentRaw   []byte
	encryptedRaw []byte
	actionRaw    []byte
	phaseRaw     []byte
}

func applyScannedResponseItemPayload(payload []byte, state *scanState) {
	item := collectResponseItemPayload(payload)
	applyResponseItemPayload(item, state)
}

func collectResponseItemPayload(payload []byte) scannedResponseItemPayload {
	var item scannedResponseItemPayload
	walkTopLevelFields(payload, func(field, value []byte) bool {
		if applyPrimaryResponseItemField(field, value, &item) {
			return true
		}
		applySecondaryResponseItemField(field, value, &item)
		return true
	})
	return item
}

func applyPrimaryResponseItemField(field, value []byte, item *scannedResponseItemPayload) bool {
	switch {
	case bytes.Equal(field, typeFieldMarker):
		item.itemTypeRaw = value
	case bytes.Equal(field, roleFieldMarker):
		item.roleRaw = value
	case bytes.Equal(field, nameFieldMarker):
		item.nameRaw = value
	case bytes.Equal(field, callIDFieldMarker):
		item.callIDRaw = value
	case bytes.Equal(field, argumentsFieldMarker):
		item.argumentsRaw = value
	case bytes.Equal(field, inputFieldMarker):
		item.inputRaw = value
	default:
		return false
	}
	return true
}

func applySecondaryResponseItemField(field, value []byte, item *scannedResponseItemPayload) {
	switch {
	case bytes.Equal(field, outputFieldMarker):
		item.outputRaw = value
	case bytes.Equal(field, statusFieldMarker):
		item.statusRaw = value
	case bytes.Equal(field, contentFieldMarker):
		item.contentRaw = value
	case bytes.Equal(field, encryptedContentFieldMarker):
		item.encryptedRaw = value
	case bytes.Equal(field, actionFieldMarker):
		item.actionRaw = value
	case bytes.Equal(field, phaseFieldMarker):
		item.phaseRaw = value
	}
}

func applyResponseItemPayload(item scannedResponseItemPayload, state *scanState) {
	switch {
	case bytes.Equal(item.itemTypeRaw, responseTypeMessageRaw):
		applyResponseMessageItem(item, state)
	case bytes.Equal(item.itemTypeRaw, responseTypeReasoningRaw):
		state.meta.Performance.ReasoningBlockCount++
		if hasScannedEncryptedReasoning(item.encryptedRaw) {
			state.meta.Performance.ReasoningRedactionCount++
		}
	case isResponseToolCallItemType(item.itemTypeRaw):
		recordScannedToolCall(item, state)
	case isResponseToolResultItemType(item.itemTypeRaw):
		applyResponseToolResultItem(item, state)
	}
}

func hasScannedEncryptedReasoning(raw []byte) bool {
	encrypted, ok := readRawJSONString(raw)
	return ok && strings.TrimSpace(encrypted) != ""
}

func applyResponseMessageItem(item scannedResponseItemPayload, state *scanState) {
	message, ok := classifyResponseMessageRaw(item.roleRaw, item.contentRaw)
	state.recordMessage(message, ok)
	if !ok {
		return
	}
	phase, ok := readRawJSONString(item.phaseRaw)
	if !ok || phase == "" {
		return
	}
	if state.meta.Performance.PhaseCounts == nil {
		state.meta.Performance.PhaseCounts = make(map[string]int, 1)
	}
	state.meta.Performance.PhaseCounts[phase]++
}

func isResponseToolCallItemType(raw []byte) bool {
	return bytes.Equal(raw, responseTypeFunctionCallRaw) ||
		bytes.Equal(raw, responseTypeCustomToolCallRaw) ||
		bytes.Equal(raw, responseTypeWebSearchCallRaw)
}

func isResponseToolResultItemType(raw []byte) bool {
	return bytes.Equal(raw, responseTypeFunctionCallOutputRaw) ||
		bytes.Equal(raw, responseTypeCustomToolCallOutputRaw)
}

func applyResponseToolResultItem(item scannedResponseItemPayload, state *scanState) {
	callID, _ := readRawJSONString(item.callIDRaw)
	state.recordToolResult(callID, item.outputRaw, item.statusRaw)
}

func recordScannedToolCall(item scannedResponseItemPayload, state *scanState) {
	callID, _ := readRawJSONString(item.callIDRaw)
	call, ok := buildScannedToolCall(item, state.readEvidence)
	if !ok {
		return
	}
	state.recordToolCall(callID, call)
}

func buildScannedToolCall(
	item scannedResponseItemPayload,
	readEvidence map[string]struct{},
) (conv.ToolCall, bool) {
	if bytes.Equal(item.itemTypeRaw, responseTypeWebSearchCallRaw) {
		return conv.ToolCall{
			Name:    toolNameWebSearch,
			Summary: buildWebSearchSummaryFromActionRaw(item.actionRaw),
			Action: conv.NormalizedAction{
				Type:    conv.NormalizedActionWeb,
				Targets: webSearchTargetsFromActionRaw(item.actionRaw),
			},
		}, true
	}

	name, ok := scanToolName(item.nameRaw)
	if !ok {
		return conv.ToolCall{}, false
	}

	input := scannedToolInput(item)
	return conv.ToolCall{
		Name:    name,
		Summary: buildToolSummaryFromInput(name, input),
		Action:  classifyCodexToolActionFromInput(name, input, readEvidence),
	}, true
}

func scannedToolInput(item scannedResponseItemPayload) string {
	if input, ok := readRawJSONString(item.inputRaw); ok && input != "" {
		return input
	}
	arguments, _ := readRawJSONString(item.argumentsRaw)
	return arguments
}
