package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ctxUsage(prompt int) TokenUsage {
	return TokenUsage{InputTokens: 3, CacheReadInputTokens: prompt - 3, OutputTokens: 100}
}

func ctxAssistant(prompt int) Message {
	return Message{Role: RoleAssistant, Text: "reply", Usage: ctxUsage(prompt)}
}

func ctxUser() Message {
	return Message{Role: RoleUser, Text: "prompt"}
}

func ctxClaudeConversation(sessions ...SessionMeta) Conversation {
	return Conversation{Ref: Ref{Provider: ProviderClaude, ID: "ctx"}, Sessions: sessions}
}

func stepPrompts(series ContextSeries) []int {
	prompts := make([]int, len(series.Steps))
	for i, step := range series.Steps {
		prompts[i] = step.PromptTokens
	}
	return prompts
}

func stepIndices(series ContextSeries) []int {
	indices := make([]int, len(series.Steps))
	for i, step := range series.Steps {
		indices[i] = step.MessageIndex
	}
	return indices
}

func TestNewContextSeriesStepRules(t *testing.T) {
	t.Parallel()

	sidechain := ctxAssistant(90_000)
	sidechain.IsSidechain = true
	divider := Message{Role: RoleUser, Text: "Subagent", IsAgentDivider: true}
	dividerWithUsage := ctxAssistant(95_000)
	dividerWithUsage.IsAgentDivider = true
	hidden := ctxAssistant(96_000)
	hidden.Visibility = MessageVisibilityHiddenSystem
	userWithUsage := ctxUser()
	userWithUsage.Usage = ctxUsage(97_000)
	noUsage := Message{Role: RoleAssistant, Text: "no usage"}

	tests := []struct {
		name         string
		conversation Conversation
		messages     []Message
		wantIndices  []int
		wantPrompts  []int
	}{
		{
			name:         "empty",
			conversation: ctxClaudeConversation(),
			messages:     nil,
			wantIndices:  []int{},
			wantPrompts:  []int{},
		},
		{
			name:         "claude split response keeps the first usage record",
			conversation: ctxClaudeConversation(),
			messages: []Message{
				ctxUser(),
				ctxAssistant(40_000),
				{Role: RoleAssistant, Text: "continued"},
				{Role: RoleAssistant, Text: "more"},
				ctxUser(),
				ctxAssistant(55_000),
			},
			wantIndices: []int{1, 5},
			wantPrompts: []int{40_000, 55_000},
		},
		{
			name:         "sidechain divider hidden user and usage free messages never start a step",
			conversation: ctxClaudeConversation(),
			messages: []Message{
				ctxUser(),
				ctxAssistant(40_000),
				divider,
				dividerWithUsage,
				sidechain,
				hidden,
				userWithUsage,
				noUsage,
				ctxAssistant(60_000),
			},
			wantIndices: []int{1, 8},
			wantPrompts: []int{40_000, 60_000},
		},
		{
			name: "a subagent conversation counts its own sidechain messages",
			conversation: Conversation{
				Ref:      Ref{Provider: ProviderClaude, ID: "agent"},
				Sessions: []SessionMeta{{IsSubagent: true}},
			},
			messages:    []Message{ctxUser(), sidechain},
			wantIndices: []int{1},
			wantPrompts: []int{90_000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			series := NewContextSeries(tt.conversation, tt.messages)

			assert.Equal(t, tt.wantIndices, stepIndices(series))
			assert.Equal(t, tt.wantPrompts, stepPrompts(series))
		})
	}
}

func TestNewContextSeriesEmptyHasNoPeak(t *testing.T) {
	t.Parallel()

	series := NewContextSeries(ctxClaudeConversation(), []Message{ctxUser()})

	assert.Empty(t, series.Steps)
	assert.Equal(t, -1, series.Peak)
	assert.Equal(t, 0, series.Resets)
	assert.Equal(t, ContextWindow{}, series.Window)
}

