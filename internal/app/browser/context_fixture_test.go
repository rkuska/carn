package browser

import (
	"fmt"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

// contextUsage builds usage whose prompt size equals prompt.
func contextUsage(prompt int) conv.TokenUsage {
	return conv.TokenUsage{InputTokens: 3, CacheReadInputTokens: prompt - 3, OutputTokens: 120}
}

// contextFixtureSession is a synthetic Claude conversation that exercises every
// message kind the viewer renders: split assistant responses, tool calls and
// results, a hidden system message, sidechain messages behind an agent divider,
// a plan, a growth spike and a context reset.
//
// Every message carries a unique "mNN" token in its text so tests can find the
// rendered line of a message. Context steps (prompt tokens):
//
//	step 0 → message 1  (30k, +15k growth after it: Growth Tag on the m02 Read call)
//	step 1 → message 4  (45k, +15k incoming, next step is a reset)
//	step 2 → message 12 (20k, reset, +40k growth after it: large Growth Tag on the m12 Edit call)
//	step 3 → message 14 (60k, +40k incoming, +3k growth after it)
//	step 4 → message 15 (63k, +3k incoming, same assistant group as step 3)
//	step 5 → message 17 (64k, +1k incoming)
func contextFixtureSession() conv.Session {
	read := conv.ToolCall{Name: "Read", Summary: "internal/parser.go"}
	bash := conv.ToolCall{Name: "Bash", Summary: "go test ./internal/parser/..."}
	edit := conv.ToolCall{Name: "Edit", Summary: "internal/parser.go"}
	messages := []conv.Message{
		{Role: conv.RoleUser, Text: "m00 Fix the parser"},
		{
			Role: conv.RoleAssistant, Text: "m01 I will look at the parser.", Thinking: "m01 plan the work",
			Usage: contextUsage(30_000),
		},
		{Role: conv.RoleAssistant, Text: "m02 reading", ToolCalls: []conv.ToolCall{read}},
		{Role: conv.RoleUser, ToolResults: []conv.ToolResult{{ToolName: "Read", Content: "m03 file body"}}},
		{
			Role: conv.RoleAssistant, Text: "m04 found it", ToolCalls: []conv.ToolCall{bash},
			Usage: contextUsage(45_000),
		},
		{Role: conv.RoleUser, ToolResults: []conv.ToolResult{{ToolName: "Bash", Content: "m05 test output"}}},
		{
			Role: conv.RoleSystem, Text: "m06 <command-name>/compact</command-name>",
			Visibility: conv.MessageVisibilityHiddenSystem,
		},
		{Role: conv.RoleAssistant, Text: "m07 side work", IsSidechain: true, Usage: contextUsage(90_000)},
		{Role: conv.RoleUser, Text: "m08 Subagent: lint", IsAgentDivider: true},
		{Role: conv.RoleUser, Text: "m09 lint the parser", IsSidechain: true},
		{Role: conv.RoleAssistant, Text: "m10 linted", IsSidechain: true, Usage: contextUsage(25_000)},
		{Role: conv.RoleUser, Text: "m11 Now add tests"},
		{
			Role: conv.RoleAssistant, Text: "m12 adding tests", ToolCalls: []conv.ToolCall{edit},
			Usage: contextUsage(20_000),
		},
		{
			Role:        conv.RoleUser,
			ToolResults: []conv.ToolResult{{ToolName: "Edit", Content: "m13 edited"}},
			Plans:       []conv.Plan{{FilePath: "plan.md", Content: "- add tests"}},
		},
		{Role: conv.RoleAssistant, Text: "m14 tests added", Usage: contextUsage(60_000)},
		{
			Role: conv.RoleAssistant, Text: "m15 summary", ToolCalls: []conv.ToolCall{read},
			Usage: contextUsage(63_000),
		},
		{Role: conv.RoleUser, ToolResults: []conv.ToolResult{{ToolName: "Read", Content: "m16 file body"}}},
		{Role: conv.RoleAssistant, Text: "m17 all done", Usage: contextUsage(64_000)},
	}
	meta := conv.SessionMeta{
		ID:        "context-fixture",
		Provider:  conv.ProviderClaude,
		Timestamp: time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC),
		Project:   conv.Project{DisplayName: "ctx/project"},
		Performance: conv.SessionPerformanceMeta{
			ModelContextWindow: 200_000,
			CompactionCount:    1,
		},
	}
	return conv.Session{Meta: meta, Messages: messages}
}

func contextFixtureConversation(session conv.Session) conv.Conversation {
	conversation := singleSessionConversation(session.Meta)
	conversation.Ref = conv.Ref{Provider: session.Meta.Provider, ID: session.Meta.ID}
	return conversation
}

func newContextTestViewer(width, height int) viewerModel {
	session := contextFixtureSession()
	return newViewerModel(session, contextFixtureConversation(session), "dark", "2006-01-02 15:04", width, height)
}

// contextToken is the unique marker a message carries in its text.
func contextToken(index int) string {
	return fmt.Sprintf("m%02d", index)
}

// allTranscriptOptions shows every optional block.
func allTranscriptOptions() transcriptOptions {
	return transcriptOptions{
		showThinking:      true,
		showTools:         true,
		showToolResults:   true,
		showSystem:        true,
		showContextLedger: true,
	}
}
