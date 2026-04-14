package browser

import (
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

func conversationProviderLabel(conversation conv.Conversation) string {
	return conversation.Ref.Provider.Label()
}

func providerPrefixedDescription(conversation conv.Conversation, desc string) string {
	label := conversationProviderLabel(conversation)
	desc = strings.TrimSpace(desc)
	switch {
	case label == "":
		return desc
	case desc == "":
		return label
	default:
		return label + "  " + desc
	}
}