func TestNewContextSeriesDeltaGrowthAndNextIndex(t *testing.T) {
	t.Parallel()

	messages := []Message{
		ctxUser(),                  // 0
		ctxAssistant(40_000),       // 1
		{Role: RoleUser},           // 2
		ctxAssistant(70_000),       // 3
		{Role: RoleAssistant},      // 4
		ctxAssistant(20_000),       // 5 reset
		ctxAssistant(24_000),       // 6
		{Role: RoleUser, Text: ""}, // 7
	}

	series := NewContextSeries(ctxClaudeConversation(), messages)

	require.Len(t, series.Steps, 4)
	assert.Equal(t, []ContextStep{
		{MessageIndex: 1, NextMessageIndex: 3, PromptTokens: 40_000, Delta: 40_000, Growth: 30_000},
		{MessageIndex: 3, NextMessageIndex: 5, PromptTokens: 70_000, Delta: 30_000, Growth: 0},
		{MessageIndex: 5, NextMessageIndex: 6, PromptTokens: 20_000, Delta: -50_000, Growth: 4_000, Reset: true},
		{MessageIndex: 6, NextMessageIndex: 8, PromptTokens: 24_000, Delta: 4_000, Growth: 0},
	}, series.Steps)
	assert.Equal(t, 1, series.Peak)
	assert.Equal(t, 1, series.Resets)
}

func TestNewContextSeriesPeakPrefersFirstMaximum(t *testing.T) {
	t.Parallel()

	series := NewContextSeries(ctxClaudeConversation(), []Message{
		ctxAssistant(40_000),
		ctxAssistant(80_000),
		ctxAssistant(80_000),
	})

	assert.Equal(t, 1, series.Peak)
}

func TestNewContextSeriesResetBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		previous  int
		current   int
		wantReset bool
	}{
		{name: "below half and above the minimum drop", previous: 100_000, current: 49_999, wantReset: true},
		{name: "exactly half is not a reset", previous: 100_000, current: 50_000, wantReset: false},
		{name: "drop of exactly the minimum resets", previous: 15_000, current: 5_000, wantReset: true},
		{name: "drop just under the minimum does not reset", previous: 15_000, current: 5_001, wantReset: false},
		{name: "small contexts never reset", previous: 9_000, current: 1_000, wantReset: false},
		{name: "growth is never a reset", previous: 40_000, current: 90_000, wantReset: false},
		{name: "mild drop is not a reset", previous: 200_000, current: 150_000, wantReset: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			series := NewContextSeries(ctxClaudeConversation(), []Message{
				ctxAssistant(tt.previous),
				ctxAssistant(tt.current),
			})

			require.Len(t, series.Steps, 2)
			assert.Equal(t, tt.wantReset, series.Steps[1].Reset)
			assert.Equal(t, tt.current-tt.previous, series.Steps[1].Delta)
			wantResets := 0
			if tt.wantReset {
				wantResets = 1
			}
			assert.Equal(t, wantResets, series.Resets)
		})
	}
}

func TestNewContextSeriesSuppressesGrowthBeforeReset(t *testing.T) {
	t.Parallel()

	series := NewContextSeries(ctxClaudeConversation(), []Message{
		ctxAssistant(150_000),
		ctxAssistant(20_000),
		ctxAssistant(30_000),
	})

	require.Len(t, series.Steps, 3)
	assert.Equal(t, 0, series.Steps[0].Growth, "the drop is a reset, not negative growth")
	assert.Equal(t, 10_000, series.Steps[1].Growth)
}

func TestNewContextSeriesCompacted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sessions []SessionMeta
		want     bool
	}{
		{name: "no sessions", want: false},
		{name: "clean session", sessions: []SessionMeta{{}}, want: false},
		{
			name:     "claude compaction",
			sessions: []SessionMeta{{}, {Performance: SessionPerformanceMeta{CompactionCount: 1}}},
			want:     true,
		},
		{
			name:     "micro compaction",
			sessions: []SessionMeta{{Performance: SessionPerformanceMeta{MicroCompactionCount: 2}}},
			want:     true,
		},
		{
			name:     "codex context compacted",
			sessions: []SessionMeta{{Performance: SessionPerformanceMeta{ContextCompactedCount: 1}}},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			series := NewContextSeries(ctxClaudeConversation(tt.sessions...), []Message{ctxAssistant(1_000)})

			assert.Equal(t, tt.want, series.Compacted)
		})
	}
}

