package codex

import (
	"bytes"

	conv "github.com/rkuska/carn/internal/conversation"
)

func scanRolloutPayload(recordTypeRaw []byte, payload []byte, state *scanState) error {
	switch {
	case bytes.Equal(recordTypeRaw, recordTypeSessionMetaRaw):
		applyScannedSessionMetaPayload(payload, state)
	case bytes.Equal(recordTypeRaw, recordTypeTurnContextRaw):
		applyScannedTurnContextPayload(payload, state)
	case bytes.Equal(recordTypeRaw, recordTypeResponseItemRaw):
		applyScannedResponseItemPayload(payload, state)
	case bytes.Equal(recordTypeRaw, recordTypeEventMsgRaw):
		applyScannedEventPayload(payload, state)
	}
	return nil
}

type scannedSessionMetaPayload struct {
	idRaw        []byte
	timestampRaw []byte
	cwdRaw       []byte
	versionRaw   []byte
	modelRaw     []byte
	sourceRaw    []byte
	gitRaw       []byte
}

func applyScannedSessionMetaPayload(payload []byte, state *scanState) {
	applySessionMetaPayload(collectSessionMetaPayload(payload), state)
}

func collectSessionMetaPayload(payload []byte) scannedSessionMetaPayload {
	var scanned scannedSessionMetaPayload
	walkTopLevelFields(payload, func(field, value []byte) bool {
		switch {
		case bytes.Equal(field, idFieldMarker):
			scanned.idRaw = value
		case bytes.Equal(field, timestampFieldMarker):
			scanned.timestampRaw = value
		case bytes.Equal(field, cwdFieldMarker):
			scanned.cwdRaw = value
		case bytes.Equal(field, cliVersionFieldMarker):
			scanned.versionRaw = value
		case bytes.Equal(field, modelProviderFieldMarker):
			scanned.modelRaw = value
		case bytes.Equal(field, sourceFieldMarker):
			scanned.sourceRaw = value
		case bytes.Equal(field, gitFieldMarker):
			scanned.gitRaw = value
		}
		return true
	})
	return scanned
}

func applySessionMetaPayload(scanned scannedSessionMetaPayload, state *scanState) {
	id, ok := readRawJSONString(scanned.idRaw)
	if !shouldApplyScanSessionMeta(id, ok, state) {
		return
	}

	state.meta.ID = id
	state.meta.Slug = slugFromThreadID(state.meta.ID)
	applySessionTimestampRaw(scanned.timestampRaw, state)
	applySessionCWDRaw(scanned.cwdRaw, state)
	applySessionVersionRaw(scanned.versionRaw, state)
	applySessionModelRaw(scanned.modelRaw, state)
	applySessionGitBranchRaw(scanned.gitRaw, state)
	applySessionSourceRaw(scanned.sourceRaw, state)
}

func applySessionTimestampRaw(raw []byte, state *scanState) {
	rawTimestamp, ok := readRawJSONString(raw)
	if !ok {
		return
	}
	if ts := parseTimestamp(rawTimestamp); !ts.IsZero() {
		state.meta.Timestamp = ts
	}
}

func applySessionCWDRaw(raw []byte, state *scanState) {
	if state.meta.CWD != "" {
		return
	}
	if cwd, ok := readRawJSONString(raw); ok {
		state.meta.CWD = cwd
	}
}

func applySessionVersionRaw(raw []byte, state *scanState) {
	if state.meta.Version != "" {
		return
	}
	if version, ok := readRawJSONString(raw); ok {
		state.meta.Version = version
	}
}

func applySessionModelRaw(raw []byte, state *scanState) {
	if state.meta.Model != "" {
		return
	}
	if model, ok := readRawJSONString(raw); ok {
		state.meta.Model = model
	}
}

func applySessionGitBranchRaw(raw []byte, state *scanState) {
	if state.meta.GitBranch != "" {
		return
	}
	if branch, ok := scanGitBranchRaw(raw); ok {
		state.meta.GitBranch = branch
	}
}

func applySessionSourceRaw(raw []byte, state *scanState) {
	if link, ok := parseSubagentLink(raw); ok {
		state.link = link
		state.meta.IsSubagent = true
	}
}

func applyScannedTurnContextPayload(payload []byte, state *scanState) {
	var cwdRaw []byte
	var modelRaw []byte

	walkTopLevelFields(payload, func(field, value []byte) bool {
		switch {
		case bytes.Equal(field, cwdFieldMarker):
			cwdRaw = value
		case bytes.Equal(field, modelFieldMarker):
			modelRaw = value
		}
		return true
	})

	if cwd, ok := readRawJSONString(cwdRaw); ok && cwd != "" {
		state.meta.CWD = cwd
	}
	if model, ok := readRawJSONString(modelRaw); ok && model != "" {
		state.meta.Model = model
	}
}

func applyScannedResponseItemPayload(payload []byte, state *scanState) {
	item := collectResponseItemPayload(payload)
	applyResponseItemPayload(item, state)
}

type scannedResponseItemPayload struct {
	itemTypeRaw []byte
	roleRaw     []byte
	nameRaw     []byte
	callIDRaw   []byte
	outputRaw   []byte
	statusRaw   []byte
	contentRaw  []byte
}

