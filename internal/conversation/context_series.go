package conversation

import "sort"

const (
	// contextResetRatio and contextResetMinDrop together mark a Context
	// Reset: the prompt fell below this share of the previous step and by at
	// least this many tokens.
	contextResetRatio   = 0.5
	contextResetMinDrop = 10_000

	minGrowthFloor        = 5_000
	minGrowthWindowShare  = 100 // window/100
	largeGrowthFloor      = 20_000
	largeGrowthWindowRate = 20 // window*5/100 == window/20
	pressureShareNumer    = 80
	pressureShareDenom    = 100
)

// ContextStep is one main-thread model request with recorded usage.
type ContextStep struct {
	// MessageIndex is the index in Session.Messages of the usage-carrying
	// message.
	MessageIndex int
	// NextMessageIndex is the MessageIndex of the next step, or the message
	// count for the last step.
	NextMessageIndex int
	// PromptTokens is Usage.PromptTokens() of the usage-carrying message.
	PromptTokens int
	// Delta is PromptTokens minus the previous step's; the first step's
	// Delta is its own PromptTokens.
	Delta int
	// Growth is the next step's PromptTokens minus this step's. It is zero
	// for the last step and when the next step is a Reset.
	Growth int
	// Reset marks a sharp drop from the previous step (compaction or a new
	// session part).
	Reset bool
}

// ContextWindow is the model's maximum context.
type ContextWindow struct {
	Tokens int
	// Known is true when Tokens was recorded or resolved from a model tier;
	// otherwise Tokens is the series peak.
	Known bool
}

// ContextSeries is the ordered Context Steps of a conversation plus the
// resolved Context Window. It is built from messages only, so it never depends
// on viewer toggles.
type ContextSeries struct {
	Steps     []ContextStep
	Window    ContextWindow
	Peak      int // index into Steps, -1 when empty
	Resets    int
	Compacted bool // any session in the conversation recorded a compaction
	// ownThread is set for a subagent conversation, whose sidechain messages
	// are its own main thread.
	ownThread bool
}

// NewContextSeries builds the Context Series of conversation from messages in
// one pass.
//
// A message starts a step when it is a visible, non-divider assistant message
// with recorded prompt tokens on the main thread. Everything else belongs to
// the step that precedes it. A subagent conversation is its own main thread, so
// its sidechain messages count.
func NewContextSeries(conversation Conversation, messages []Message) ContextSeries {
	series := ContextSeries{
		Peak:      -1,
		Compacted: conversationCompacted(conversation),
		ownThread: conversation.IsSubagent(),
	}
	for i := range messages {
		if prompt, ok := series.stepPrompt(messages[i]); ok {
			series = series.withStep(i, prompt)
		}
	}
	if n := len(series.Steps); n > 0 {
		series.Steps[n-1].NextMessageIndex = len(messages)
		series.Window = resolveContextWindow(conversation, series.Steps[series.Peak].PromptTokens)
	}
	return series
}

// OnThread reports whether msg belongs to the thread the series describes: the
// main thread, or every message of a subagent's own conversation.
func (s ContextSeries) OnThread(msg Message) bool {
	return !msg.IsSidechain || s.ownThread
}

func (s ContextSeries) stepPrompt(msg Message) (int, bool) {
	if msg.Role != RoleAssistant || !msg.IsVisible() || msg.IsAgentDivider || !s.OnThread(msg) {
		return 0, false
	}
	prompt := msg.Usage.PromptTokens()
	return prompt, prompt > 0
}

// withStep returns s with a step for the message at messageIndex appended. It
// links the previous step to the new one, so the caller must keep the result.
func (s ContextSeries) withStep(messageIndex, prompt int) ContextSeries {
	step := ContextStep{MessageIndex: messageIndex, PromptTokens: prompt, Delta: prompt}
	if n := len(s.Steps); n > 0 {
		previous := &s.Steps[n-1]
		step.Delta = prompt - previous.PromptTokens
		step.Reset = isContextReset(previous.PromptTokens, prompt)
		previous.NextMessageIndex = messageIndex
		if !step.Reset {
			previous.Growth = step.Delta
		}
	}
	s.Steps = append(s.Steps, step)
	index := len(s.Steps) - 1
	if step.Reset {
		s.Resets++
	}
	if s.Peak < 0 || prompt > s.Steps[s.Peak].PromptTokens {
		s.Peak = index
	}
	return s
}

func isContextReset(previous, current int) bool {
	return float64(current) < float64(previous)*contextResetRatio &&
		previous-current >= contextResetMinDrop
}

func conversationCompacted(conversation Conversation) bool {
	for _, session := range conversation.Sessions {
		if session.Performance.HasCompaction() {
			return true
		}
	}
	return false
}

// StepAt returns the index of the last step whose MessageIndex is at or before
// messageIndex. It reports false for indices before the first step.
func (s ContextSeries) StepAt(messageIndex int) (int, bool) {
	next := sort.Search(len(s.Steps), func(i int) bool {
		return s.Steps[i].MessageIndex > messageIndex
	})
	if next == 0 {
		return 0, false
	}
	return next - 1, true
}

// MinGrowth is the smallest growth worth a Growth Tag.
func (s ContextSeries) MinGrowth() int {
	return max(minGrowthFloor, s.Window.Tokens/minGrowthWindowShare)
}

// LargeGrowth is the growth that deserves emphasis.
func (s ContextSeries) LargeGrowth() int {
	return max(largeGrowthFloor, s.Window.Tokens/largeGrowthWindowRate)
}

// IsNotable reports whether step is a Notable Step: a Context Reset, or a step
// whose incoming growth reaches MinGrowth. The first step is the baseline and
// is never notable.
func (s ContextSeries) IsNotable(step int) bool {
	if step <= 0 || step >= len(s.Steps) {
		return false
	}
	return s.Steps[step].Reset || s.Steps[step].Delta >= s.MinGrowth()
}

// Share is tokens as a fraction of the Context Window (the series peak when
// the window is unknown).
func (s ContextSeries) Share(tokens int) float64 {
	if s.Window.Tokens <= 0 {
		return 0
	}
	return float64(tokens) / float64(s.Window.Tokens)
}

// UnderPressure reports whether tokens reach 80% of a known Context Window.
func (s ContextSeries) UnderPressure(tokens int) bool {
	return s.Window.Known && tokens*pressureShareDenom >= s.Window.Tokens*pressureShareNumer
}
