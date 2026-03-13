package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

type loadState struct {
	meta           conv.SessionMeta
	callMeta       map[string]toolEventMeta
	messages       []conv.Message
	thinkingParts  []string
	pendingCalls   []conv.ToolCall
	pendingResults []conv.ToolResult
	pendingPlans   []conv.Plan
}

func loadConversation(ctx context.Context, conversation conv.Conversation) (conv.Session, error) {
	path := firstSessionPath(conversation)
	if path == "" {
		return conv.Session{}, fmt.Errorf("loadConversation: %w", errMissingPath)
	}

	file, scanner, err := openScanner(path)
	if err != nil {
		return conv.Session{}, err
	}
	defer func() { _ = file.Close() }()

	state := newLoadState(conversation.Sessions[0])
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return conv.Session{}, fmt.Errorf("loadConversation_ctx: %w", err)
		}

		envelope, err := parseEnvelope(scanner.Bytes())
		if err != nil {
			return conv.Session{}, err
		}
		if err := state.applyEnvelope(envelope); err != nil {
			return conv.Session{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return conv.Session{}, fmt.Errorf("scanner.Err: %w", err)
	}

	return state.session(), nil
}

func newLoadState(meta conv.SessionMeta) loadState {
	return loadState{
		meta:           meta,
		callMeta:       make(map[string]toolEventMeta),
		messages:       make([]conv.Message, 0),
		thinkingParts:  make([]string, 0),
		pendingCalls:   make([]conv.ToolCall, 0),
		pendingResults: make([]conv.ToolResult, 0),
		pendingPlans:   make([]conv.Plan, 0),
	}
}

func (s *loadState) applyEnvelope(envelope recordEnvelope) error {
	switch envelope.Type {
	case recordTypeResponseItem:
		return s.applyResponseItem(envelope.Payload)
	case recordTypeEventMsg:
		return s.applyEvent(envelope.Payload, envelope.Timestamp)
	default:
		return nil
	}
}

func (s *loadState) applyResponseItem(raw json.RawMessage) error {
	var payload responseItemPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("json.Unmarshal_responseItem: %w", err)
	}

	switch payload.Type {
	case responseTypeMessage:
		s.applyMessage(payload)
	case responseTypeReasoning:
		s.applyReasoning(payload)
	case responseTypeFunctionCall, responseTypeCustomToolCall, responseTypeWebSearchCall:
		s.applyToolCall(payload)
	case responseTypeFunctionCallOutput, responseTypeCustomToolCallOutput:
		s.applyToolResult(payload)
	}
	return nil
}

func (s *loadState) applyMessage(payload responseItemPayload) {
	message, ok := classifyResponseMessage(payload.Role, payload.Content)
	if !ok {
		return
	}
	s.applyVisibleMessage(message)
}

func (s *loadState) applyReasoning(payload responseItemPayload) {
	s.appendThinking(extractReasoningText(payload.Summary))
}

func (s *loadState) applyToolCall(payload responseItemPayload) {
	call := buildToolCall(payload)
	if call.Name == "" {
		return
	}

	s.pendingCalls = append(s.pendingCalls, call)
	s.callMeta[payload.CallID] = toolEventMeta{
		call:  call,
		input: payload.Input,
	}
}

func (s *loadState) applyToolResult(payload responseItemPayload) {
	meta := s.callMeta[payload.CallID]
	if meta.call.Name == "" {
		meta.call.Name = payload.CallID
	}
	s.pendingResults = append(s.pendingResults, buildToolResult(payload, meta))
}

func (s *loadState) applyEvent(raw json.RawMessage, timestamp string) error {
	var payload eventPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("json.Unmarshal_event: %w", err)
	}

	if s.applyEventMessage(payload) {
		return nil
	}

	switch payload.Type {
	case eventTypeAgentReasoning:
		s.appendThinking(payload.Text)
	case eventTypeItemCompleted:
		if plan, ok := extractCompletedPlan(payload.Item, parseTimestamp(timestamp)); ok {
			s.applyPlan(plan)
		}
	}
	return nil
}

func (s *loadState) applyEventMessage(payload eventPayload) bool {
	switch payload.Type {
	case eventTypeUserMessage:
		if message, ok := classifyEventUserMessage(payload.Message); ok {
			s.applyVisibleMessage(message)
		}
		return true
	case eventTypeAgentMessage:
		if message, ok := classifyEventAssistantMessage(payload.Message); ok {
			s.applyVisibleMessage(message)
		}
		return true
	case eventTypeTaskComplete:
		if message, ok := classifyTaskCompleteMessage(payload.LastAgentMessage); ok {
			s.applyVisibleMessage(message)
		}
		return true
	default:
		return false
	}
}

func (s *loadState) applyVisibleMessage(message visibleMessage) {
	switch {
	case message.isAgentDivider:
		s.flushAssistant("")
		s.messages = appendDividerMessage(s.messages, message.text)
	case message.role == conv.RoleUser:
		s.flushAssistant("")
		s.messages = appendUserMessage(s.messages, message.text)
	case message.role == conv.RoleAssistant:
		s.flushAssistant(message.text)
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

func (s *loadState) applyPlan(plan conv.Plan) {
	if len(s.messages) > 0 &&
		s.messages[len(s.messages)-1].Role == conv.RoleAssistant &&
		len(s.thinkingParts) == 0 &&
		len(s.pendingCalls) == 0 &&
		len(s.pendingResults) == 0 &&
		len(s.pendingPlans) == 0 {
		s.messages[len(s.messages)-1].Plans = appendUniquePlans(
			s.messages[len(s.messages)-1].Plans,
			[]conv.Plan{plan},
		)
		return
	}
	s.pendingPlans = appendUniquePlans(s.pendingPlans, []conv.Plan{plan})
}

func (s *loadState) flushAssistant(text string) {
	s.messages = appendAssistantMessage(
		s.messages,
		joinText(s.thinkingParts),
		s.pendingCalls,
		s.pendingResults,
		s.pendingPlans,
		text,
	)
	s.thinkingParts = s.thinkingParts[:0]
	s.pendingCalls = s.pendingCalls[:0]
	s.pendingResults = s.pendingResults[:0]
	s.pendingPlans = s.pendingPlans[:0]
}

func (s *loadState) session() conv.Session {
	s.flushAssistant("")
	return conv.Session{
		Meta:     s.meta,
		Messages: s.messages,
	}
}

var errMissingPath = errors.New("missing conversation file path")

func appendUserMessage(messages []conv.Message, text string) []conv.Message {
	if text == "" {
		return messages
	}
	if len(messages) > 0 {
		last := messages[len(messages)-1]
		if last.Role == conv.RoleUser && !last.IsAgentDivider && last.Text == text {
			return messages
		}
	}
	return append(messages, conv.Message{Role: conv.RoleUser, Text: text})
}

func appendDividerMessage(messages []conv.Message, text string) []conv.Message {
	if text == "" {
		return messages
	}
	if len(messages) > 0 {
		last := messages[len(messages)-1]
		if last.IsAgentDivider && last.Text == text {
			return messages
		}
	}
	return append(messages, conv.Message{
		Role:           conv.RoleUser,
		Text:           text,
		IsAgentDivider: true,
	})
}
