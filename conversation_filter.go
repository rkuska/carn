package main

func (c conversation) hasConversationContent() bool {
	for _, session := range c.sessions {
		if session.hasConversationContent {
			return true
		}
	}
	return false
}

func filterRenderableConversations(convs []conversation) []conversation {
	filtered := make([]conversation, 0, len(convs))
	for _, conv := range convs {
		if conv.hasConversationContent() {
			filtered = append(filtered, conv)
		}
	}
	return filtered
}
