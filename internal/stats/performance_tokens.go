package stats

import conv "github.com/rkuska/carn/internal/conversation"

func performanceSessionTotalTokens(session performanceSession) int {
	if session.meta == nil {
		return 0
	}

	usage := session.meta.TotalUsage
	if session.provider == conv.ProviderCodex {
		return usage.InputTokens + usage.OutputTokens
	}

	return usage.TotalTokens()
}

func performanceSessionPromptTokens(session performanceSession) int {
	if session.meta == nil {
		return 0
	}

	usage := session.meta.TotalUsage
	if session.provider == conv.ProviderCodex {
		return usage.InputTokens
	}

	return usage.InputTokens +
		usage.CacheCreationInputTokens +
		usage.CacheReadInputTokens
}
