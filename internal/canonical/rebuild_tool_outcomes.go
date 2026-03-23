package canonical

import (
	"context"
	"errors"
	"fmt"

	conv "github.com/rkuska/carn/internal/conversation"
)

func enrichConversationToolOutcomes(
	ctx context.Context,
	sources sourceRegistry,
	convValue conversation,
	session sessionFull,
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

	enriched := convValue
	enriched.Sessions = append([]sessionMeta(nil), convValue.Sessions...)
	for i, meta := range convValue.Sessions {
		loaded, err := source.LoadSession(ctx, convValue, meta)
		if err != nil {
			return conversation{}, sessionFull{}, fmt.Errorf("source.LoadSession_%s: %w", meta.ID, err)
		}
		applyToolOutcomeCounts(&meta, conv.DeriveToolOutcomeCounts(loaded.Messages))
		enriched.Sessions[i] = meta
	}

	if len(enriched.Sessions) > 0 {
		session.Meta.ToolCounts = enriched.Sessions[0].ToolCounts
		session.Meta.ToolErrorCounts = enriched.Sessions[0].ToolErrorCounts
		session.Meta.ToolRejectCounts = enriched.Sessions[0].ToolRejectCounts
	}
	return enriched, session, nil
}

func applyToolOutcomeCounts(meta *sessionMeta, counts conv.ToolOutcomeCounts) {
	meta.ToolCounts = counts.Calls
	meta.ToolErrorCounts = counts.Errors
	meta.ToolRejectCounts = counts.Rejections
}
