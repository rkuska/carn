package codex

import (
	"context"
	"fmt"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type loadState struct {
	meta                  conv.SessionMeta
	link                  subagentLink
	callMeta              map[string]toolEventMeta
	messages              []parsedMessage
	pendingUsage          conv.TokenUsage
	thinkingParts         []string
	pendingHiddenThinking bool
	pendingCalls          []conv.ToolCall
	pendingResults        []conv.ToolResult
	pendingPlans          []conv.Plan
	pendingTimestamp      time.Time
	usageTargetIndex      int
	readEvidence          map[string]struct{}
	currentEffort         string
	pendingPerformance    conv.MessagePerformanceMeta
}

func loadConversation(ctx context.Context, conversation conv.Conversation) (conv.Session, error) {
	if len(conversation.Sessions) == 0 {
		return conv.Session{}, fmt.Errorf("loadConversation: %w", errMissingPath)
	}

	parent, err := loadRollout(ctx, conversation.Sessions[0])
	if err != nil {
		return conv.Session{}, fmt.Errorf("loadRollout_parent: %w", err)
	}

	linked, err := loadLinkedTranscripts(ctx, conversation.Sessions[1:])
	if err != nil {
		return conv.Session{}, fmt.Errorf("loadLinkedTranscripts: %w", err)
	}

	return conv.Session{
		Meta:     parent.meta,
		Messages: projectParsedMessages(mergeSubagentTranscripts(parent, linked)),
	}, nil
}

func loadRollout(ctx context.Context, meta conv.SessionMeta) (rolloutTranscript, error) {
	path := meta.FilePath
	if path == "" {
		return rolloutTranscript{}, fmt.Errorf("loadRollout: %w", errMissingPath)
	}

	state := newLoadState(meta)
	if err := visitRolloutRecords(ctx, path, func(recordType string, payload []byte, timestamp string) {
		state.applyRecord(recordType, payload, timestamp)
	}); err != nil {
		return rolloutTranscript{}, fmt.Errorf("visitRolloutRecords: %w", err)
	}
	return state.transcript(), nil
}

func newLoadState(meta conv.SessionMeta) loadState {
	return loadState{
		meta:             meta,
		callMeta:         make(map[string]toolEventMeta),
		messages:         make([]parsedMessage, 0),
		thinkingParts:    make([]string, 0),
		pendingCalls:     make([]conv.ToolCall, 0),
		pendingResults:   make([]conv.ToolResult, 0),
		pendingPlans:     make([]conv.Plan, 0),
		usageTargetIndex: -1,
		readEvidence:     make(map[string]struct{}),
	}
}
