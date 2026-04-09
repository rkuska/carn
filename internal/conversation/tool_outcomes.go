package conversation

type ToolOutcomeCounts struct {
	Calls      map[string]int
	Errors     map[string]int
	Rejections map[string]int
}

type ActionOutcomeCounts struct {
	Calls      map[string]int
	Errors     map[string]int
	Rejections map[string]int
}

func DeriveToolOutcomeCounts(messages []Message) ToolOutcomeCounts {
	var counts ToolOutcomeCounts

	for _, message := range messages {
		for _, call := range message.ToolCalls {
			counts.Calls = incrementToolOutcomeCount(counts.Calls, call.Name)
		}
		for _, result := range message.ToolResults {
			if !result.IsError {
				continue
			}
			if result.IsRejected() {
				counts.Rejections = incrementToolOutcomeCount(counts.Rejections, result.ToolName)
				continue
			}
			counts.Errors = incrementToolOutcomeCount(counts.Errors, result.ToolName)
		}
	}

	counts.Calls = nilIfZeroToolOutcomeCounts(counts.Calls)
	counts.Errors = nilIfZeroToolOutcomeCounts(counts.Errors)
	counts.Rejections = nilIfZeroToolOutcomeCounts(counts.Rejections)
	return counts
}

func incrementToolOutcomeCount(counts map[string]int, name string) map[string]int {
	if name == "" {
		return counts
	}
	if counts == nil {
		counts = make(map[string]int, 1)
	}
	counts[name]++
	return counts
}

func nilIfZeroToolOutcomeCounts(counts map[string]int) map[string]int {
	if len(counts) == 0 {
		return nil
	}
	return counts
}

func DeriveActionOutcomeCounts(messages []Message) ActionOutcomeCounts {
	var counts ActionOutcomeCounts

	for _, message := range messages {
		for _, call := range message.ToolCalls {
			if call.Action.IsZero() {
				continue
			}
			counts.Calls = incrementToolOutcomeCount(counts.Calls, string(call.Action.Type))
		}
		for _, result := range message.ToolResults {
			if result.Action.IsZero() || !result.IsError {
				continue
			}
			key := string(result.Action.Type)
			if result.IsRejected() {
				counts.Rejections = incrementToolOutcomeCount(counts.Rejections, key)
				continue
			}
			counts.Errors = incrementToolOutcomeCount(counts.Errors, key)
		}
	}

	counts.Calls = nilIfZeroToolOutcomeCounts(counts.Calls)
	counts.Errors = nilIfZeroToolOutcomeCounts(counts.Errors)
	counts.Rejections = nilIfZeroToolOutcomeCounts(counts.Rejections)
	return counts
}
