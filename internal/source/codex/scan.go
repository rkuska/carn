package codex

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

type scanState struct {
	meta         conv.SessionMeta
	firstRawTS   []byte
	lastRawTS    []byte
	lastRole     conv.Role
	lastText     string
	callNameByID map[string]string
	link         subagentLink
	drift        *src.DriftReport
}

func scanRollouts(ctx context.Context, rawDir string) ([]conv.Conversation, src.DriftReport, error) {
	paths, err := listJSONLPaths(rawDir)
	if err != nil {
		return nil, src.DriftReport{}, fmt.Errorf("listJSONLPaths: %w", err)
	}
	if len(paths) == 0 {
		return nil, src.DriftReport{}, nil
	}

	rollouts, drift, err := scanRolloutsParallel(ctx, paths)
	if err != nil {
		return nil, src.DriftReport{}, fmt.Errorf("scanRolloutsParallel: %w", err)
	}
	return groupRollouts(rollouts), drift, nil
}

func scanRollout(path string) (_ scannedRollout, _ bool, retErr error) {
	file, br, err := openReader(path)
	if err != nil {
		return scannedRollout{}, false, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("file.Close: %w", closeErr)
		}
	}()
	defer readerPool.Put(br)

	state := newScanState(path)
	if scanErr := scanRolloutReader(br, &state); scanErr != nil {
		return scannedRollout{drift: derefDriftReport(state.drift)}, false, fmt.Errorf("scanRolloutReader: %w", scanErr)
	}

	return state.rollout()
}

func scanRolloutReader(br *bufio.Reader, state *scanState) error {
	var overflow []byte
	for {
		line, nextOverflow, err := readScanLine(br, overflow)
		overflow = nextOverflow

		if len(line) > 0 {
			if scanErr := scanRolloutLine(line, state); scanErr != nil {
				return fmt.Errorf("scanRolloutLine: %w", scanErr)
			}
		}

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("readScanLine: %w", err)
		}
	}
}

func newScanState(path string) scanState {
	drift := src.NewDriftReport()
	return scanState{
		meta: conv.SessionMeta{
			FilePath: path,
		},
		drift: &drift,
	}
}

func (s *scanState) observeRecordTimestamp(raw []byte) {
	raw = rawJSONStringInner(raw)
	if len(raw) == 0 {
		return
	}
	if len(s.firstRawTS) == 0 {
		s.firstRawTS = bytes.Clone(raw)
	}
	if bytes.Compare(raw, s.lastRawTS) > 0 {
		s.lastRawTS = bytes.Clone(raw)
	}
}

func (s *scanState) recordMessage(message visibleMessage, ok bool) {
	if !shouldRecordMessage(message, ok) {
		return
	}

	if s.lastRole == message.role && s.lastText == message.text {
		return
	}
	s.lastRole = message.role
	s.lastText = message.text
	s.meta.MessageCount++
	s.meta.MainMessageCount++
	switch message.role {
	case conv.RoleUser:
		s.meta.UserMessageCount++
	case conv.RoleAssistant:
		s.meta.AssistantMessageCount++
	case conv.RoleSystem:
	}
	if message.role == conv.RoleUser && s.meta.FirstMessage == "" {
		s.meta.FirstMessage = message.text
	}
}

func shouldRecordMessage(message visibleMessage, ok bool) bool {
	return ok &&
		message.text != "" &&
		!message.isAgentDivider &&
		message.visibility != conv.MessageVisibilityHiddenSystem
}

func (s *scanState) recordToolCall(callID, name string) {
	if name == "" {
		return
	}
	if s.meta.ToolCounts == nil {
		s.meta.ToolCounts = make(map[string]int, 2)
	}
	s.meta.ToolCounts[name]++
	if callID != "" {
		if s.callNameByID == nil {
			s.callNameByID = make(map[string]string, 2)
		}
		s.callNameByID[callID] = name
	}
}

func (s *scanState) recordToolResult(callID string, outputRaw, statusRaw []byte) {
	if s.callNameByID == nil {
		return
	}
	name := s.callNameByID[callID]
	if name == "" {
		return
	}
	if isCodexToolRejectRaw(outputRaw) {
		if s.meta.ToolRejectCounts == nil {
			s.meta.ToolRejectCounts = make(map[string]int, 2)
		}
		s.meta.ToolRejectCounts[name]++
		return
	}
	if !hasCodexToolError(name, outputRaw, statusRaw) {
		return
	}
	if s.meta.ToolErrorCounts == nil {
		s.meta.ToolErrorCounts = make(map[string]int, 2)
	}
	s.meta.ToolErrorCounts[name]++
}

func (s *scanState) rollout() (scannedRollout, bool, error) {
	if s.meta.ID == "" {
		return scannedRollout{drift: derefDriftReport(s.drift)}, false, nil
	}

	meta := s.meta
	if meta.Timestamp.IsZero() {
		meta.Timestamp = parseTimestamp(string(s.firstRawTS))
	}
	meta.LastTimestamp = parseTimestamp(string(s.lastRawTS))
	if meta.LastTimestamp.IsZero() {
		meta.LastTimestamp = meta.Timestamp
	}
	meta.Project = conv.Project{DisplayName: conv.ProjectName(meta.CWD)}
	if len(meta.ToolCounts) == 0 {
		meta.ToolCounts = nil
	}
	if len(meta.ToolErrorCounts) == 0 {
		meta.ToolErrorCounts = nil
	}
	if len(meta.ToolRejectCounts) == 0 {
		meta.ToolRejectCounts = nil
	}
	if meta.Slug == "" {
		meta.Slug = slugFromThreadID(meta.ID)
	}

	return scannedRollout{meta: meta, link: s.link, drift: derefDriftReport(s.drift)}, true, nil
}

func derefDriftReport(report *src.DriftReport) src.DriftReport {
	if report == nil {
		return src.DriftReport{}
	}
	return *report
}
