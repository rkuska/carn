package codex

import (
	"bytes"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func (s *scanState) observeRecordTimestamp(raw []byte) {
	raw = rawJSONStringInner(raw)
	if len(raw) == 0 {
		return
	}
	if len(s.firstRawTS) == 0 {
		s.firstRawTS = append(s.firstRawTS[:0], raw...)
	}
	if bytes.Compare(raw, s.lastRawTS) > 0 {
		s.lastRawTS = append(s.lastRawTS[:0], raw...)
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

func (s *scanState) recordToolCall(callID string, call conv.ToolCall) {
	name := call.Name
	if name == "" {
		return
	}
	if s.meta.ToolCounts == nil {
		s.meta.ToolCounts = make(map[string]int, 2)
	}
	s.meta.ToolCounts[name]++
	if !call.Action.IsZero() {
		if s.meta.ActionCounts == nil {
			s.meta.ActionCounts = make(map[string]int, 2)
		}
		s.meta.ActionCounts[string(call.Action.Type)]++
	}
	if callID != "" {
		if s.callByID == nil {
			s.callByID = make(map[string]conv.ToolCall, 2)
		}
		s.callByID[callID] = call
	}
	rememberReadEvidence(call.Action, s.readEvidence)
}

func (s *scanState) recordToolResult(callID string, outputRaw, statusRaw []byte) {
	if s.callByID == nil {
		return
	}
	call := s.callByID[callID]
	if call.Name == "" {
		return
	}
	if isCodexToolRejectRaw(outputRaw) {
		s.incrementRejectedToolCounts(call)
		return
	}
	if !hasCodexToolError(call.Name, outputRaw, statusRaw) {
		return
	}
	s.incrementErroredToolCounts(call)
}

func (s *scanState) rollout() (scannedRollout, bool, error) {
	if s.meta.ID == "" {
		return scannedRollout{drift: derefDriftReport(s.drift)}, false, nil
	}

	meta := s.finalizeRolloutMeta()
	return scannedRollout{meta: meta, link: s.link, drift: derefDriftReport(s.drift)}, true, nil
}

func (s *scanState) incrementRejectedToolCounts(call conv.ToolCall) {
	if s.meta.ToolRejectCounts == nil {
		s.meta.ToolRejectCounts = make(map[string]int, 2)
	}
	s.meta.ToolRejectCounts[call.Name]++
	if call.Action.IsZero() {
		return
	}
	if s.meta.ActionRejectCounts == nil {
		s.meta.ActionRejectCounts = make(map[string]int, 2)
	}
	s.meta.ActionRejectCounts[string(call.Action.Type)]++
}

func (s *scanState) incrementErroredToolCounts(call conv.ToolCall) {
	if s.meta.ToolErrorCounts == nil {
		s.meta.ToolErrorCounts = make(map[string]int, 2)
	}
	s.meta.ToolErrorCounts[call.Name]++
	if call.Action.IsZero() {
		return
	}
	if s.meta.ActionErrorCounts == nil {
		s.meta.ActionErrorCounts = make(map[string]int, 2)
	}
	s.meta.ActionErrorCounts[string(call.Action.Type)]++
}

func (s *scanState) finalizeRolloutMeta() conv.SessionMeta {
	meta := s.meta
	applyRolloutTimestamps(&meta, s.firstRawTS, s.lastRawTS)
	meta.Project = conv.Project{DisplayName: conv.ProjectName(meta.CWD)}
	normalizeRolloutCountMaps(&meta)
	if meta.Slug == "" {
		meta.Slug = slugFromThreadID(meta.ID)
	}
	return meta
}

func applyRolloutTimestamps(meta *conv.SessionMeta, firstRawTS, lastRawTS []byte) {
	if meta.Timestamp.IsZero() {
		meta.Timestamp = parseTimestamp(string(firstRawTS))
	}
	meta.LastTimestamp = parseTimestamp(string(lastRawTS))
	if meta.LastTimestamp.IsZero() {
		meta.LastTimestamp = meta.Timestamp
	}
}

func normalizeRolloutCountMaps(meta *conv.SessionMeta) {
	meta.ToolCounts = nilIfEmptyCountMap(meta.ToolCounts)
	meta.ToolErrorCounts = nilIfEmptyCountMap(meta.ToolErrorCounts)
	meta.ToolRejectCounts = nilIfEmptyCountMap(meta.ToolRejectCounts)
	meta.ActionCounts = nilIfEmptyCountMap(meta.ActionCounts)
	meta.ActionErrorCounts = nilIfEmptyCountMap(meta.ActionErrorCounts)
	meta.ActionRejectCounts = nilIfEmptyCountMap(meta.ActionRejectCounts)
}

func nilIfEmptyCountMap(counts map[string]int) map[string]int {
	if len(counts) == 0 {
		return nil
	}
	return counts
}

func derefDriftReport(report *src.DriftReport) src.DriftReport {
	if report == nil {
		return src.DriftReport{}
	}
	return *report
}
