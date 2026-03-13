package claude

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

func parseAndIndexLine(
	ctx context.Context,
	line []byte,
	recType role,
	toolCallIndex map[string]parsedToolCall,
) (parsedMessage, bool) {
	switch recType {
	case roleUser:
		msg, ok := parseParsedUserMessage(line)
		if !ok {
			return parsedMessage{}, false
		}
		for i, result := range msg.toolResults {
			if toolCall, found := toolCallIndex[result.toolUseID]; found {
				msg.toolResults[i].toolName = toolCall.name
				msg.toolResults[i].toolSummary = toolCall.summary
			}
		}
		return msg, true
	case roleAssistant:
		msg, ok := parseParsedAssistantMessage(ctx, line)
		if !ok {
			return parsedMessage{}, false
		}
		for _, toolCall := range msg.toolCalls {
			if toolCall.id != "" {
				toolCallIndex[toolCall.id] = toolCall
			}
		}
		return msg, true
	}
	return parsedMessage{}, false
}

func parseSessionMessagesDetailed(ctx context.Context, filePath string) ([]parsedMessage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = file.Close() }()

	var messages []parsedMessage
	toolCallIndex := make(map[string]parsedToolCall)
	for line, err := range jsonlLines(file, jsonlScanBufferSize) {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("parseSessionMessagesDetailed_ctx: %w", err)
		}
		if err != nil {
			return nil, fmt.Errorf("parseSessionMessagesDetailed_jsonlLines: %w", err)
		}

		recType := role(extractType(line))
		if msg, ok := parseAndIndexLine(ctx, line, recType, toolCallIndex); ok {
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

func parseConversationMessagesDetailed(ctx context.Context, conv conversation) ([]parsedMessage, tokenUsage, error) {
	if err := ctx.Err(); err != nil {
		return nil, tokenUsage{}, fmt.Errorf("parseConversationMessagesDetailed_ctx: %w", err)
	}

	paths := conv.FilePaths()
	if len(paths) == 0 {
		return nil, tokenUsage{}, nil
	}

	results, err := parseConversationPathsParallel(ctx, paths)
	if err != nil {
		return nil, tokenUsage{}, fmt.Errorf("parseConversationPathsParallel: %w", err)
	}

	totalMessages := 0
	for _, result := range results {
		totalMessages += len(result.messages)
	}

	allMessages := make([]parsedMessage, 0, totalMessages)
	for _, result := range results {
		if result.ok {
			allMessages = append(allMessages, result.messages...)
		}
	}
	return allMessages, aggregateUsage(allMessages), nil
}

func parseConversationPathsParallel(ctx context.Context, paths []string) ([]parsedSessionMessagesResult, error) {
	results := make([]parsedSessionMessagesResult, len(paths))
	limit := min(len(paths), 4)
	sem := semaphore.NewWeighted(int64(limit))
	group, groupCtx := errgroup.WithContext(ctx)
	log := zerolog.Ctx(ctx)

	for i := range paths {
		index := i
		path := paths[i]

		group.Go(func() error {
			if err := sem.Acquire(groupCtx, 1); err != nil {
				return fmt.Errorf("sem.Acquire_%s: %w", path, err)
			}
			defer sem.Release(1)

			msgs, err := parseSessionMessagesDetailed(groupCtx, path)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return fmt.Errorf("parseSessionMessagesDetailed_%s: %w", path, err)
				}
				log.Debug().Err(err).Msgf("parseSessionMessagesDetailed failed for %s", path)
				return nil
			}

			results[index] = parsedSessionMessagesResult{messages: msgs, ok: true}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("errgroup.Wait: %w", err)
	}
	return results, nil
}

func parseParsedUserMessage(line []byte) (parsedMessage, bool) {
	var rec jsonRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return parsedMessage{}, false
	}

	var msg jsonMessage
	if err := json.Unmarshal(rec.Message, &msg); err != nil {
		return parsedMessage{}, false
	}

	content, toolResults := extractUserContent(msg.Content)
	if content == "" && len(toolResults) == 0 {
		return parsedMessage{}, false
	}
	messageRole, visibility := classifyUserText(content)

	var ts time.Time
	if rec.Timestamp != "" {
		ts, _ = time.Parse(time.RFC3339Nano, rec.Timestamp)
	}

	var plans []plan
	if len(rec.ToolUseResult) > 0 {
		if len(toolResults) == 1 {
			if patch := extractStructuredPatch(rec.ToolUseResult); patch != nil {
				toolResults[0].structuredPatch = patch
			}
		}
		if plan, ok := extractExitPlanResult(rec.ToolUseResult, ts); ok {
			plans = append(plans, plan)
		}
	}

	return parsedMessage{
		role:        messageRole,
		timestamp:   ts,
		text:        content,
		toolResults: toolResults,
		plans:       plans,
		visibility:  visibility,
		isSidechain: rec.IsSidechain,
	}, true
}
