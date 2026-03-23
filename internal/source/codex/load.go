package codex

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type loadState struct {
	meta                  conv.SessionMeta
	link                  subagentLink
	callMeta              map[string]toolEventMeta
	messages              []parsedMessage
	pendingUsage          conv.TokenUsage
	thinkingParts         []string
	pendingHiddenThinking bool
	pendingCalls          []conv.ToolCall
	pendingResults        []conv.ToolResult
	pendingPlans          []conv.Plan
	pendingTimestamp      time.Time
	usageTargetIndex      int
}

func loadConversation(ctx context.Context, conversation conv.Conversation) (conv.Session, error) {
	if len(conversation.Sessions) == 0 {
		return conv.Session{}, fmt.Errorf("loadConversation: %w", errMissingPath)
	}

	parent, err := loadRollout(ctx, conversation.Sessions[0])
	if err != nil {
		return conv.Session{}, fmt.Errorf("loadRollout_parent: %w", err)
	}

	linked, err := loadLinkedTranscripts(ctx, conversation.Sessions[1:])
	if err != nil {
		return conv.Session{}, fmt.Errorf("loadLinkedTranscripts: %w", err)
	}

	return conv.Session{
		Meta:     parent.meta,
		Messages: projectParsedMessages(mergeSubagentTranscripts(parent, linked)),
	}, nil
}

func loadRollout(ctx context.Context, meta conv.SessionMeta) (rolloutTranscript, error) {
	path := meta.FilePath
	if path == "" {
		return rolloutTranscript{}, fmt.Errorf("loadRollout: %w", errMissingPath)
	}

	state := newLoadState(meta)
	if err := visitRolloutRecords(ctx, path, func(recordType string, payload []byte, timestamp string) {
		state.applyRecord(recordType, payload, timestamp)
	}); err != nil {
		return rolloutTranscript{}, fmt.Errorf("visitRolloutRecords: %w", err)
	}
	return state.transcript(), nil
}

func newLoadState(meta conv.SessionMeta) loadState {
	return loadState{
		meta:             meta,
		callMeta:         make(map[string]toolEventMeta),
		messages:         make([]parsedMessage, 0),
		thinkingParts:    make([]string, 0),
		pendingCalls:     make([]conv.ToolCall, 0),
		pendingResults:   make([]conv.ToolResult, 0),
		pendingPlans:     make([]conv.Plan, 0),
		usageTargetIndex: -1,
	}
}

func (s *loadState) applyRecord(recordType string, payload []byte, timestamp string) {
	switch recordType {
	case recordTypeSessionMeta:
		s.applySessionMeta(payload)
	case recordTypeResponseItem:
		s.applyResponseItem(payload, parseTimestamp(timestamp))
	case recordTypeEventMsg:
		s.applyEvent(payload, timestamp)
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
	s.applyClassifiedMessage(message, timestamp)
}

func (s *loadState) applyReasoning(payload []byte, timestamp time.Time) {
	s.markPendingTimestamp(timestamp)
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
	}
}

