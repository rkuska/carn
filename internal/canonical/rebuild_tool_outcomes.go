package canonical

import (
	"context"
	"errors"
	"fmt"

	conv "github.com/rkuska/carn/internal/conversation"
)

type scannedToolOutcomeSource interface {
	UsesScannedToolOutcomeCounts() bool
}

func enrichConversationToolOutcomes(
	ctx context.Context,
	sources sourceRegistry,
	convValue conversation,
	session sessionFull,
	preloadedSessions []sessionFull,
) (conversation, sessionFull, error) {
	if len(convValue.Sessions) == 0 {
		return convValue, session, nil
	}

	source, ok := sources.lookup(conversationProvider(convValue.Ref.Provider))
	if !ok {
		return conversation{}, sessionFull{}, fmt.Errorf(
			"enrichConversationToolOutcomes: %w",
			errors.New("provider is not registered"),
		)
	}
	if metadataSource, ok := any(source).(scannedToolOutcomeSource); ok && metadataSource.UsesScannedToolOutcomeCounts() {
		enriched, enrichedSession := applyScannedToolOutcomeCounts(convValue, session)
		return enriched, enrichedSession, nil
	}

	if len(preloadedSessions) == len(convValue.Sessions) && len(preloadedSessions) > 0 {
		enriched, enrichedSession := applyPreloadedToolOutcomeCounts(convValue, session, preloadedSessions)
		return enriched, enrichedSession, nil
	}

	enriched, err := loadToolOutcomeCounts(ctx, source, convValue)
	if err != nil {
		return conversation{}, sessionFull{}, err
	}
	enrichedSessionConv, enrichedSession := applyFirstSessionToolOutcomeCounts(enriched, session)
	return enrichedSessionConv, enrichedSession, nil
}

func applyPreloadedToolOutcomeCounts(
	convValue conversation,
	session sessionFull,
	preloadedSessions []sessionFull,
) (conversation, sessionFull) {
	enriched := copyConversationSessions(convValue)
	for i, meta := range convValue.Sessions {
		applyToolOutcomeCounts(&meta, conv.DeriveToolOutcomeCounts(preloadedSessions[i].Messages))
		enriched.Sessions[i] = meta
	}
	return applyFirstSessionToolOutcomeCounts(enriched, session)
}

func loadToolOutcomeCounts(
	ctx context.Context,
	source Source,
	convValue conversation,
) (conversation, error) {
	enriched := copyConversationSessions(convValue)
	for i, meta := range convValue.Sessions {
		loaded, err := source.LoadSession(ctx, convValue, meta)
		if err != nil {
			return conversation{}, fmt.Errorf("source.LoadSession_%s: %w", meta.ID, err)
		}
		applyToolOutcomeCounts(&meta, conv.DeriveToolOutcomeCounts(loaded.Messages))
		enriched.Sessions[i] = meta
	}
	return enriched, nil
}

func applyScannedToolOutcomeCounts(convValue conversation, session sessionFull) (conversation, sessionFull) {
	enriched := copyConversationSessions(convValue)
	if len(enriched.Sessions) == 0 {
		return enriched, session
	}

	return applyFirstSessionToolOutcomeCounts(enriched, session)
}

func copyConversationSessions(convValue conversation) conversation {
	enriched := convValue
	enriched.Sessions = append([]sessionMeta(nil), convValue.Sessions...)
	return enriched
}

func applyFirstSessionToolOutcomeCounts(
	enriched conversation,
	session sessionFull,
) (conversation, sessionFull) {
	if len(enriched.Sessions) == 0 {
		return enriched, session
	}
	session.Meta.ToolCounts = enriched.Sessions[0].ToolCounts
	session.Meta.ToolErrorCounts = enriched.Sessions[0].ToolErrorCounts
	session.Meta.ToolRejectCounts = enriched.Sessions[0].ToolRejectCounts
	return enriched, session
}

func applyToolOutcomeCounts(meta *sessionMeta, counts conv.ToolOutcomeCounts) {
	meta.ToolCounts = counts.Calls
	meta.ToolErrorCounts = counts.Errors
	meta.ToolRejectCounts = counts.Rejections
}
