package claude

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func findSubagentFiles(parentFilePath string) []string {
	base := strings.TrimSuffix(parentFilePath, ".jsonl")
	subagentDir := filepath.Join(base, "subagents")
	if _, err := os.Stat(subagentDir); err != nil {
		return nil
	}
	pattern := filepath.Join(subagentDir, "agent-*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	return matches
}

func loadLinkedTranscripts(ctx context.Context, meta sessionMeta) []parsedLinkedTranscript {
	subFiles := findSubagentFiles(meta.FilePath)
	if len(subFiles) == 0 {
		return nil
	}

	log := zerolog.Ctx(ctx)
	linked := make([]parsedLinkedTranscript, 0, len(subFiles))
	for _, filePath := range subFiles {
		subMessages, err := parseSessionMessagesDetailed(ctx, filePath)
		if err != nil {
			log.Debug().Err(err).Msgf("skipping subagent file %s", filePath)
			continue
		}
		if len(subMessages) == 0 {
			continue
		}

		linked = append(linked, parsedLinkedTranscript{
			kind:     linkedTranscriptKindSubagent,
			title:    linkedTranscriptTitle(subMessages),
			anchor:   firstTimestamp(subMessages),
			messages: subMessages,
		})
	}

	sort.SliceStable(linked, func(i, j int) bool {
		if linked[i].anchor.IsZero() {
			return false
		}
		if linked[j].anchor.IsZero() {
			return true
		}
		return linked[i].anchor.Before(linked[j].anchor)
	})
	return linked
}

func linkedTranscriptTitle(messages []parsedMessage) string {
	title := "Subagent"
	for _, msg := range messages {
		if msg.message.Role == roleUser && msg.message.Text != "" && !isSystemInterrupt(msg.message.Text) {
			return conv.Truncate(msg.message.Text, maxFirstMessage)
		}
	}
	return title
}

func firstTimestamp(messages []parsedMessage) time.Time {
	return src.FirstNonZeroTime(messages, func(msg parsedMessage) time.Time {
		return msg.timestamp
	})
}

func parseConversationWithoutLinkedTranscripts(ctx context.Context, conv conversation) (sessionFull, error) {
	messages, totalUsage, err := parseConversationMessagesProjected(ctx, conv)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return sessionFull{}, fmt.Errorf("parseConversation_ctx: %w", err)
		}
		return sessionFull{}, fmt.Errorf("parseConversationMessagesProjected: %w", err)
	}

	meta := conv.Sessions[0]
	meta.TotalUsage = totalUsage
	deduplicateMessagePlans(messages)
	return sessionFull{
		Meta:     meta,
		Messages: messages,
	}, nil
}

func hasLinkedTranscripts(conv conversation) bool {
	for _, session := range conv.Sessions {
		if len(findSubagentFiles(session.FilePath)) > 0 {
			return true
		}
	}
	return false
}

func parseConversationWithSubagents(ctx context.Context, conv conversation) (sessionFull, error) {
	if !hasLinkedTranscripts(conv) {
		return parseConversationWithoutLinkedTranscripts(ctx, conv)
	}

	baseMessages, totalUsage, err := parseConversationMessagesDetailed(ctx, conv)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return sessionFull{}, fmt.Errorf("parseConversation_ctx: %w", err)
		}
		return sessionFull{}, fmt.Errorf("parseConversationMessagesDetailed: %w", err)
	}

	linked := make([]parsedLinkedTranscript, 0, len(conv.Sessions))
	for _, path := range conv.FilePaths() {
		meta := sessionMeta{FilePath: path, Project: conv.Project}
		linked = append(linked, loadLinkedTranscripts(ctx, meta)...)
	}

	meta := conv.Sessions[0]
	meta.TotalUsage = totalUsage
	deduplicatePlans(baseMessages)
	return sessionFull{
		Meta:     meta,
		Messages: projectConversationTranscript(baseMessages, linked),
	}, nil
}
