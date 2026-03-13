package codex

import (
	"context"
	"encoding/json"
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
	thinkingParts         []string
	pendingHiddenThinking bool
	pendingCalls          []conv.ToolCall
	pendingResults        []conv.ToolResult
	pendingPlans          []conv.Plan
	pendingTimestamp      time.Time
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

	file, scanner, err := openScanner(path)
	if err != nil {
		return rolloutTranscript{}, err
	}
	defer func() { _ = file.Close() }()

	state := newLoadState(meta)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return rolloutTranscript{}, fmt.Errorf("loadConversation_ctx: %w", err)
		}

		envelope, err := parseEnvelope(scanner.Bytes())
		if err != nil {
			return rolloutTranscript{}, err
		}
		if err := state.applyEnvelope(envelope); err != nil {
			return rolloutTranscript{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return rolloutTranscript{}, fmt.Errorf("scanner.Err: %w", err)
	}

	return state.transcript(), nil
}

func newLoadState(meta conv.SessionMeta) loadState {
	return loadState{
		meta:           meta,
		callMeta:       make(map[string]toolEventMeta),
		messages:       make([]parsedMessage, 0),
		thinkingParts:  make([]string, 0),
		pendingCalls:   make([]conv.ToolCall, 0),
		pendingResults: make([]conv.ToolResult, 0),
		pendingPlans:   make([]conv.Plan, 0),
	}
}

func (s *loadState) applyEnvelope(envelope recordEnvelope) error {
	switch envelope.Type {
	case recordTypeSessionMeta:
		return s.applySessionMeta(envelope.Payload)
	case recordTypeResponseItem:
		return s.applyResponseItem(envelope.Payload, parseTimestamp(envelope.Timestamp))
	case recordTypeEventMsg:
		return s.applyEvent(envelope.Payload, envelope.Timestamp)
	default:
		return nil
	}
}

func (s *loadState) applySessionMeta(raw json.RawMessage) error {
	var payload sessionMetaPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("json.Unmarshal_sessionMeta: %w", err)
	}
	if payload.ID != "" && payload.ID != s.meta.ID {
		return nil
	}
	if link, ok := parseSubagentLink(payload.Source); ok {
		s.link = link
	}
	return nil
}

func (s *loadState) applyResponseItem(raw json.RawMessage, timestamp time.Time) error {
	var payload responseItemPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("json.Unmarshal_responseItem: %w", err)
	}

	switch payload.Type {
	case responseTypeMessage:
		s.applyMessage(payload, timestamp)
	case responseTypeReasoning:
		s.applyReasoning(payload, timestamp)
	case responseTypeFunctionCall, responseTypeCustomToolCall, responseTypeWebSearchCall:
		s.applyToolCall(payload, timestamp)
	case responseTypeFunctionCallOutput, responseTypeCustomToolCallOutput:
		s.applyToolResult(payload, timestamp)
	}
	return nil
}

func (s *loadState) applyMessage(payload responseItemPayload, timestamp time.Time) {
	message, ok := classifyResponseMessage(payload.Role, payload.Content)
	if !ok {
		return
	}
	s.applyClassifiedMessage(message, timestamp)
}

func (s *loadState) applyReasoning(payload responseItemPayload, timestamp time.Time) {
	s.markPendingTimestamp(timestamp)
	if summary := extractReasoningText(payload.Summary); summary != "" {
		s.appendThinking(summary)
		return
	}
	if strings.TrimSpace(payload.EncryptedContent) != "" {
		s.pendingHiddenThinking = true
	}
}

func (s *loadState) applyToolCall(payload responseItemPayload, timestamp time.Time) {
	call := buildToolCall(payload)
	if call.Name == "" {
		return
	}

	s.markPendingTimestamp(timestamp)
	s.pendingCalls = append(s.pendingCalls, call)
	s.callMeta[payload.CallID] = toolEventMeta{
		call:  call,
		input: payload.Input,
	}
}

func (s *loadState) applyToolResult(payload responseItemPayload, timestamp time.Time) {
	meta := s.callMeta[payload.CallID]
	if meta.call.Name == "" {
		meta.call.Name = payload.CallID
	}
	s.markPendingTimestamp(timestamp)
	s.pendingResults = append(s.pendingResults, buildToolResult(payload, meta))
}

func (s *loadState) applyEvent(raw json.RawMessage, timestamp string) error {
	var payload eventPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("json.Unmarshal_event: %w", err)
	}

	ts := parseTimestamp(timestamp)
	if s.applyEventMessage(payload, ts) {
		return nil
	}

	switch payload.Type {
	case eventTypeAgentReasoning:
		s.markPendingTimestamp(ts)
		s.appendThinking(payload.Text)
	case eventTypeItemCompleted:
		if plan, ok := extractCompletedPlan(payload.Item, ts); ok {
			s.applyPlan(plan, ts)
		}
	}
	return nil
}

func (s *loadState) applyEventMessage(payload eventPayload, timestamp time.Time) bool {
	switch payload.Type {
	case eventTypeUserMessage:
		if message, ok := classifyEventUserMessage(payload.Message); ok {
			s.applyClassifiedMessage(message, timestamp)
		}
		return true
	case eventTypeAgentMessage:
		if message, ok := classifyEventAssistantMessage(payload.Message); ok {
			s.applyClassifiedMessage(message, timestamp)
		}
		return true
	case eventTypeTaskComplete:
		if message, ok := classifyTaskCompleteMessage(payload.LastAgentMessage); ok {
			s.applyClassifiedMessage(message, timestamp)
		}
		return true
	default:
		return false
	}
}

func (s *loadState) applyClassifiedMessage(message visibleMessage, timestamp time.Time) {
	switch {
	case message.isAgentDivider:
		s.flushAssistant("", time.Time{})
		s.messages = appendParsedDividerMessage(s.messages, message.text, timestamp)
	case message.role == conv.RoleSystem:
		s.flushAssistant("", time.Time{})
		s.messages = appendParsedSystemMessage(s.messages, message.text, message.visibility, timestamp)
	case message.role == conv.RoleUser:
		s.flushAssistant("", time.Time{})
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

func (s *loadState) flushAssistant(text string, timestamp time.Time) {
	thinking := joinText(s.thinkingParts)
	hasHiddenThinking := s.pendingHiddenThinking && strings.TrimSpace(thinking) == ""
	s.messages = appendParsedAssistantMessage(
		s.messages,
		thinking,
		hasHiddenThinking,
		s.pendingCalls,
		s.pendingResults,
		s.pendingPlans,
		text,
		maxTime(s.pendingTimestamp, timestamp),
	)
	s.thinkingParts = s.thinkingParts[:0]
	s.pendingHiddenThinking = false
	s.pendingCalls = s.pendingCalls[:0]
	s.pendingResults = s.pendingResults[:0]
	s.pendingPlans = s.pendingPlans[:0]
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

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
