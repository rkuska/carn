package claude

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

func findSubagentFiles(parentFilePath string) []string {
	base := strings.TrimSuffix(parentFilePath, ".jsonl")
	pattern := filepath.Join(base, "subagents", "agent-*.jsonl")
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
		if msg.role == roleUser && msg.text != "" && !isSystemInterrupt(msg.text) {
			return truncate(msg.text, maxFirstMessage)
		}
	}
	return title
}

func firstTimestamp(messages []parsedMessage) time.Time {
	for _, msg := range messages {
		if !msg.timestamp.IsZero() {
			return msg.timestamp
		}
	}
	return time.Time{}
}

func findInsertPosition(messages []parsedMessage, anchor time.Time) int {
	if anchor.IsZero() {
		return len(messages)
	}

	pos := 0
	for i, msg := range messages {
		if !msg.timestamp.IsZero() && !msg.timestamp.After(anchor) {
			pos = i + 1
		}
	}
	return pos
}

func parseConversationWithSubagents(ctx context.Context, conv conversation) (sessionFull, error) {
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
