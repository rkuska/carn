package conversation

// claudeContextTiers are the context windows Claude models ship with, smallest
// first. Claude does not record the window, so it is inferred from the largest
// prompt a conversation reached.
var claudeContextTiers = [...]int{200_000, 1_000_000}

// resolveContextWindow returns the Context Window of conversation given the
// largest prompt (peak) of its Context Series.
//
// A window recorded by the provider (Codex model_context_window) always wins.
// Claude falls back to the smallest tier that holds the peak. Everything else,
// and a Claude peak beyond every tier, scales to the peak and stays unknown.
func resolveContextWindow(conversation Conversation, peak int) ContextWindow {
	recorded := 0
	for _, session := range conversation.Sessions {
		recorded = max(recorded, session.Performance.ModelContextWindow)
	}
	if recorded > 0 {
		return ContextWindow{Tokens: recorded, Known: true}
	}
	if peak <= 0 {
		return ContextWindow{}
	}
	if conversationProvider(conversation) == ProviderClaude {
		for _, tier := range claudeContextTiers {
			if peak <= tier {
				return ContextWindow{Tokens: tier, Known: true}
			}
		}
	}
	return ContextWindow{Tokens: peak}
}

func conversationProvider(conversation Conversation) Provider {
	if conversation.Ref.Provider != "" {
		return conversation.Ref.Provider
	}
	return conversation.firstSession().Provider
}