func (s *loadState) applyToolCall(itemType string, payload []byte, timestamp time.Time) {
	call := buildToolCall(itemType, payload)
	if call.Name == "" {
		return
	}

	callID, _ := extractTopLevelRawJSONStringFieldByMarker(payload, callIDFieldMarker)
	input, _ := extractTopLevelRawJSONStringFieldByMarker(payload, inputFieldMarker)

	s.markPendingTimestamp(timestamp)
	s.pendingCalls = append(s.pendingCalls, call)
	s.callMeta[callID] = toolEventMeta{
		call:  call,
		input: input,
	}
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
	if s.applyEventMessage(payload, ts) {
		return
	}

	itemType, ok := extractTopLevelRawJSONStringFieldByMarker(payload, typeFieldMarker)
	if !ok {
		return
	}

	switch itemType {
	case eventTypeTokenCount:
		s.applyTokenCount(payload)
	case eventTypeAgentReasoning:
		s.markPendingTimestamp(ts)
		if text, ok := extractTopLevelRawJSONStringFieldByMarker(payload, textFieldMarker); ok {
			s.appendThinking(text)
		}
	case eventTypeItemCompleted:
		item, ok := extractTopLevelRawJSONFieldByMarker(payload, itemFieldMarker)
		if !ok {
			return
		}
		if plan, ok := extractCompletedPlan(item, ts); ok {
			s.applyPlan(plan, ts)
		}
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

func (s *loadState) applyClassifiedMessage(message visibleMessage, timestamp time.Time) {
	switch {
	case message.isAgentDivider:
		s.flushAssistant("", time.Time{})
		s.clearAssistantUsageTarget()
		s.messages = appendParsedDividerMessage(s.messages, message.text, timestamp)
	case message.role == conv.RoleSystem:
		s.flushAssistant("", time.Time{})
		s.clearAssistantUsageTarget()
		s.messages = appendParsedSystemMessage(s.messages, message.text, message.visibility, timestamp)
	case message.role == conv.RoleUser:
		s.flushAssistant("", time.Time{})
		s.clearAssistantUsageTarget()
		s.messages = appendParsedUserMessage(s.messages, message.text, timestamp)
	case message.role == conv.RoleAssistant:
		s.flushAssistant(message.text, timestamp)
	}
}

func (s *loadState) appendThinking(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if len(s.thinkingParts) > 0 && s.thinkingParts[len(s.thinkingParts)-1] == text {
		return
	}
	s.thinkingParts = append(s.thinkingParts, text)
}

func (s *loadState) applyPlan(plan conv.Plan, timestamp time.Time) {
	if len(s.messages) > 0 &&
		s.messages[len(s.messages)-1].role == conv.RoleAssistant &&
		len(s.thinkingParts) == 0 &&
		len(s.pendingCalls) == 0 &&
		len(s.pendingResults) == 0 &&
		len(s.pendingPlans) == 0 {
		s.messages[len(s.messages)-1].plans = appendUniquePlans(
			s.messages[len(s.messages)-1].plans,
			[]conv.Plan{plan},
		)
		return
	}
	s.markPendingTimestamp(timestamp)
	s.pendingPlans = appendUniquePlans(s.pendingPlans, []conv.Plan{plan})
}

func (s *loadState) applyTokenCount(payload []byte) {
	info, ok := extractTopLevelRawJSONFieldByMarker(payload, infoFieldMarker)
	if !ok {
		return
	}
	usage, ok := scanLastTokenUsageInfo(info)
	if !ok {
		return
	}
	s.attachAssistantUsage(usage)
}

func (s *loadState) attachAssistantUsage(usage conv.TokenUsage) {
	if s.hasPendingAssistantContent() {
		s.pendingUsage = usage
		return
	}
	if s.usageTargetIndex < 0 || s.usageTargetIndex >= len(s.messages) {
		return
	}
	target := &s.messages[s.usageTargetIndex]
	if target.role != conv.RoleAssistant || target.isAgentDivider {
		return
	}
	target.usage = usage
}

func (s *loadState) flushAssistant(text string, timestamp time.Time) {
	thinking := joinText(s.thinkingParts)
	hasHiddenThinking := s.pendingHiddenThinking && strings.TrimSpace(thinking) == ""
	var appended bool
	s.messages, s.usageTargetIndex, appended = appendParsedAssistantMessage(s.messages, assistantContent{
		thinking:          thinking,
		hasHiddenThinking: hasHiddenThinking,
		calls:             s.pendingCalls,
		results:           s.pendingResults,
		plans:             s.pendingPlans,
		usage:             s.pendingUsage,
		text:              text,
		timestamp:         maxTime(s.pendingTimestamp, timestamp),
	})
	if !appended && text == "" && !s.hasPendingAssistantContent() {
		s.clearAssistantUsageTarget()
	}
	s.thinkingParts = s.thinkingParts[:0]
	s.pendingHiddenThinking = false
	s.pendingCalls = s.pendingCalls[:0]
	s.pendingResults = s.pendingResults[:0]
	s.pendingPlans = s.pendingPlans[:0]
	s.pendingUsage = conv.TokenUsage{}
	s.pendingTimestamp = time.Time{}
}

func (s *loadState) transcript() rolloutTranscript {
	s.flushAssistant("", time.Time{})
	return rolloutTranscript{
		meta:     s.meta,
		link:     s.link,
		messages: s.messages,
	}
}

var errMissingPath = errors.New("missing conversation file path")

func (s *loadState) markPendingTimestamp(timestamp time.Time) {
	if timestamp.After(s.pendingTimestamp) {
		s.pendingTimestamp = timestamp
	}
}

func (s *loadState) hasPendingAssistantContent() bool {
	return len(s.thinkingParts) > 0 ||
		s.pendingHiddenThinking ||
		len(s.pendingCalls) > 0 ||
		len(s.pendingResults) > 0 ||
		len(s.pendingPlans) > 0
}

func (s *loadState) clearAssistantUsageTarget() {
	s.usageTargetIndex = -1
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
