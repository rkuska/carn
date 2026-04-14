package stats

import (
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type sessionMeta = conv.SessionMeta
type session = conv.Session
type message = conv.Message
type conversation = conv.Conversation

func testMeta(
	id string,
	timestamp time.Time,
	options ...func(*conv.SessionMeta),
) conv.SessionMeta {
	meta := conv.SessionMeta{
		ID:               id,
		Provider:         conv.ProviderClaude,
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

func testConversation(provider conv.Provider, id string, sessions ...conv.SessionMeta) conv.Conversation {
	project := conv.Project{DisplayName: "proj"}
	if len(sessions) > 0 && sessions[0].Project.DisplayName != "" {
		project = sessions[0].Project
	}
	return conv.Conversation{
		Ref:      conv.Ref{Provider: provider, ID: id},
		Name:     id,
		Project:  project,
		Sessions: sessions,
	}
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

func withProvider(provider conv.Provider) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.Provider = provider
	}
}

func withVersion(version string) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.Version = version
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

func withActionCounts(counts map[string]int) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.ActionCounts = counts
	}
}

func withToolErrorCounts(counts map[string]int) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.ToolErrorCounts = counts
	}
}

func withToolRejectCounts(counts map[string]int) func(*conv.SessionMeta) {
	return func(meta *conv.SessionMeta) {
		meta.ToolRejectCounts = counts
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
	return assistantUsageMessageWithUsage(conv.TokenUsage{
		InputTokens:  input,
		OutputTokens: output,
	})
}

func assistantUsageMessageWithUsage(usage conv.TokenUsage) conv.Message {
	return conv.Message{
		Role:  conv.RoleAssistant,
		Usage: usage,
	}
}

func userMessage() conv.Message {
	return conv.Message{Role: conv.RoleUser, Text: "user"}
}

func sidechainUserMessage() conv.Message {
	return conv.Message{Role: conv.RoleUser, Text: "user", IsSidechain: true}
}

func userToolResultMessage() conv.Message {
	return conv.Message{
		Role:        conv.RoleUser,
		ToolResults: []conv.ToolResult{{ToolName: "Read", Content: "ok"}},
	}
}

func sidechainAssistantUsageMessage(input, output int) conv.Message {
	msg := assistantUsageMessage(input, output)
	msg.IsSidechain = true
	return msg
}

func systemMessage(text string) conv.Message {
	return conv.Message{Role: conv.RoleSystem, Text: text}
}

func agentDividerMessage() conv.Message {
	return conv.Message{IsAgentDivider: true}
}

func subagentSession(id string, messages []conv.Message) conv.Session {
	s := testSession(id, messages)
	s.Meta.IsSubagent = true
	return s
}
