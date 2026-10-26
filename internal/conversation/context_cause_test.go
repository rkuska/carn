package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextSeriesStepCause(t *testing.T) {
	t.Parallel()

	read := ToolCall{Name: "Read", Summary: "a.go"}
	bash := ToolCall{Name: "Bash"}
	sidechainCall := Message{Role: RoleAssistant, IsSidechain: true, ToolCalls: []ToolCall{{Name: "Grep", Summary: "x"}}}

	messages := []Message{
		ctxUser(),            // 0
		ctxAssistant(40_000), // 1 step 0
		{Role: RoleAssistant, ToolCalls: []ToolCall{read, bash}}, // 2
		sidechainCall,                                      // 3
		ctxAssistant(70_000),                               // 4 step 1
		{Role: RoleUser, Text: "next question"},            // 5
		ctxAssistant(75_000),                               // 6 step 2
		{Role: RoleAssistant, ToolCalls: []ToolCall{bash}}, // 7
		ctxAssistant(80_000),                               // 8 step 3
		{Role: RoleAssistant, ToolCalls: []ToolCall{read}}, // 9
		ctxAssistant(81_000),                               // 10 step 4
		ctxAssistant(82_000),                               // 11 step 5
		sidechainCall,                                      // 12
		ctxAssistant(83_000),                               // 13 step 6
	}

	tests := []struct {
		name         string
		step         int
		conversation Conversation
		want         string
	}{
		{name: "first step", step: 0, want: "system prompt"},
		{name: "out of range", step: 99, want: "system prompt"},
		{name: "first call and the count of the others", step: 1, want: "Read: a.go +1 more"},
		{name: "a user prompt without tool calls", step: 2, want: "user prompt"},
		{name: "single call without a summary", step: 3, want: "Bash"},
		{name: "single call with a summary", step: 4, want: "Read: a.go"},
		{name: "model output", step: 5, want: "model output"},
		{name: "sidechain calls are ignored", step: 6, want: "model output"},
		{
			name: "a subagent conversation counts its own sidechain", step: 6,
			conversation: ctxClaudeConversation(SessionMeta{IsSubagent: true}), want: "Grep: x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			series := NewContextSeries(tt.conversation, messages)

			assert.Equal(t, tt.want, series.StepCause(messages, tt.step))
		})
	}
}

func TestContextSeriesStepCauseOfAReset(t *testing.T) {
	t.Parallel()

	messages := []Message{ctxAssistant(150_000), ctxAssistant(20_000)}
	compacted := ctxClaudeConversation(SessionMeta{Performance: SessionPerformanceMeta{CompactionCount: 1}})

	assert.Equal(t, "reset", NewContextSeries(ctxClaudeConversation(), messages).StepCause(messages, 1))
	assert.Equal(t, "compaction", NewContextSeries(compacted, messages).StepCause(messages, 1))
}
