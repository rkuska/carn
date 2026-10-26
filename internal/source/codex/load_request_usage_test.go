package codex

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func codexAssistantTextLine(timestamp, text string) map[string]any {
	return map[string]any{
		"timestamp": timestamp,
		"type":      recordTypeResponseItem,
		"payload": map[string]any{
			"type": responseTypeMessage,
			"role": responseRoleAssistant,
			"content": []map[string]any{
				{"type": "output_text", "text": text},
			},
		},
	}
}

func codexReasoningLine(timestamp, summary string) map[string]any {
	return map[string]any{
		"timestamp": timestamp,
		"type":      recordTypeResponseItem,
		"payload": map[string]any{
			"type":    responseTypeReasoning,
			"summary": []map[string]any{{"type": "summary_text", "text": summary}},
		},
	}
}

func codexCallLines(timestamp, callID string) []map[string]any {
	return []map[string]any{
		{
			"timestamp": timestamp,
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":      responseTypeFunctionCall,
				"name":      "exec_command",
				"arguments": `{"cmd":"go test ./..."}`,
				"call_id":   callID,
			},
		},
		{
			"timestamp": timestamp,
			"type":      recordTypeResponseItem,
			"payload": map[string]any{
				"type":    responseTypeFunctionCallOutput,
				"call_id": callID,
				"output":  "ok\n",
				"status":  "completed",
			},
		},
	}
}

// codexRequestCount renders a token_count for one model request. The running
// total is the cumulative sum of all requests so far.
func codexRequestCount(timestamp string, total, last int) map[string]any {
	return codexTokenCountLine(
		timestamp,
		map[string]any{"input_tokens": total, "cached_input_tokens": 0, "output_tokens": total / 10},
		map[string]any{"input_tokens": last, "cached_input_tokens": 0, "output_tokens": last / 10},
	)
}

func one(line map[string]any) []map[string]any {
	return []map[string]any{line}
}

func promptsOf(messages []conv.Message) []int {
	prompts := make([]int, len(messages))
	for i, msg := range messages {
		prompts[i] = msg.Usage.PromptTokens()
	}
	return prompts
}