func collectResponseItemPayload(payload []byte) scannedResponseItemPayload {
	var item scannedResponseItemPayload
	walkTopLevelFields(payload, func(field, value []byte) bool {
		switch {
		case bytes.Equal(field, typeFieldMarker):
			item.itemTypeRaw = value
		case bytes.Equal(field, roleFieldMarker):
			item.roleRaw = value
		case bytes.Equal(field, nameFieldMarker):
			item.nameRaw = value
		case bytes.Equal(field, callIDFieldMarker):
			item.callIDRaw = value
		case bytes.Equal(field, outputFieldMarker):
			item.outputRaw = value
		case bytes.Equal(field, statusFieldMarker):
			item.statusRaw = value
		case bytes.Equal(field, contentFieldMarker):
			item.contentRaw = value
		}
		return true
	})
	return item
}

func applyResponseItemPayload(item scannedResponseItemPayload, state *scanState) {
	switch {
	case bytes.Equal(item.itemTypeRaw, responseTypeMessageRaw):
		state.recordMessage(classifyResponseMessageRaw(item.roleRaw, item.contentRaw))
	case bytes.Equal(item.itemTypeRaw, responseTypeFunctionCallRaw),
		bytes.Equal(item.itemTypeRaw, responseTypeCustomToolCallRaw):
		recordScannedToolCall(item, state)
	case bytes.Equal(item.itemTypeRaw, responseTypeWebSearchCallRaw):
		callID, _ := readRawJSONString(item.callIDRaw)
		state.recordToolCall(callID, "web_search")
	case bytes.Equal(item.itemTypeRaw, responseTypeFunctionCallOutputRaw),
		bytes.Equal(item.itemTypeRaw, responseTypeCustomToolCallOutputRaw):
		callID, _ := readRawJSONString(item.callIDRaw)
		state.recordToolResult(callID, item.outputRaw, item.statusRaw)
	}
}

func recordScannedToolCall(item scannedResponseItemPayload, state *scanState) {
	name, ok := scanToolName(item.nameRaw)
	if !ok {
		return
	}
	callID, _ := readRawJSONString(item.callIDRaw)
	state.recordToolCall(callID, name)
}

type scannedEventPayload struct {
	eventTypeRaw        []byte
	messageRaw          []byte
	lastAgentMessageRaw []byte
	infoRaw             []byte
}

func applyScannedEventPayload(payload []byte, state *scanState) {
	applyEventPayload(collectEventPayload(payload), state)
}

func collectEventPayload(payload []byte) scannedEventPayload {
	var scanned scannedEventPayload
	walkTopLevelFields(payload, func(field, value []byte) bool {
		switch {
		case bytes.Equal(field, typeFieldMarker):
			scanned.eventTypeRaw = value
		case bytes.Equal(field, messageFieldMarker):
			scanned.messageRaw = value
		case bytes.Equal(field, lastAgentMessageFieldMarker):
			scanned.lastAgentMessageRaw = value
		case bytes.Equal(field, infoFieldMarker):
			scanned.infoRaw = value
		}
		return true
	})
	return scanned
}

func applyEventPayload(scanned scannedEventPayload, state *scanState) {
	switch {
	case bytes.Equal(scanned.eventTypeRaw, eventTypeTokenCountRaw):
		state.meta.TotalUsage = scanTokenUsageInfo(scanned.infoRaw)
	case bytes.Equal(scanned.eventTypeRaw, eventTypeUserMessageRaw):
		recordEventMessage(scanned.messageRaw, classifyEventUserMessage, state)
	case bytes.Equal(scanned.eventTypeRaw, eventTypeAgentMessageRaw):
		recordEventMessage(scanned.messageRaw, classifyEventAssistantMessage, state)
	case bytes.Equal(scanned.eventTypeRaw, eventTypeTaskCompleteRaw):
		recordEventMessage(scanned.lastAgentMessageRaw, classifyTaskCompleteMessage, state)
	}
}

func recordEventMessage(
	raw []byte,
	classify func(string) (visibleMessage, bool),
	state *scanState,
) {
	message, ok := readRawJSONString(raw)
	if !ok {
		return
	}
	state.recordMessage(classify(message))
}

func scanGitBranchRaw(raw []byte) (string, bool) {
	var branchRaw []byte
	walkTopLevelFields(raw, func(field, value []byte) bool {
		if bytes.Equal(field, branchFieldMarker) {
			branchRaw = value
			return false
		}
		return true
	})
	return readRawJSONString(branchRaw)
}

func scanTokenUsageInfo(raw []byte) conv.TokenUsage {
	var usageRaw []byte
	walkTopLevelFields(raw, func(field, value []byte) bool {
		if bytes.Equal(field, totalTokenUsageFieldMarker) {
			usageRaw = value
			return false
		}
		return true
	})

	var usage conv.TokenUsage
	walkTopLevelFields(usageRaw, func(field, value []byte) bool {
		switch {
		case bytes.Equal(field, inputTokensFieldMarker):
			usage.InputTokens, _ = readRawJSONInt(value, 0)
		case bytes.Equal(field, cachedInputTokensFieldMarker):
			usage.CacheReadInputTokens, _ = readRawJSONInt(value, 0)
		case bytes.Equal(field, outputTokensFieldMarker):
			usage.OutputTokens, _ = readRawJSONInt(value, 0)
		case bytes.Equal(field, reasoningTokensFieldMarker):
			reasoningTokens, _ := readRawJSONInt(value, 0)
			usage.OutputTokens += reasoningTokens
		}
		return true
	})
	return usage
}
