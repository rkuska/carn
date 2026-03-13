package claude

import (
	"sort"

	conv "github.com/rkuska/carn/internal/conversation"
)

type groupKey struct {
	dirName string
	slug    string
}

func groupConversations(sessions []scannedSession) []conversation {
	groups := make(map[groupKey][]scannedSession)
	var ungrouped []scannedSession

	for _, session := range sessions {
		if session.meta.IsSubagent || session.meta.Slug == "" {
			ungrouped = append(ungrouped, session)
			continue
		}
		groups[session.groupKey] = append(groups[session.groupKey], session)
	}

	conversations := make([]conversation, 0, len(groups)+len(ungrouped))
	for key, members := range groups {
		renderable := false
		metaMembers := make([]sessionMeta, len(members))
		for i, member := range members {
			metaMembers[i] = member.meta
			if member.hasConversationContent {
				renderable = true
			}
		}
		if !renderable {
			continue
		}
		sort.Slice(metaMembers, func(i, j int) bool {
			return metaMembers[i].Timestamp.Before(metaMembers[j].Timestamp)
		})
		conversations = append(conversations, conversation{
			Ref:      conversationRefForGroup(key),
			Name:     key.slug,
			Project:  metaMembers[0].Project,
			Sessions: metaMembers,
		})
	}

	for _, session := range ungrouped {
		if !session.hasConversationContent {
			continue
		}
		conversations = append(conversations, conversation{
			Ref:      conversationRefForPath(session.groupKey.slug),
			Name:     session.meta.Slug,
			Project:  session.meta.Project,
			Sessions: []sessionMeta{session.meta},
		})
	}

	return conversations
}

func conversationRefForGroup(key groupKey) conversationRef {
	return conversationRef{
		Provider: conv.ProviderClaude,
		ID:       "group:" + key.dirName + ":" + key.slug,
	}
}

func conversationRefForPath(relPath string) conversationRef {
	return conversationRef{
		Provider: conv.ProviderClaude,
		ID:       "path:" + relPath,
	}
}