func TestLoadAttachesOneUsagePerModelRequest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		lines       []map[string]any
		wantPrompts []int
		wantCalls   []int
	}{
		{
			name: "tool only requests stay separate messages",
			lines: slices.Concat(
				one(codexSessionMetaLine("thread-requests")),
				one(codexUserMessageLine("2026-03-16T10:00:01Z", "Run it.")),
				codexCallLines("2026-03-16T10:00:02Z", "call-1"),
				one(codexRequestCount("2026-03-16T10:00:03Z", 1000, 1000)),
				codexCallLines("2026-03-16T10:00:04Z", "call-2"),
				one(codexRequestCount("2026-03-16T10:00:05Z", 2500, 1500)),
				one(codexAssistantTextLine("2026-03-16T10:00:06Z", "Done.")),
				one(codexRequestCount("2026-03-16T10:00:07Z", 4500, 2000)),
			),
			wantPrompts: []int{0, 1000, 1500, 2000},
			wantCalls:   []int{0, 1, 1, 0},
		},
		{
			name: "usage sits on the first message of the request",
			lines: slices.Concat(
				one(codexSessionMetaLine("thread-first-message")),
				one(codexUserMessageLine("2026-03-16T10:00:01Z", "Run it.")),
				one(codexReasoningLine("2026-03-16T10:00:02Z", "Plan the run.")),
				one(codexAssistantTextLine("2026-03-16T10:00:03Z", "Checking first.")),
				codexCallLines("2026-03-16T10:00:04Z", "call-1"),
				one(codexRequestCount("2026-03-16T10:00:05Z", 1000, 1000)),
				one(codexReasoningLine("2026-03-16T10:00:06Z", "Wrap up.")),
				one(codexAssistantTextLine("2026-03-16T10:00:07Z", "Done.")),
				one(codexRequestCount("2026-03-16T10:00:08Z", 2800, 1800)),
			),
			wantPrompts: []int{0, 1000, 0, 1800},
			wantCalls:   []int{0, 0, 1, 0},
		},
		{
			name: "repeated token count with an unchanged total is ignored",
			lines: slices.Concat(
				one(codexSessionMetaLine("thread-duplicate-count")),
				one(codexUserMessageLine("2026-03-16T10:00:01Z", "Hello.")),
				one(codexAssistantTextLine("2026-03-16T10:00:02Z", "Hi.")),
				one(codexRequestCount("2026-03-16T10:00:03Z", 1000, 1000)),
				one(codexRequestCount("2026-03-16T10:00:04Z", 1000, 1000)),
			),
			wantPrompts: []int{0, 1000},
			wantCalls:   []int{0, 0},
		},
		{
			name: "token count without content keeps the first usage",
			lines: slices.Concat(
				one(codexSessionMetaLine("thread-empty-window")),
				one(codexUserMessageLine("2026-03-16T10:00:01Z", "Hello.")),
				one(codexAssistantTextLine("2026-03-16T10:00:02Z", "Hi.")),
				one(codexRequestCount("2026-03-16T10:00:03Z", 1000, 1000)),
				one(codexRequestCount("2026-03-16T10:00:04Z", 2500, 1500)),
			),
			wantPrompts: []int{0, 1000},
			wantCalls:   []int{0, 0},
		},
		{
			name: "duplicate answer event after the token count does not add a message",
			lines: slices.Concat(
				one(codexSessionMetaLine("thread-duplicate-answer")),
				one(codexUserMessageLine("2026-03-16T10:00:01Z", "Hello.")),
				one(codexAssistantTextLine("2026-03-16T10:00:02Z", "Hi.")),
				one(codexRequestCount("2026-03-16T10:00:03Z", 1000, 1000)),
				one(map[string]any{
					"timestamp": "2026-03-16T10:00:04Z",
					"type":      recordTypeEventMsg,
					"payload": map[string]any{
						"type":               eventTypeTaskComplete,
						"last_agent_message": "Hi.",
					},
				}),
			),
			wantPrompts: []int{0, 1000},
			wantCalls:   []int{0, 0},
		},
		{
			name: "a user message closes the request window",
			lines: slices.Concat(
				one(codexSessionMetaLine("thread-window-reset")),
				one(codexUserMessageLine("2026-03-16T10:00:01Z", "Hello.")),
				one(codexAssistantTextLine("2026-03-16T10:00:02Z", "Hi.")),
				one(codexUserMessageLine("2026-03-16T10:00:03Z", "More?")),
				one(codexAssistantTextLine("2026-03-16T10:00:04Z", "Sure.")),
				one(codexRequestCount("2026-03-16T10:00:05Z", 1200, 1200)),
			),
			wantPrompts: []int{0, 0, 0, 1200},
			wantCalls:   []int{0, 0, 0, 0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			session := loadSingleCodexTestSession(t, tc.lines)

			assert.Equal(t, tc.wantPrompts, promptsOf(session.Messages))
			calls := make([]int, len(session.Messages))
			for i, msg := range session.Messages {
				calls[i] = len(msg.ToolCalls)
			}
			assert.Equal(t, tc.wantCalls, calls)
		})
	}
}

func TestLoadFlagsMergedSubagentMessagesAsSidechain(t *testing.T) {
	t.Parallel()

	rawDir := copyCodexFixtureDir(t)
	scanResult, err := New().Scan(context.Background(), rawDir)
	require.NoError(t, err)

	var main conv.Conversation
	for _, conversation := range scanResult.Conversations {
		if conversation.Sessions[0].ID == "019cexample-main" {
			main = conversation
		}
	}
	session, err := New().Load(context.Background(), main)
	require.NoError(t, err)

	flags := make([]string, len(session.Messages))
	for i, msg := range session.Messages {
		flags[i] = fmt.Sprintf("%s divider=%t sidechain=%t", msg.Role, msg.IsAgentDivider, msg.IsSidechain)
	}
	assert.Equal(t, []string{
		"system divider=false sidechain=false",
		"system divider=false sidechain=false",
		"system divider=false sidechain=false",
		"user divider=false sidechain=false",
		"assistant divider=false sidechain=false",
		"user divider=true sidechain=false",
		"user divider=false sidechain=true",
		"assistant divider=false sidechain=true",
	}, flags)
}
