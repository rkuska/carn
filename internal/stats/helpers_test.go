package stats

import (
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type sessionMeta = conv.SessionMeta
type session = conv.Session
type message = conv.Message

func testMeta(
	id string,
	timestamp time.Time,
	options ...func(*conv.SessionMeta),
) conv.SessionMeta {
	meta := conv.SessionMeta{
		ID:               id,
		Slug:             id,
		Timestamp:        timestamp,
		LastTimestamp:    timestamp,
		Project:          conv.Project{DisplayName: "proj"},
		Model:            "claude-sonnet-4",
		MainMessageCount: 1,
		MessageCount:     1,
	}
	for _, option := range options {
		option(&meta)
	}
	return meta
}

func withLastTimestamp(ts time.Time) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.LastTimestamp = ts
	}
}

func withProject(name string) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.Project = conv.Project{DisplayName: name}
	}
}

func withModel(model string) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.Model = model
	}
}

func withMainMessages(count int) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.MainMessageCount = count
		meta.MessageCount = count
	}
}

func withUsage(input, cacheWrite, cacheRead, output int) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.TotalUsage = conv.TokenUsage{
			InputTokens:              input,
			CacheCreationInputTokens: cacheWrite,
			CacheReadInputTokens:     cacheRead,
			OutputTokens:             output,
		}
	}
}

func withRoleCounts(user, assistant int) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.UserMessageCount = user
		meta.AssistantMessageCount = assistant
	}
}

func withToolCounts(counts map[string]int) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.ToolCounts = counts
	}
}

func withToolErrorCounts(counts map[string]int) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.ToolErrorCounts = counts
	}
}

func testSession(id string, messages []conv.Message) conv.Session {
	return conv.Session{
		Meta: conv.SessionMeta{
			ID:        id,
			Slug:      id,
			Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			Project:   conv.Project{DisplayName: "proj"},
			Model:     "claude-sonnet-4",
		},
		Messages: messages,
	}
}

func assistantUsageMessage(input, output int) conv.Message {
	return conv.Message{
		Role: conv.RoleAssistant,
		Usage: conv.TokenUsage{
			InputTokens:  input,
			OutputTokens: output,
		},
	}
}

func userMessage() conv.Message {
	return conv.Message{Role: conv.RoleUser}
}
