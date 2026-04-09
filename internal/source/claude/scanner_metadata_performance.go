package claude

func accumulateUserPerformanceStats(line []byte, stats *scanStats) {
	thinkingMetadata, ok, err := jsonRawField(line, "thinkingMetadata")
	if err != nil || !ok {
		return
	}
	maxThinkingTokens, _, err := jsonIntField(thinkingMetadata, "maxThinkingTokens")
	if err != nil {
		return
	}
	stats.performance.MaxThinkingTokens = max(stats.performance.MaxThinkingTokens, maxThinkingTokens)
}

func (s *scanStats) recordToolCall(call toolCall, toolUseID string) {
	if call.Name == "" {
		return
	}
	addCount(&s.toolCounts, call.Name, 1)
	if call.Action.Type != "" {
		addCount(&s.actionCounts, string(call.Action.Type), 1)
	}
	if toolUseID == "" {
		return
	}
	if s.toolCallByID == nil {
		s.toolCallByID = make(map[string]toolCall, 2)
	}
	s.toolCallByID[toolUseID] = call
}

func (s *scanStats) recordToolError(toolUseID string) {
	call := s.toolCallByID[toolUseID]
	if call.Name == "" {
		return
	}
	addCount(&s.toolErrorCounts, call.Name, 1)
	if call.Action.Type != "" {
		addCount(&s.actionErrorCounts, string(call.Action.Type), 1)
	}
}

func (s *scanStats) recordToolReject(toolUseID string) {
	call := s.toolCallByID[toolUseID]
	if call.Name == "" {
		return
	}
	addCount(&s.toolRejectCounts, call.Name, 1)
	if call.Action.Type != "" {
		addCount(&s.actionRejects, string(call.Action.Type), 1)
	}
}

func addCount(counts *map[string]int, key string, value int) {
	if key == "" || value == 0 {
		return
	}
	if *counts == nil {
		*counts = make(map[string]int, 2)
	}
	(*counts)[key] += value
}

func normalizeSessionPerformanceMeta(meta sessionPerformanceMeta) sessionPerformanceMeta {
	if len(meta.APIErrorCounts) == 0 {
		meta.APIErrorCounts = nil
	}
	if len(meta.StopReasonCounts) == 0 {
		meta.StopReasonCounts = nil
	}
	if len(meta.PhaseCounts) == 0 {
		meta.PhaseCounts = nil
	}
	if len(meta.EffortCounts) == 0 {
		meta.EffortCounts = nil
	}
	if len(meta.ServerToolUseCounts) == 0 {
		meta.ServerToolUseCounts = nil
	}
	if len(meta.ServiceTierCounts) == 0 {
		meta.ServiceTierCounts = nil
	}
	if len(meta.SpeedCounts) == 0 {
		meta.SpeedCounts = nil
	}
	return meta
}
