package codex

import (
	"context"
	"fmt"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

type Source struct{}

func New() Source {
	return Source{}
}

func (Source) Provider() conv.Provider {
	return conv.ProviderCodex
}

func (Source) UsesScannedToolOutcomeCounts() bool {
	return true
}

func (Source) Scan(ctx context.Context, rawDir string) (src.ScanResult, error) {
	conversations, drift, malformedData, err := scanRollouts(ctx, rawDir)
	if err != nil {
		return src.ScanResult{}, fmt.Errorf("scan_scanRollouts: %w", err)
	}
	drift.Log(ctx, conv.ProviderCodex)
	return src.ScanResult{
		Conversations: conversations,
		Drift:         drift,
		MalformedData: malformedData,
	}, nil
}

func (Source) Load(ctx context.Context, conversation conv.Conversation) (conv.Session, error) {
	session, err := loadConversation(ctx, conversation)
	if err != nil {
		return conv.Session{}, fmt.Errorf("load_loadConversation: %w", err)
	}
	return session, nil
}

func (s Source) LoadSession(
	ctx context.Context,
	conversation conv.Conversation,
	meta conv.SessionMeta,
) (conv.Session, error) {
	session, err := s.Load(ctx, singleSessionConversation(conversation, meta))
	if err != nil {
		return conv.Session{}, fmt.Errorf("load: %w", err)
	}
	return session, nil
}

func singleSessionConversation(conversation conv.Conversation, meta conv.SessionMeta) conv.Conversation {
	return conv.Conversation{
		Ref:       conv.Ref{Provider: conversation.Ref.Provider, ID: meta.ID},
		Name:      conversation.Name,
		Project:   conversation.Project,
		Sessions:  []conv.SessionMeta{meta},
		PlanCount: conversation.PlanCount,
	}
}
