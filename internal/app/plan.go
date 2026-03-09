package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"
)

type plan struct {
	filePath  string
	content   string
	timestamp time.Time
}

// exitPlanResult is the JSON structure of toolUseResult for accepted ExitPlanMode.
// Rejected plans have a plain string toolUseResult, so Unmarshal into this struct
// fails gracefully — extractExitPlanResult returns false.
type exitPlanResult struct {
	Plan     string `json:"plan"`
	FilePath string `json:"filePath"`
}

// extractExitPlanResult extracts an accepted plan from an ExitPlanMode
// toolUseResult on user messages. The filePath and plan content are read
// directly from the session data. Returns false for rejected plans
// (plain string toolUseResult), malformed data, or missing fields.
// Never returns an error — plan extraction must not prevent session parsing.
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
		filePath:  result.FilePath,
		content:   result.Plan,
		timestamp: ts,
	}, true
}

func writePlan(w *bufio.Writer, p plan) error {
	bw := binWriter{w: w}
	bw.writeString(p.filePath)
	bw.writeString(p.content)
	tsNano := int64(0)
	if !p.timestamp.IsZero() {
		tsNano = p.timestamp.UnixNano()
	}
	bw.writeInt(tsNano)
	if bw.err != nil {
		return fmt.Errorf("writePlan: %w", bw.err)
	}
	return nil
}

func readPlan(r *bufio.Reader) (plan, error) {
	br := binReader{r: r}
	filePath := br.readString()
	content := br.readString()
	tsNano := br.readInt()
	if br.err != nil {
		return plan{}, fmt.Errorf("readPlan: %w", br.err)
	}
	var ts time.Time
	if tsNano != 0 {
		ts = unixTime(tsNano)
	}
	return plan{
		filePath:  filePath,
		content:   content,
		timestamp: ts,
	}, nil
}

// formatPlan renders a plan with filename header for transcript display.
func formatPlan(p plan) string {
	return fmt.Sprintf("Plan: %s\n\n%s", filepath.Base(p.filePath), p.content)
}

// deduplicatePlans keeps only the last plan per filePath across all messages.
// Claude overwrites plans at the same path during iteration, so earlier
// versions are superseded. Walking in reverse ensures the final version
// survives at its original message position.
func deduplicatePlans(messages []parsedMessage) {
	seen := make(map[string]struct{})
	for i := len(messages) - 1; i >= 0; i-- {
		if len(messages[i].plans) == 0 {
			continue
		}
		kept := messages[i].plans[:0]
		for _, p := range messages[i].plans {
			if _, dup := seen[p.filePath]; !dup {
				seen[p.filePath] = struct{}{}
				kept = append(kept, p)
			}
		}
		messages[i].plans = kept
	}
}

// lastPlan returns the final plan from a message list, or false if none exist.
func lastPlan(messages []message) (plan, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		if n := len(messages[i].plans); n > 0 {
			return messages[i].plans[n-1], true
		}
	}
	return plan{}, false
}

// countPlansInMessages counts the total number of plans across all messages.
func countPlansInMessages(messages []message) int {
	count := 0
	for _, msg := range messages {
		count += len(msg.plans)
	}
	return count
}
