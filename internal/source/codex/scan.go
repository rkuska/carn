package codex

import (
	"context"
	"encoding/json"
	"fmt"

	conv "github.com/rkuska/carn/internal/conversation"
)

type scanState struct {
	meta       conv.SessionMeta
	firstRawTS string
	lastRawTS  string
	lastRole   conv.Role
	lastText   string
	link       subagentLink
}

func scanRollouts(ctx context.Context, rawDir string) ([]conv.Conversation, error) {
	paths, err := listJSONLPaths(rawDir)
	if err != nil {
		return nil, fmt.Errorf("listJSONLPaths: %w", err)
	}
	if len(paths) == 0 {
		return nil, nil
	}

	rollouts, err := scanRolloutsParallel(ctx, paths)
	if err != nil {
		return nil, fmt.Errorf("scanRolloutsParallel: %w", err)
	}
	return groupRollouts(rollouts), nil
}

func scanRollout(path string) (scannedRollout, bool, error) {
	file, br, err := openReader(path)
	if err != nil {
		return scannedRollout{}, false, err
	}
	defer func() { _ = file.Close() }()
	defer readerPool.Put(br)

	var pc scanContext
	state := newScanState(path)
	dec := json.NewDecoder(br)
	for dec.More() {
		pc.reset()
		if err := dec.Decode(&pc.rec); err != nil {
			return scannedRollout{}, false, fmt.Errorf("json.Decode: %w", err)
		}
		state.applyRecord(&pc.rec)
	}

	return state.rollout()
}

func newScanState(path string) scanState {
	return scanState{
		meta: conv.SessionMeta{
			FilePath: path,
		},
	}
}

func (s *scanState) applyRecord(rec *scanRecord) {
	s.observeRecordTimestamp(rec.Timestamp)

	p := &rec.Payload
	switch rec.Type {
	case recordTypeSessionMeta:
		s.applySessionMeta(p)
	case recordTypeTurnContext:
		s.applyTurnContext(p)
	case recordTypeResponseItem:
		s.applyResponseItem(p)
	case recordTypeEventMsg:
		s.applyEvent(p)
	}
}

func (s *scanState) observeRecordTimestamp(value string) {
	if value == "" {
		return
	}
	if s.firstRawTS == "" {
		s.firstRawTS = value
	}
	if value > s.lastRawTS {
		s.lastRawTS = value
	}
}

func (s *scanState) applySessionMeta(p *scanPayload) {
	if s.meta.ID != "" && p.ID != s.meta.ID {
		return
	}

	s.meta.ID = p.ID
	s.meta.Slug = slugFromThreadID(s.meta.ID)
	if ts := parseTimestamp(p.PayloadTS); !ts.IsZero() {
		s.meta.Timestamp = ts
	}
	if s.meta.CWD == "" {
		s.meta.CWD = p.CWD
	}
	if s.meta.Version == "" {
		s.meta.Version = p.CLIVersion
	}
	if s.meta.Model == "" {
		s.meta.Model = p.ModelProvider
	}
	if s.meta.GitBranch == "" {
		s.meta.GitBranch = p.Git.Branch
	}
	if link, ok := parseSubagentLink(p.Source); ok {
		s.link = link
		s.meta.IsSubagent = true
	}
}

func (s *scanState) applyTurnContext(p *scanPayload) {
	if p.CWD != "" {
		s.meta.CWD = p.CWD
	}
	if p.Model != "" {
		s.meta.Model = p.Model
	}
}

func (s *scanState) applyResponseItem(p *scanPayload) {
	switch p.ItemType {
	case responseTypeMessage:
		s.recordMessage(classifyResponseMessage(p.Role, p.Content))
	case responseTypeFunctionCall, responseTypeCustomToolCall, responseTypeWebSearchCall:
		s.recordToolCall(p)
	}
}

func (s *scanState) recordMessage(message visibleMessage, ok bool) {
	if !ok || message.text == "" || message.isAgentDivider || message.visibility == conv.MessageVisibilityHiddenSystem {
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

func (s *scanState) recordToolCall(p *scanPayload) {
	name := scanToolName(p)
	if name == "" {
		return
	}
	if s.meta.ToolCounts == nil {
		s.meta.ToolCounts = make(map[string]int, 2)
	}
	s.meta.ToolCounts[name]++
}

func (s *scanState) applyEvent(p *scanPayload) {
	switch p.ItemType {
	case eventTypeTokenCount:
		s.meta.TotalUsage = usageFromScanPayload(p)
	case eventTypeUserMessage:
		s.recordMessage(classifyEventUserMessage(p.Message))
	case eventTypeAgentMessage:
		s.recordMessage(classifyEventAssistantMessage(p.Message))
	case eventTypeTaskComplete:
		s.recordMessage(classifyTaskCompleteMessage(p.LastAgentMessage))
	}
}

func (s *scanState) rollout() (scannedRollout, bool, error) {
	if s.meta.ID == "" {
		return scannedRollout{}, false, nil
	}

	meta := s.meta
	if meta.Timestamp.IsZero() {
		meta.Timestamp = parseTimestamp(s.firstRawTS)
	}
	meta.LastTimestamp = parseTimestamp(s.lastRawTS)
	if meta.LastTimestamp.IsZero() {
		meta.LastTimestamp = meta.Timestamp
	}
	meta.Project = conv.Project{DisplayName: conv.ProjectName(meta.CWD)}
	if len(meta.ToolCounts) == 0 {
		meta.ToolCounts = nil
	}
	if meta.Slug == "" {
		meta.Slug = slugFromThreadID(meta.ID)
	}

	return scannedRollout{meta: meta, link: s.link}, true, nil
}
