package codex

import (
	"bytes"

	conv "github.com/rkuska/carn/internal/conversation"
)

type scannedEventPayload struct {
	eventTypeRaw          []byte
	messageRaw            []byte
	lastAgentMessageRaw   []byte
	infoRaw               []byte
	phaseRaw              []byte
	modelContextWindowRaw []byte
	rateLimitsRaw         []byte
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
		case bytes.Equal(field, phaseFieldMarker):
			scanned.phaseRaw = value
		case bytes.Equal(field, modelContextWindowFieldMarker):
			scanned.modelContextWindowRaw = value
		case bytes.Equal(field, rateLimitsFieldMarker):
			scanned.rateLimitsRaw = value
		}
		return true
	})
	return scanned
}

func applyEventPayload(scanned scannedEventPayload, state *scanState) {
	if applyTokenCountEventPayload(scanned, state) {
		return
	}
	if applyMessageEventPayload(scanned, state) {
		return
	}
	applyPerformanceEventPayload(scanned, state)
}

func recordEventAssistantMessage(scanned scannedEventPayload, state *scanState) {
	recordEventMessage(scanned.messageRaw, classifyEventAssistantMessage, state)
	if phase, ok := readRawJSONString(scanned.phaseRaw); ok && phase != "" {
		if state.meta.Performance.PhaseCounts == nil {
			state.meta.Performance.PhaseCounts = make(map[string]int, 1)
		}
		state.meta.Performance.PhaseCounts[phase]++
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

func scanTokenUsageInfo(raw []byte) conv.TokenUsage {
	usage, _ := scanTokenUsageInfoField(raw, totalTokenUsageFieldMarker)
	return usage
}

func scanLastTokenUsageInfo(raw []byte) (conv.TokenUsage, bool) {
	return scanTokenUsageInfoField(raw, lastTokenUsageFieldMarker)
}

func scanTokenUsageInfoField(raw []byte, fieldMarker []byte) (conv.TokenUsage, bool) {
	var usageRaw []byte
	walkTopLevelFields(raw, func(field, value []byte) bool {
		if bytes.Equal(field, fieldMarker) {
			usageRaw = value
			return false
		}
		return true
	})
	if len(usageRaw) == 0 {
		return conv.TokenUsage{}, false
	}
	return scanTokenUsageRaw(usageRaw), true
}

func scanTokenUsageRaw(raw []byte) conv.TokenUsage {
	var usage conv.TokenUsage
	walkTopLevelFields(raw, func(field, value []byte) bool {
		switch {
		case bytes.Equal(field, inputTokensFieldMarker):
			usage.InputTokens, _ = readRawJSONInt(value, 0)
		case bytes.Equal(field, cachedInputTokensFieldMarker):
			usage.CacheReadInputTokens, _ = readRawJSONInt(value, 0)
		case bytes.Equal(field, outputTokensFieldMarker):
			usage.OutputTokens, _ = readRawJSONInt(value, 0)
		case bytes.Equal(field, reasoningTokensFieldMarker):
			usage.ReasoningOutputTokens, _ = readRawJSONInt(value, 0)
		}
		return true
	})
	return usage
}

func scanModelContextWindow(raw []byte) (int, bool) {
	windowRaw, ok := extractTopLevelRawJSONFieldByMarker(raw, modelContextWindowFieldMarker)
	if !ok {
		return 0, false
	}
	return readRawJSONInt(windowRaw, 0)
}

func applyTokenCountEventPayload(scanned scannedEventPayload, state *scanState) bool {
	if !bytes.Equal(scanned.eventTypeRaw, eventTypeTokenCountRaw) {
		return false
	}
	state.meta.TotalUsage = scanTokenUsageInfo(scanned.infoRaw)
	if len(scanned.rateLimitsRaw) > 0 {
		state.meta.Performance.RateLimitSnapshotCount++
	}
	if window, ok := scanModelContextWindow(scanned.infoRaw); ok {
		state.meta.Performance.ModelContextWindow = max(state.meta.Performance.ModelContextWindow, window)
	}
	return true
}

func applyMessageEventPayload(scanned scannedEventPayload, state *scanState) bool {
	switch {
	case bytes.Equal(scanned.eventTypeRaw, eventTypeUserMessageRaw):
		recordEventMessage(scanned.messageRaw, classifyEventUserMessage, state)
		return true
	case bytes.Equal(scanned.eventTypeRaw, eventTypeAgentMessageRaw):
		recordEventAssistantMessage(scanned, state)
		return true
	case bytes.Equal(scanned.eventTypeRaw, eventTypeTaskCompleteRaw):
		state.meta.Performance.TaskCompleteCount++
		recordEventMessage(scanned.lastAgentMessageRaw, classifyTaskCompleteMessage, state)
		return true
	default:
		return false
	}
}

func applyPerformanceEventPayload(scanned scannedEventPayload, state *scanState) {
	switch {
	case bytes.Equal(scanned.eventTypeRaw, eventTypeAgentReasoningRaw):
		state.meta.Performance.ReasoningBlockCount++
	case bytes.Equal(scanned.eventTypeRaw, eventTypeTaskStartedRaw):
		state.meta.Performance.TaskStartedCount++
		if window, ok := readRawJSONInt(scanned.modelContextWindowRaw, 0); ok {
			state.meta.Performance.ModelContextWindow = max(state.meta.Performance.ModelContextWindow, window)
		}
	case bytes.Equal(scanned.eventTypeRaw, eventTypeTurnAbortedRaw):
		state.meta.Performance.AbortCount++
	case bytes.Equal(scanned.eventTypeRaw, eventTypeContextCompactedRaw):
		state.meta.Performance.ContextCompactedCount++
	}
}
