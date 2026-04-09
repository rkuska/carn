package codex

import (
	"strings"
	"time"
)

func (s *loadState) applyRecord(recordType string, payload []byte, timestamp string) {
	switch recordType {
	case recordTypeSessionMeta:
		s.applySessionMeta(payload)
	case recordTypeTurnContext:
		s.applyTurnContext(payload)
	case recordTypeResponseItem:
		s.applyResponseItem(payload, parseTimestamp(timestamp))
	case recordTypeEventMsg:
		s.applyEvent(payload, timestamp)
	}
}

func (s *loadState) applyTurnContext(payload []byte) {
	if effort, ok := extractTopLevelRawJSONStringFieldByMarker(payload, effortFieldMarker); ok {
		s.currentEffort = effort
	}
}

func (s *loadState) applySessionMeta(payload []byte) {
	if id, ok := extractTopLevelRawJSONStringFieldByMarker(payload, idFieldMarker); ok && id != "" && id != s.meta.ID {
		return
	}
	if source, ok := extractTopLevelRawJSONFieldByMarker(payload, sourceFieldMarker); ok {
		if link, ok := parseSubagentLink(source); ok {
			s.link = link
		}
	}
}

func (s *loadState) applyResponseItem(payload []byte, timestamp time.Time) {
	itemType, ok := extractTopLevelRawJSONStringFieldByMarker(payload, typeFieldMarker)
	if !ok {
		return
	}

	switch itemType {
	case responseTypeMessage:
		s.applyMessage(payload, timestamp)
	case responseTypeReasoning:
		s.applyReasoning(payload, timestamp)
	case responseTypeFunctionCall, responseTypeCustomToolCall, responseTypeWebSearchCall:
		s.applyToolCall(itemType, payload, timestamp)
	case responseTypeFunctionCallOutput, responseTypeCustomToolCallOutput:
		s.applyToolResult(payload, timestamp)
	}
}

func (s *loadState) applyMessage(payload []byte, timestamp time.Time) {
	role, _ := extractTopLevelRawJSONStringFieldByMarker(payload, roleFieldMarker)
	content, ok := extractTopLevelRawJSONFieldByMarker(payload, contentFieldMarker)
	if !ok {
		return
	}
	message, ok := classifyResponseMessage(role, content)
	if !ok {
		return
	}
	message.phase, _ = extractTopLevelRawJSONStringFieldByMarker(payload, phaseFieldMarker)
	s.applyClassifiedMessage(message, timestamp)
}

func (s *loadState) applyReasoning(payload []byte, timestamp time.Time) {
	s.markPendingTimestamp(timestamp)
	s.pendingPerformance.ReasoningBlockCount++
	if summaryRaw, ok := extractTopLevelRawJSONFieldByMarker(payload, summaryFieldMarker); ok {
		if summary := extractReasoningText(summaryRaw); summary != "" {
			s.appendThinking(summary)
			return
		}
	}
	if encryptedContent, ok := extractTopLevelRawJSONStringFieldByMarker(
		payload,
		encryptedContentFieldMarker,
	); ok && strings.TrimSpace(encryptedContent) != "" {
		s.pendingHiddenThinking = true
		s.pendingPerformance.ReasoningRedactionCount++
	}
}

func (s *loadState) applyToolCall(itemType string, payload []byte, timestamp time.Time) {
	call := buildToolCall(itemType, payload, s.readEvidence)
	if call.Name == "" {
		return
	}

	callID, _ := extractTopLevelRawJSONStringFieldByMarker(payload, callIDFieldMarker)
	input := extractToolInputString(payload)

	s.markPendingTimestamp(timestamp)
	s.pendingCalls = append(s.pendingCalls, call)
	s.callMeta[callID] = toolEventMeta{
		call:  call,
		input: input,
	}
	rememberReadEvidence(call.Action, s.readEvidence)
}

func (s *loadState) applyToolResult(payload []byte, timestamp time.Time) {
	callID, _ := extractTopLevelRawJSONStringFieldByMarker(payload, callIDFieldMarker)
	meta := s.callMeta[callID]
	if meta.call.Name == "" {
		meta.call.Name = callID
	}
	s.markPendingTimestamp(timestamp)
	s.pendingResults = append(s.pendingResults, buildToolResult(payload, meta))
}

func (s *loadState) applyEvent(payload []byte, timestamp string) {
	ts := parseTimestamp(timestamp)
	itemType, ok := extractTopLevelRawJSONStringFieldByMarker(payload, typeFieldMarker)
	s.recordTaskCompleteEvent(itemType, ok)
	if s.applyEventMessage(payload, ts) {
		return
	}
	if !ok {
		return
	}
	if s.applyTokenCountEvent(itemType, payload) {
		return
	}
	if s.applyReasoningEvent(itemType, payload, ts) {
		return
	}
	if s.applyTaskStartedEvent(itemType, payload) {
		return
	}
	if s.applyCompletedItemEvent(itemType, payload, ts) {
		return
	}
	s.applySimpleCounterEvent(itemType)
}

func (s *loadState) recordTaskCompleteEvent(itemType string, ok bool) {
	if ok && itemType == eventTypeTaskComplete {
		s.meta.Performance.TaskCompleteCount++
	}
}

func (s *loadState) applyEventMessage(payload []byte, timestamp time.Time) bool {
	itemType, ok := extractTopLevelRawJSONStringFieldByMarker(payload, typeFieldMarker)
	if !ok {
		return false
	}

	classified, handled := classifyLoadedEventMessage(payload, itemType)
	if !handled {
		return false
	}
	if classified.role != "" || classified.isAgentDivider {
		s.applyClassifiedMessage(classified, timestamp)
	}
	return true
}

func (s *loadState) applyTokenCountEvent(itemType string, payload []byte) bool {
	if itemType != eventTypeTokenCount {
		return false
	}
	s.applyTokenCount(payload)
	return true
}

func (s *loadState) applyReasoningEvent(itemType string, payload []byte, timestamp time.Time) bool {
	if itemType != eventTypeAgentReasoning {
		return false
	}
	s.markPendingTimestamp(timestamp)
	s.pendingPerformance.ReasoningBlockCount++
	if text, ok := extractTopLevelRawJSONStringFieldByMarker(payload, textFieldMarker); ok {
		s.appendThinking(text)
	}
	return true
}

func (s *loadState) applyTaskStartedEvent(itemType string, payload []byte) bool {
	if itemType != eventTypeTaskStarted {
		return false
	}
	s.meta.Performance.TaskStartedCount++
	raw, ok := extractTopLevelRawJSONFieldByMarker(payload, modelContextWindowFieldMarker)
	if !ok {
		return true
	}
	window, windowOK := readRawJSONInt(raw, 0)
	if windowOK {
		s.meta.Performance.ModelContextWindow = max(s.meta.Performance.ModelContextWindow, window)
	}
	return true
}

func (s *loadState) applyCompletedItemEvent(itemType string, payload []byte, timestamp time.Time) bool {
	if itemType != eventTypeItemCompleted {
		return false
	}
	item, ok := extractTopLevelRawJSONFieldByMarker(payload, itemFieldMarker)
	if !ok {
		return true
	}
	if plan, ok := extractCompletedPlan(item, timestamp); ok {
		s.applyPlan(plan, timestamp)
	}
	return true
}

func (s *loadState) applySimpleCounterEvent(itemType string) {
	switch itemType {
	case eventTypeTurnAborted:
		s.meta.Performance.AbortCount++
	case eventTypeContextCompacted:
		s.meta.Performance.ContextCompactedCount++
	}
}
