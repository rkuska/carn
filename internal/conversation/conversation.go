package conversation

import "time"

type Conversation struct {
	Ref           Ref
	Name          string
	Project       Project
	Sessions      []SessionMeta
	PlanCount     int
	SearchPreview string

	// Cached derived display fields. Conversation remains a value type;
	// callers opt into eager caching by calling PrecomputeDisplay on an address.
	displayCache *conversationDisplayCache
}

func (c Conversation) ID() string {
	if c.Ref.ID != "" {
		return c.Ref.ID
	}
	return c.firstSession().ID
}

func (c Conversation) CacheKey() string {
	if key := c.Ref.CacheKey(); key != "" {
		return key
	}
	return c.ID()
}

func (c Conversation) ResumeID() string {
	return c.latestPrimarySession().ID
}

func (c Conversation) ResumeCWD() string {
	return c.latestPrimarySession().CWD
}

func (c Conversation) ResumeTarget() ResumeTarget {
	return ResumeTarget{
		Provider: c.Ref.Provider,
		ID:       c.ResumeID(),
		CWD:      c.ResumeCWD(),
	}
}

func (c Conversation) Timestamp() time.Time {
	return c.firstSession().Timestamp
}

func (c Conversation) FilePaths() []string {
	paths := make([]string, len(c.Sessions))
	for i, s := range c.Sessions {
		paths[i] = s.FilePath
	}
	return paths
}

func (c Conversation) LatestFilePath() string {
	return c.latestPrimarySession().FilePath
}

func (c Conversation) FirstMessage() string {
	return c.firstSession().FirstMessage
}

func (c Conversation) TotalMessageCount() int {
	total := 0
	for _, s := range c.Sessions {
		total += s.MessageCount
	}
	return total
}

func (c Conversation) MainMessageCount() int {
	total := 0
	for _, s := range c.Sessions {
		total += s.MainMessageCount
	}
	return total
}

func (c Conversation) TotalTokenUsage() TokenUsage {
	var total TokenUsage
	for _, s := range c.Sessions {
		total.InputTokens += s.TotalUsage.InputTokens
		total.CacheCreationInputTokens += s.TotalUsage.CacheCreationInputTokens
		total.CacheReadInputTokens += s.TotalUsage.CacheReadInputTokens
		total.OutputTokens += s.TotalUsage.OutputTokens
	}
	return total
}

func (c Conversation) Duration() time.Duration {
	earliest := c.firstSession().Timestamp
	var latest time.Time
	for _, s := range c.Sessions {
		if s.LastTimestamp.After(latest) {
			latest = s.LastTimestamp
		}
	}
	if earliest.IsZero() || latest.IsZero() {
		return 0
	}
	return latest.Sub(earliest)
}

func (c Conversation) IsSubagent() bool {
	return c.firstSession().IsSubagent
}

func (c Conversation) Model() string {
	if m := c.firstSession().Model; m != "" {
		return m
	}
	for i := len(c.Sessions) - 1; i >= 0; i-- {
		if m := c.Sessions[i].Model; m != "" {
			return m
		}
	}
	return ""
}

func (c Conversation) Version() string {
	if v := c.latestPrimarySession().Version; v != "" {
		return v
	}
	for i := len(c.Sessions) - 1; i >= 0; i-- {
		if v := c.Sessions[i].Version; v != "" {
			return v
		}
	}
	return ""
}

func (c Conversation) GitBranch() string {
	return c.firstSession().GitBranch
}

func (c Conversation) PartCount() int {
	count := 0
	for _, session := range c.Sessions {
		if session.IsSubagent {
			continue
		}
		count++
	}
	if count == 0 {
		return len(c.Sessions)
	}
	return count
}

func (c Conversation) latestPrimarySession() SessionMeta {
	if len(c.Sessions) == 0 {
		return SessionMeta{}
	}
	for i := len(c.Sessions) - 1; i >= 0; i-- {
		if !c.Sessions[i].IsSubagent {
			return c.Sessions[i]
		}
	}
	return c.Sessions[len(c.Sessions)-1]
}

func (c Conversation) firstSession() SessionMeta {
	if len(c.Sessions) == 0 {
		return SessionMeta{}
	}
	return c.Sessions[0]
}
