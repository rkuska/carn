package claude

import (
	"context"
	"fmt"
)

func loadConversationBundle(ctx context.Context, conv conversation) (sessionFull, []sessionFull, error) {
	if len(conv.Sessions) == 0 {
		return sessionFull{}, nil, nil
	}

	if !hasLinkedTranscripts(conv) {
		session, sessions, complete, err := loadProjectedConversationBundle(ctx, conv)
		if err != nil {
			return sessionFull{}, nil, fmt.Errorf("loadProjectedConversationBundle: %w", err)
		}
		if complete {
			return session, sessions, nil
		}
	}

	session, err := parseConversationWithSubagents(ctx, conv)
	if err != nil {
		return sessionFull{}, nil, fmt.Errorf("parseConversationWithSubagents: %w", err)
	}

	sessions := make([]sessionFull, 0, len(conv.Sessions))
	for _, meta := range conv.Sessions {
		loaded, err := parseConversationWithSubagents(ctx, singleSessionConversation(conv, meta))
		if err != nil {
			return sessionFull{}, nil, fmt.Errorf("parseConversationWithSubagents_%s: %w", meta.ID, err)
		}
		sessions = append(sessions, loaded)
	}
	return session, sessions, nil
}

func loadProjectedConversationBundle(
	ctx context.Context,
	conv conversation,
) (sessionFull, []sessionFull, bool, error) {
	switch len(conv.Sessions) {
	case 0:
		return sessionFull{}, nil, true, nil
	case 1:
		session, err := parseConversationWithoutLinkedTranscripts(ctx, conv)
		if err != nil {
			return sessionFull{}, nil, false, fmt.Errorf("parseConversationWithoutLinkedTranscripts: %w", err)
		}
		return session, []sessionFull{session}, true, nil
	}

	prealloc := make([]message, conv.TotalMessageCount())
	results, err := parseConversationPathsProjectedParallel(ctx, conv.Sessions, prealloc)
	if err != nil {
		return sessionFull{}, nil, false, fmt.Errorf("parseConversationPathsProjectedParallel: %w", err)
	}
	if !projectedResultsComplete(results) {
		return sessionFull{}, nil, false, nil
	}

	sessions := make([]sessionFull, len(conv.Sessions))
	for i, meta := range conv.Sessions {
		messages := append([]message(nil), results[i].messages...)
		deduplicateMessagePlans(messages)
		meta.TotalUsage = results[i].usage
		sessions[i] = sessionFull{
			Meta:     meta,
			Messages: messages,
		}
	}

	allMessages := collectProjectedConversation(results, conv.Sessions, prealloc)
	deduplicateMessagePlans(allMessages)

	meta := conv.Sessions[0]
	meta.TotalUsage = aggregateProjectedUsage(results)
	return sessionFull{
		Meta:     meta,
		Messages: allMessages,
	}, sessions, true, nil
}

func projectedResultsComplete(results []parsedSessionProjectionResult) bool {
	for _, result := range results {
		if !result.ok {
			return false
		}
	}
	return true
}
