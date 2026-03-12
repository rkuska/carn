package claude

import (
	"encoding/json"
	"time"
)

type exitPlanResult struct {
	Plan     string `json:"plan"`
	FilePath string `json:"filePath"`
}

func extractExitPlanResult(raw json.RawMessage, ts time.Time) (plan, bool) {
	if len(raw) == 0 {
		return plan{}, false
	}

	var result exitPlanResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return plan{}, false
	}
	if result.Plan == "" || result.FilePath == "" {
		return plan{}, false
	}
	return plan{
		FilePath:  result.FilePath,
		Content:   result.Plan,
		Timestamp: ts,
	}, true
}

func deduplicatePlans(messages []parsedMessage) {
	seen := make(map[string]struct{})
	for i := len(messages) - 1; i >= 0; i-- {
		if len(messages[i].plans) == 0 {
			continue
		}
		kept := messages[i].plans[:0]
		for _, p := range messages[i].plans {
			if _, dup := seen[p.FilePath]; dup {
				continue
			}
			seen[p.FilePath] = struct{}{}
			kept = append(kept, p)
		}
		messages[i].plans = kept
	}
}
