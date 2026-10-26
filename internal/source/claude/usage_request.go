package claude

// requestTracker remembers the last API response id seen per thread. Claude
// writes one record per content block of a response and repeats the same usage
// on each of them, so usage must be attributed to the first record of an id
// only. Records of one id are contiguous within a thread, which makes the last
// id sufficient.
type requestTracker struct {
	last [2]string // main thread, sidechain
}

// isNew reports whether id starts a response that has not been attributed yet
// and remembers it. Records without an id are always new.
func (t *requestTracker) isNew(id string, sidechain bool) bool {
	if id == "" {
		return true
	}
	slot := 0
	if sidechain {
		slot = 1
	}
	if t.last[slot] == id {
		return false
	}
	t.last[slot] = id
	return true
}

// requestUsage returns the usage of the current assistant record, or zero when
// an earlier record of the same API response already carries it.
func (pc *parseContext) requestUsage() tokenUsage {
	raw := pc.rec.Message.Usage
	if raw == nil {
		return tokenUsage{}
	}
	usage := tokenUsage{
		InputTokens:              raw.InputTokens,
		CacheCreationInputTokens: raw.CacheCreationInputTokens,
		CacheReadInputTokens:     raw.CacheReadInputTokens,
		OutputTokens:             raw.OutputTokens,
	}
	if usage == (tokenUsage{}) || !pc.requests.isNew(pc.rec.Message.ID, pc.rec.IsSidechain) {
		return tokenUsage{}
	}
	return usage
}

// extractMessageID returns message.id of a raw assistant record, or "" when the
// record has none.
func extractMessageID(line []byte) string {
	id, _, err := jsonStringField(line, "message", "id")
	if err != nil {
		return ""
	}
	return id
}
