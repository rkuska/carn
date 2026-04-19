package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	src "github.com/rkuska/carn/internal/source"
)

// parseRecord is a flat struct that merges the outer JSONL record with the
// nested message object. A single json.Decode fills both levels, eliminating
// the intermediate json.RawMessage copy for the "message" field.
// Used in both the parse phase and metadata scan path.
type parseRecord struct {
	Type                 string          `json:"type"`
	SessionID            string          `json:"sessionId"`
	Slug                 string          `json:"slug"`
	CWD                  string          `json:"cwd"`
	GitBranch            string          `json:"gitBranch"`
	Version              string          `json:"version"`
	Timestamp            string          `json:"timestamp"`
	IsSidechain          bool            `json:"isSidechain"`
	IsMeta               bool            `json:"isMeta"`
	ToolUseResult        json.RawMessage `json:"toolUseResult"`
	ThinkingMetadata     json.RawMessage `json:"thinkingMetadata"`
	Subtype              string          `json:"subtype"`
	DurationMS           int             `json:"durationMs"`
	RetryAttempt         int             `json:"retryAttempt"`
	RetryInMS            float64         `json:"retryInMs"`
	MaxRetries           int             `json:"maxRetries"`
	Error                json.RawMessage `json:"error"`
	CompactMetadata      json.RawMessage `json:"compactMetadata"`
	MicrocompactMetadata json.RawMessage `json:"microcompactMetadata"`
	Message              struct {
		Role       string          `json:"role"`
		Content    json.RawMessage `json:"content"`
		Model      string          `json:"model"`
		StopReason string          `json:"stop_reason"`
		Usage      *jsonUsage      `json:"usage"`
	} `json:"message"`
}

// parseContext holds reusable JSON decode containers to avoid per-line heap
// allocations. The rec struct is zeroed before each record parse via reset().
type parseContext struct {
	rec           parseRecord
	toolCallIndex map[string]toolCall
}

func (pc *parseContext) reset() {
	pc.rec = parseRecord{}
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
		for i, tc := range msg.message.ToolCalls {
			if i < len(toolCallIDs) && toolCallIDs[i] != "" {
				pc.toolCallIndex[toolCallIDs[i]] = tc
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
	messages := make([]parsedMessage, 0, 32)
	if err := visitSessionMessages(ctx, filePath, pc, func(msg parsedMessage) {
		messages = append(messages, msg)
	}); err != nil {
		return nil, fmt.Errorf("visitSessionMessages: %w", err)
	}
	return messages, nil
}

func visitSessionMessages(
	ctx context.Context,
	filePath string,
	pc *parseContext,
	visit func(parsedMessage),
) error {
	file, err := os.Open(filePath)
	if err != nil {
		err = fmt.Errorf("os.Open: %w", err)
		if errors.Is(err, fs.ErrNotExist) {
			return src.MarkMalformedRawData(err)
		}
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			zerolog.Ctx(ctx).Warn().Err(closeErr).Msg("file.Close")
		}
	}()

	br, ok := parseReaderPool.Get().(*bufio.Reader)
	if !ok {
		br = bufio.NewReaderSize(nil, jsonlParseBufferSize)
	}
	br.Reset(file)
	defer parseReaderPool.Put(br)

	pc.resetToolCallIndex()
	return visitSessionReaderMessages(ctx, br, pc, visit)
}

func visitSessionReaderMessages(
	ctx context.Context,
	br *bufio.Reader,
	pc *parseContext,
	visit func(parsedMessage),
) error {
	var overflow []byte
	for {
		line, nextOverflow, done, err := readNextSessionLine(ctx, br, overflow)
		overflow = nextOverflow
		if err != nil {
			return err
		}
		if len(line) == 0 {
			if done {
				return nil
			}
			continue
		}

		pc.reset()
		if err := parseRecordLine(line, &pc.rec); err != nil {
			return src.MarkMalformedRawData(fmt.Errorf("visitSessionReaderMessages_parseRecordLine: %w", err))
		}
		if msg, ok := parseAndIndexLine(ctx, pc); ok {
			visit(msg)
		}
		if done {
			return nil
		}
	}
}

func readNextSessionLine(
	ctx context.Context,
	br *bufio.Reader,
	overflow []byte,
) ([]byte, []byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, overflow, false, fmt.Errorf("readNextSessionLine_ctx: %w", err)
	}

	line, nextOverflow, err := readJSONLLine(br, overflow)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, nextOverflow, false, src.MarkMalformedRawData(
			fmt.Errorf("readNextSessionLine_readJSONLLine: %w", err),
		)
	}
	return line, nextOverflow, errors.Is(err, io.EOF), nil
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
		if tc, found := toolCallIndex[toolUseIDs[i]]; found {
			toolResults[i].ToolName = tc.Name
			toolResults[i].ToolSummary = tc.Summary
			toolResults[i].Action = tc.Action
		}
	}
}

func extractUserPlans(raw json.RawMessage, timestamp time.Time) []plan {
	if extracted, ok := extractExitPlanResult(raw, timestamp); ok {
		return []plan{extracted}
	}
	return nil
}
