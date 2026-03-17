package claude

import (
	"bufio"
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

// parseRecord is a flat struct that merges the outer JSONL record with the
// nested message object. A single json.Decode fills both levels, eliminating
// the intermediate json.RawMessage copy for the "message" field.
// Used only in the parse phase; the metadata scan path uses metadataRecord.
type parseRecord struct {
	Type          string          `json:"type"`
	SessionID     string          `json:"sessionId"`
	Slug          string          `json:"slug"`
	CWD           string          `json:"cwd"`
	GitBranch     string          `json:"gitBranch"`
	Version       string          `json:"version"`
	Timestamp     string          `json:"timestamp"`
	IsSidechain   bool            `json:"isSidechain"`
	IsMeta        bool            `json:"isMeta"`
	ToolUseResult json.RawMessage `json:"toolUseResult"`
	Message       struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
		Model   string          `json:"model"`
		Usage   *jsonUsage      `json:"usage"`
	} `json:"message"`
}

// parseContext holds reusable JSON decode containers to avoid per-line heap
// allocations. The rec struct is zeroed before each decode via reset().
// The blocks slice retains its backing array across calls — json.Unmarshal
// reuses the capacity, only overwriting fields present in each JSON record.
type parseContext struct {
	rec           parseRecord
	blocks        []contentBlock
	toolCallIndex map[string]toolCall
}

func (pc *parseContext) reset() {
	pc.rec = parseRecord{}
	pc.blocks = pc.blocks[:0]
}

func (pc *parseContext) resetToolCallIndex() {
	if pc.toolCallIndex == nil {
		pc.toolCallIndex = make(map[string]toolCall)
	} else {
		clear(pc.toolCallIndex)
	}
}

func parseAndIndexLine(
	ctx context.Context,
	pc *parseContext,
) (parsedMessage, bool) {
	switch role(pc.rec.Type) {
	case roleUser:
		return parseParsedUserMessage(pc)
	case roleAssistant:
		msg, toolCallIDs, ok := parseParsedAssistantMessage(ctx, pc)
		if !ok {
			return parsedMessage{}, false
		}
		for i, toolCall := range msg.message.ToolCalls {
			if i < len(toolCallIDs) && toolCallIDs[i] != "" {
				pc.toolCallIndex[toolCallIDs[i]] = toolCall
			}
		}
		return msg, true
	}
	return parsedMessage{}, false
}

func parseSessionMessagesDetailed(ctx context.Context, filePath string) ([]parsedMessage, error) {
	var pc parseContext
	return parseSessionWithContext(ctx, filePath, &pc)
}

func parseSessionWithContext(ctx context.Context, filePath string, pc *parseContext) ([]parsedMessage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = file.Close() }()

	br := parseReaderPool.Get().(*bufio.Reader)
	br.Reset(file)
	defer parseReaderPool.Put(br)

	messages := make([]parsedMessage, 0, 32)
	pc.resetToolCallIndex()
	dec := json.NewDecoder(br)
	for dec.More() {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("parseSessionWithContext_ctx: %w", err)
		}
		pc.reset()
		if err := dec.Decode(&pc.rec); err != nil {
			return nil, fmt.Errorf("parseSessionWithContext_decode: %w", err)
		}
		if msg, ok := parseAndIndexLine(ctx, pc); ok {
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

func parseConversationMessagesDetailed(ctx context.Context, conv conversation) ([]parsedMessage, tokenUsage, error) {
	if err := ctx.Err(); err != nil {
		return nil, tokenUsage{}, fmt.Errorf("parseConversationMessagesDetailed_ctx: %w", err)
	}

	if len(conv.Sessions) == 1 {
		var pc parseContext
		messages, err := parseSessionWithContext(ctx, conv.Sessions[0].FilePath, &pc)
		if err != nil {
			return nil, tokenUsage{}, fmt.Errorf("parseSessionWithContext: %w", err)
		}
		return messages, aggregateUsage(messages), nil
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

			// parseContext is reused across sequential file parses within
			// the same goroutine — the blocks slice retains its backing array.
			var pc parseContext
			msgs, err := parseSessionWithContext(groupCtx, path, &pc)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return fmt.Errorf("parseSessionWithContext_%s: %w", path, err)
				}
				log.Debug().Err(err).Msgf("parseSessionWithContext failed for %s", path)
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

func parseParsedUserMessage(pc *parseContext) (parsedMessage, bool) {
	content, toolResults, toolUseIDs := extractUserContentWithToolUseIDs(pc.rec.Message.Content)
	if content == "" && len(toolResults) == 0 {
		return parsedMessage{}, false
	}
	messageRole, visibility := classifyUserText(content)
	timestamp := parseRecordTimestamp(pc.rec.Timestamp)
	toolResults = applyStructuredPatch(pc.rec.ToolUseResult, toolResults)
	linkToolResults(toolResults, toolUseIDs, pc.toolCallIndex)
	plans := extractUserPlans(pc.rec.ToolUseResult, timestamp)

	return parsedMessage{
		message: message{
			Role:        messageRole,
			Text:        content,
			ToolResults: toolResults,
			Plans:       plans,
			Visibility:  visibility,
			IsSidechain: pc.rec.IsSidechain,
		},
		timestamp: timestamp,
	}, true
}

func applyStructuredPatch(raw json.RawMessage, toolResults []toolResult) []toolResult {
	if len(toolResults) != 1 || len(raw) == 0 {
		return toolResults
	}
	if patch := extractStructuredPatch(raw); patch != nil {
		toolResults[0].StructuredPatch = patch
	}
	return toolResults
}

func linkToolResults(
	toolResults []toolResult,
	toolUseIDs []string,
	toolCallIndex map[string]toolCall,
) {
	for i := range toolResults {
		if i >= len(toolUseIDs) {
			return
		}
		if toolCall, found := toolCallIndex[toolUseIDs[i]]; found {
			toolResults[i].ToolName = toolCall.Name
			toolResults[i].ToolSummary = toolCall.Summary
		}
	}
}

func extractUserPlans(raw json.RawMessage, timestamp time.Time) []plan {
	if extracted, ok := extractExitPlanResult(raw, timestamp); ok {
		return []plan{extracted}
	}
	return nil
}
