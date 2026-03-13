package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type scanState struct {
	meta          conv.SessionMeta
	firstRecordTS time.Time
	lastRole      conv.Role
	lastText      string
}

func scanRollouts(ctx context.Context, rawDir string) ([]conv.Conversation, error) {
	conversations := make([]conv.Conversation, 0)

	err := filepath.WalkDir(rawDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !isJSONLExt(path) {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		conversation, ok, err := scanRollout(path)
		if err != nil {
			return fmt.Errorf("scanRollout_%s: %w", filepath.Base(path), err)
		}
		if ok {
			conversations = append(conversations, conversation)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("filepath.WalkDir: %w", err)
	}

	return conversations, nil
}

func scanRollout(path string) (conv.Conversation, bool, error) {
	file, scanner, err := openScanner(path)
	if err != nil {
		return conv.Conversation{}, false, err
	}
	defer func() { _ = file.Close() }()

	state := newScanState(path)
	for scanner.Scan() {
		envelope, err := parseEnvelope(scanner.Bytes())
		if err != nil {
			return conv.Conversation{}, false, err
		}
		if err := state.applyEnvelope(envelope); err != nil {
			return conv.Conversation{}, false, err
		}
	}
	if err := scanner.Err(); err != nil {
		return conv.Conversation{}, false, fmt.Errorf("scanner.Err: %w", err)
	}

	return state.conversation()
}

func newScanState(path string) scanState {
	return scanState{
		meta: conv.SessionMeta{
			FilePath:   path,
			ToolCounts: make(map[string]int),
		},
	}
}

func (s *scanState) applyEnvelope(envelope recordEnvelope) error {
	s.observeRecordTimestamp(envelope.Timestamp)

	switch envelope.Type {
	case recordTypeSessionMeta:
		return s.applySessionMeta(envelope.Payload)
	case recordTypeTurnContext:
		return s.applyTurnContext(envelope.Payload)
	case recordTypeResponseItem:
		return s.applyResponseItem(envelope.Payload)
	case recordTypeEventMsg:
		return s.applyEvent(envelope.Payload)
	default:
		return nil
	}
}

func (s *scanState) observeRecordTimestamp(value string) {
	ts := parseTimestamp(value)
	if ts.IsZero() {
		return
	}

	if s.meta.Timestamp.IsZero() {
		s.firstRecordTS = ts
	}
	if ts.After(s.meta.LastTimestamp) {
		s.meta.LastTimestamp = ts
	}
}

func (s *scanState) applySessionMeta(raw json.RawMessage) error {
	var payload sessionMetaPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("json.Unmarshal_sessionMeta: %w", err)
	}
	if s.meta.ID != "" && payload.ID != s.meta.ID {
		return nil
	}

	s.meta.ID = payload.ID
	s.meta.Slug = slugFromThreadID(s.meta.ID)
	if ts := parseTimestamp(payload.Timestamp); !ts.IsZero() {
		s.meta.Timestamp = ts
	}
	if s.meta.CWD == "" {
		s.meta.CWD = payload.CWD
	}
	if s.meta.Version == "" {
		s.meta.Version = payload.CLIVersion
	}
	if s.meta.Model == "" {
		s.meta.Model = payload.ModelProvider
	}
	if s.meta.GitBranch == "" {
		s.meta.GitBranch = payload.Git.Branch
	}
	s.meta.IsSubagent = s.meta.IsSubagent || sourceIsSubagent(payload.Source)
	return nil
}

func (s *scanState) applyTurnContext(raw json.RawMessage) error {
	var payload turnContextPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("json.Unmarshal_turnContext: %w", err)
	}
	if payload.CWD != "" {
		s.meta.CWD = payload.CWD
	}
	if payload.Model != "" {
		s.meta.Model = payload.Model
	}
	return nil
}

func (s *scanState) applyResponseItem(raw json.RawMessage) error {
	var payload responseItemPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("json.Unmarshal_responseItem: %w", err)
	}

	switch payload.Type {
	case responseTypeMessage:
		s.recordMessage(classifyResponseMessage(payload.Role, payload.Content))
	case responseTypeFunctionCall, responseTypeCustomToolCall, responseTypeWebSearchCall:
		s.recordToolCall(payload)
	}
	return nil
}

func (s *scanState) recordMessage(message visibleMessage, ok bool) {
	if !ok || message.text == "" || message.isAgentDivider {
		return
	}

	if s.lastRole == message.role && s.lastText == message.text {
		return
	}
	s.lastRole = message.role
	s.lastText = message.text
	s.meta.MessageCount++
	s.meta.MainMessageCount++
	if message.role == conv.RoleUser && s.meta.FirstMessage == "" {
		s.meta.FirstMessage = message.text
	}
}

func (s *scanState) recordToolCall(payload responseItemPayload) {
	call := buildToolCall(payload)
	if call.Name == "" {
		return
	}
	s.meta.ToolCounts[call.Name]++
}

func (s *scanState) applyEvent(raw json.RawMessage) error {
	var payload eventPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("json.Unmarshal_event: %w", err)
	}
	switch payload.Type {
	case eventTypeTokenCount:
		s.meta.TotalUsage = usageFromEvent(payload)
	case eventTypeUserMessage:
		s.recordMessage(classifyEventUserMessage(payload.Message))
	case eventTypeAgentMessage:
		s.recordMessage(classifyEventAssistantMessage(payload.Message))
	case eventTypeTaskComplete:
		s.recordMessage(classifyTaskCompleteMessage(payload.LastAgentMessage))
	}
	return nil
}

func (s *scanState) conversation() (conv.Conversation, bool, error) {
	if s.meta.ID == "" {
		return conv.Conversation{}, false, nil
	}

	meta := s.meta
	if meta.Timestamp.IsZero() {
		meta.Timestamp = s.firstRecordTS
	}
	if meta.LastTimestamp.IsZero() {
		meta.LastTimestamp = meta.Timestamp
	}
	meta.Project = conv.Project{DisplayName: projectNameFromCWD(meta.CWD)}
	if len(meta.ToolCounts) == 0 {
		meta.ToolCounts = nil
	}
	if meta.Slug == "" {
		meta.Slug = slugFromThreadID(meta.ID)
	}

	return conv.Conversation{
		Ref: conv.Ref{
			Provider: conv.ProviderCodex,
			ID:       meta.ID,
		},
		Project:  meta.Project,
		Sessions: []conv.SessionMeta{meta},
	}, true, nil
}
