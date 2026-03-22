package claude

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

func parseSessionProjectedWithContext(
	ctx context.Context,
	filePath string,
	pc *parseContext,
) ([]message, tokenUsage, error) {
	return parseSessionProjectedWithContextInto(ctx, filePath, pc, nil)
}

func parseSessionProjectedWithContextInto(
	ctx context.Context,
	filePath string,
	pc *parseContext,
	messages []message,
) ([]message, tokenUsage, error) {
	if messages == nil {
		messages = make([]message, 0, 32)
	} else {
		messages = messages[:0]
	}
	var totalUsage tokenUsage
	if err := visitSessionMessages(ctx, filePath, pc, func(msg parsedMessage) {
		m := msg.message
		m.Usage = msg.usage
		messages = append(messages, m)
		addUsage(&totalUsage, msg.usage)
	}); err != nil {
		return nil, tokenUsage{}, fmt.Errorf("visitSessionMessages: %w", err)
	}
	return messages, totalUsage, nil
}

func parseConversationMessagesProjected(ctx context.Context, conv conversation) ([]message, tokenUsage, error) {
	if err := ctx.Err(); err != nil {
		return nil, tokenUsage{}, fmt.Errorf("parseConversationMessagesProjected_ctx: %w", err)
	}

	switch len(conv.Sessions) {
	case 0:
		return nil, tokenUsage{}, nil
	case 1:
		return parseSingleConversationProjected(ctx, conv.Sessions[0].FilePath)
	}

	prealloc := make([]message, conv.TotalMessageCount())
	results, err := parseConversationPathsProjectedParallel(ctx, conv.Sessions, prealloc)
	if err != nil {
		return nil, tokenUsage{}, fmt.Errorf("parseConversationPathsProjectedParallel: %w", err)
	}
	return collectProjectedConversation(results, conv.Sessions, prealloc), aggregateProjectedUsage(results), nil
}

func parseSingleConversationProjected(
	ctx context.Context,
	filePath string,
) ([]message, tokenUsage, error) {
	var pc parseContext
	messages, usage, err := parseSessionProjectedWithContext(ctx, filePath, &pc)
	if err != nil {
		return nil, tokenUsage{}, fmt.Errorf("parseSessionProjectedWithContext: %w", err)
	}
	return messages, usage, nil
}

func collectProjectedConversation(
	results []parsedSessionProjectionResult,
	sessions []sessionMeta,
	prealloc []message,
) []message {
	actualTotal, exact := projectedMessageTotals(results, sessions)
	if exact {
		return prealloc[:actualTotal]
	}

	allMessages := make([]message, 0, actualTotal)
	for _, result := range results {
		if result.ok {
			allMessages = append(allMessages, result.messages...)
		}
	}
	return allMessages
}

func projectedMessageTotals(results []parsedSessionProjectionResult, sessions []sessionMeta) (int, bool) {
	actualTotal := 0
	exact := len(results) > 0
	for i, result := range results {
		if !result.ok {
			exact = false
			continue
		}
		actualTotal += len(result.messages)
		if len(result.messages) != sessions[i].MessageCount {
			exact = false
		}
	}
	return actualTotal, exact
}

func aggregateProjectedUsage(results []parsedSessionProjectionResult) tokenUsage {
	var totalUsage tokenUsage
	for _, result := range results {
		if result.ok {
			addUsage(&totalUsage, result.usage)
		}
	}
	return totalUsage
}

func addUsage(total *tokenUsage, usage tokenUsage) {
	total.InputTokens += usage.InputTokens
	total.CacheCreationInputTokens += usage.CacheCreationInputTokens
	total.CacheReadInputTokens += usage.CacheReadInputTokens
	total.OutputTokens += usage.OutputTokens
}

func parseConversationPathsProjectedParallel(
	ctx context.Context,
	sessions []sessionMeta,
	prealloc []message,
) ([]parsedSessionProjectionResult, error) {
	results := make([]parsedSessionProjectionResult, len(sessions))
	limit := min(len(sessions), 4)
	sem := semaphore.NewWeighted(int64(limit))
	group, groupCtx := errgroup.WithContext(ctx)
	log := zerolog.Ctx(ctx)
	offsets := sessionMessageOffsets(sessions)

	for i := range sessions {
		index := i
		session := sessions[i]

		group.Go(func() error {
			if err := sem.Acquire(groupCtx, 1); err != nil {
				return fmt.Errorf("sem.Acquire_%s: %w", session.FilePath, err)
			}
			defer sem.Release(1)

			var pc parseContext
			msgBuffer := prealloc[offsets[index] : offsets[index] : offsets[index]+session.MessageCount]
			msgs, usage, err := parseSessionProjectedWithContextInto(
				groupCtx,
				session.FilePath,
				&pc,
				msgBuffer,
			)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return fmt.Errorf("parseSessionProjectedWithContext_%s: %w", session.FilePath, err)
				}
				log.Debug().Err(err).Msgf("parseSessionProjectedWithContext failed for %s", session.FilePath)
				return nil
			}

			results[index] = parsedSessionProjectionResult{messages: msgs, usage: usage, ok: true}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("errgroup.Wait: %w", err)
	}
	return results, nil
}

func sessionMessageOffsets(sessions []sessionMeta) []int {
	offsets := make([]int, len(sessions))
	total := 0
	for i, session := range sessions {
		offsets[i] = total
		total += session.MessageCount
	}
	return offsets
}
