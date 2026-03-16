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

	file, br, err := openReader(path)
	if err != nil {
		return rolloutTranscript{}, err
	}
	defer func() { _ = file.Close() }()
	defer readerPool.Put(br)

	var pc parseContext
	state := newLoadState(meta)
	dec := json.NewDecoder(br)
	for dec.More() {
		if err := ctx.Err(); err != nil {
			return rolloutTranscript{}, fmt.Errorf("loadConversation_ctx: %w", err)
		}

		pc.reset()
		if err := dec.Decode(&pc.rec); err != nil {
			return rolloutTranscript{}, fmt.Errorf("json.Decode: %w", err)
		}
		state.applyRecord(&pc.rec)
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

func (s *loadState) applyRecord(rec *codexRecord) {
	p := &rec.Payload
	switch rec.Type {
	case recordTypeSessionMeta:
		s.applySessionMeta(p)
	case recordTypeResponseItem:
		s.applyResponseItem(p, parseTimestamp(rec.Timestamp))
	case recordTypeEventMsg:
		s.applyEvent(p, rec.Timestamp)
	}
}

func (s *loadState) applySessionMeta(p *codexPayload) {
	if p.ID != "" && p.ID != s.meta.ID {
		return
	}
	if link, ok := parseSubagentLink(p.Source); ok {
		s.link = link
	}
}

func (s *loadState) applyResponseItem(p *codexPayload, timestamp time.Time) {
	switch p.ItemType {
	case responseTypeMessage:
		s.applyMessage(p, timestamp)
	case responseTypeReasoning:
		s.applyReasoning(p, timestamp)
	case responseTypeFunctionCall, responseTypeCustomToolCall, responseTypeWebSearchCall:
		s.applyToolCall(p, timestamp)
	case responseTypeFunctionCallOutput, responseTypeCustomToolCallOutput:
		s.applyToolResult(p, timestamp)
	}
}

func (s *loadState) applyMessage(p *codexPayload, timestamp time.Time) {
	message, ok := classifyResponseMessage(p.Role, p.Content)
	if !ok {
		return
	}
	s.applyClassifiedMessage(message, timestamp)
}

func (s *loadState) applyReasoning(p *codexPayload, timestamp time.Time) {
	s.markPendingTimestamp(timestamp)
	if summary := extractReasoningText(p.Summary); summary != "" {
		s.appendThinking(summary)
		return
	}
	if strings.TrimSpace(p.EncryptedContent) != "" {
		s.pendingHiddenThinking = true
	}
}

func (s *loadState) applyToolCall(p *codexPayload, timestamp time.Time) {
	call := buildToolCall(p)
	if call.Name == "" {
		return
	}

	s.markPendingTimestamp(timestamp)
	s.pendingCalls = append(s.pendingCalls, call)
	s.callMeta[p.CallID] = toolEventMeta{
		call:  call,
		input: p.Input,
	}
}

func (s *loadState) applyToolResult(p *codexPayload, timestamp time.Time) {
	meta := s.callMeta[p.CallID]
	if meta.call.Name == "" {
		meta.call.Name = p.CallID
	}
	s.markPendingTimestamp(timestamp)
	s.pendingResults = append(s.pendingResults, buildToolResult(p, meta))
}

func (s *loadState) applyEvent(p *codexPayload, timestamp string) {
	ts := parseTimestamp(timestamp)
	if s.applyEventMessage(p, ts) {
		return
	}

	switch p.ItemType {
	case eventTypeAgentReasoning:
		s.markPendingTimestamp(ts)
		s.appendThinking(p.Text)
	case eventTypeItemCompleted:
		if plan, ok := extractCompletedPlan(p.Item, ts); ok {
			s.applyPlan(plan, ts)
		}
	}
}

func (s *loadState) applyEventMessage(p *codexPayload, timestamp time.Time) bool {
	switch p.ItemType {
	case eventTypeUserMessage:
		if message, ok := classifyEventUserMessage(p.Message); ok {
			s.applyClassifiedMessage(message, timestamp)
		}
		return true
	case eventTypeAgentMessage:
		if message, ok := classifyEventAssistantMessage(p.Message); ok {
			s.applyClassifiedMessage(message, timestamp)
		}
		return true
	case eventTypeTaskComplete:
		if message, ok := classifyTaskCompleteMessage(p.LastAgentMessage); ok {
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