func TestContextSeriesStepAt(t *testing.T) {
	t.Parallel()

	series := NewContextSeries(ctxClaudeConversation(), []Message{
		ctxUser(),            // 0
		ctxAssistant(40_000), // 1 step 0
		{Role: RoleUser},     // 2
		ctxAssistant(70_000), // 3 step 1
		{Role: RoleUser},     // 4
	})

	tests := []struct {
		name         string
		messageIndex int
		wantStep     int
		wantOK       bool
	}{
		{name: "before the first step", messageIndex: 0, wantOK: false},
		{name: "negative index", messageIndex: -1, wantOK: false},
		{name: "on the first step message", messageIndex: 1, wantStep: 0, wantOK: true},
		{name: "between steps", messageIndex: 2, wantStep: 0, wantOK: true},
		{name: "on the second step message", messageIndex: 3, wantStep: 1, wantOK: true},
		{name: "after the last step", messageIndex: 4, wantStep: 1, wantOK: true},
		{name: "past the end", messageIndex: 99, wantStep: 1, wantOK: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			step, ok := series.StepAt(tt.messageIndex)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantStep, step)
		})
	}
}

func TestContextSeriesStepAtEmpty(t *testing.T) {
	t.Parallel()

	_, ok := NewContextSeries(ctxClaudeConversation(), nil).StepAt(0)

	assert.False(t, ok)
}

func TestContextSeriesGrowthThresholds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		window    ContextWindow
		wantMin   int
		wantLarge int
	}{
		{name: "200k", window: ContextWindow{Tokens: 200_000, Known: true}, wantMin: 5_000, wantLarge: 20_000},
		{name: "258.4k", window: ContextWindow{Tokens: 258_400, Known: true}, wantMin: 5_000, wantLarge: 20_000},
		{name: "1M", window: ContextWindow{Tokens: 1_000_000, Known: true}, wantMin: 10_000, wantLarge: 50_000},
		{name: "unknown uses the peak", window: ContextWindow{Tokens: 700_000}, wantMin: 7_000, wantLarge: 35_000},
		{name: "no window", window: ContextWindow{}, wantMin: 5_000, wantLarge: 20_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			series := ContextSeries{Window: tt.window, Peak: -1}

			assert.Equal(t, tt.wantMin, series.MinGrowth())
			assert.Equal(t, tt.wantLarge, series.LargeGrowth())
		})
	}
}

func TestContextSeriesIsNotable(t *testing.T) {
	t.Parallel()

	series := ContextSeries{
		Window: ContextWindow{Tokens: 200_000, Known: true},
		Steps: []ContextStep{
			{Delta: 40_000},
			{Delta: 5_000},
			{Delta: 4_999},
			{Delta: -80_000, Reset: true},
			{Delta: 30_000},
		},
	}

	tests := []struct {
		name string
		step int
		want bool
	}{
		{name: "first step is the baseline", step: 0, want: false},
		{name: "delta at the threshold", step: 1, want: true},
		{name: "delta below the threshold", step: 2, want: false},
		{name: "reset", step: 3, want: true},
		{name: "large delta", step: 4, want: true},
		{name: "negative index", step: -1, want: false},
		{name: "past the end", step: 5, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, series.IsNotable(tt.step))
		})
	}
}

func TestContextSeriesShareAndPressure(t *testing.T) {
	t.Parallel()

	known200k := ContextWindow{Tokens: 200_000, Known: true}

	tests := []struct {
		name          string
		window        ContextWindow
		tokens        int
		wantShare     float64
		wantPressured bool
	}{
		{name: "half of a known window", window: known200k, tokens: 100_000, wantShare: 0.5},
		{name: "exactly 80 percent", window: known200k, tokens: 160_000, wantShare: 0.8, wantPressured: true},
		{name: "just under 80 percent", window: known200k, tokens: 159_999, wantShare: 0.799995},
		{name: "unknown window scales to the peak", window: ContextWindow{Tokens: 100_000}, tokens: 100_000, wantShare: 1},
		{name: "no window", window: ContextWindow{}, tokens: 100_000, wantShare: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			series := ContextSeries{Window: tt.window, Peak: -1}

			assert.InDelta(t, tt.wantShare, series.Share(tt.tokens), 1e-9)
			assert.Equal(t, tt.wantPressured, series.UnderPressure(tt.tokens))
		})
	}
}
