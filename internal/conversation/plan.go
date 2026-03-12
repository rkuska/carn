package conversation

import (
	"fmt"
	"path/filepath"
	"time"
)

type Plan struct {
	FilePath  string
	Content   string
	Timestamp time.Time
}

func FormatPlan(p Plan) string {
	return fmt.Sprintf("Plan: %s\n\n%s", filepath.Base(p.FilePath), p.Content)
}

func LastPlan(messages []Message) (Plan, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		if n := len(messages[i].Plans); n > 0 {
			return messages[i].Plans[n-1], true
		}
	}
	return Plan{}, false
}

func AllPlans(messages []Message) []Plan {
	var plans []Plan
	for _, msg := range messages {
		plans = append(plans, msg.Plans...)
	}
	return plans
}

func CountPlans(messages []Message) int {
	count := 0
	for _, msg := range messages {
		count += len(msg.Plans)
	}
	return count
}
