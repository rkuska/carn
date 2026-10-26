package conversation

import "fmt"

// StepCause names what grew the context before step: a reset or compaction, the
// system prompt for the first step, otherwise the first tool call of the
// previous step (with a count of the others) or the user prompt that followed
// it. Only messages on the series' thread count.
func (s ContextSeries) StepCause(messages []Message, step int) string {
	switch {
	case step <= 0 || step >= len(s.Steps):
		return "system prompt"
	case s.Steps[step].Reset && s.Compacted:
		return "compaction"
	case s.Steps[step].Reset:
		return "reset"
	}
	activity := s.stepActivity(messages, s.Steps[step-1])
	switch {
	case activity.calls > 1:
		return fmt.Sprintf("%s +%d more", activity.first, activity.calls-1)
	case activity.calls == 1:
		return activity.first
	case activity.userPrompt:
		return "user prompt"
	}
	return "model output"
}

// stepActivity is what happened between a step and the next: its tool calls and
// whether a user prompt followed.
type stepActivity struct {
	calls      int
	first      string
	userPrompt bool
}

func (s ContextSeries) stepActivity(messages []Message, step ContextStep) stepActivity {
	var activity stepActivity
	for i := step.MessageIndex; i < step.NextMessageIndex && i < len(messages); i++ {
		msg := messages[i]
		if !s.OnThread(msg) {
			continue
		}
		activity.userPrompt = activity.userPrompt || (msg.Role == RoleUser && !msg.IsAgentDivider && msg.Text != "")
		for _, call := range msg.ToolCalls {
			if activity.calls == 0 {
				activity.first = toolCallLabel(call)
			}
			activity.calls++
		}
	}
	return activity
}

func toolCallLabel(call ToolCall) string {
	if call.Summary == "" {
		return call.Name
	}
	return call.Name + ": " + call.Summary
}
