package stats

func performanceSessionTotalTokens(session performanceSession) int {
	if session.meta == nil {
		return 0
	}
	return session.meta.TotalUsage.TotalTokens()
}

func performanceSessionPromptTokens(session performanceSession) int {
	if session.meta == nil {
		return 0
	}
	u := session.meta.TotalUsage
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
}
