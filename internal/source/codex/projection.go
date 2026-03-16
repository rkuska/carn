package codex

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

func joinText(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.TrimSpace(strings.Join(filtered, "\n\n"))
}

func joinUniqueText(existing, added string) string {
	trimmedAdded := strings.TrimSpace(added)
	if trimmedAdded == "" {
		return existing
	}
	trimmedExisting := strings.TrimSpace(existing)
	if trimmedExisting == "" {
		return added
	}
	if trimmedExisting == trimmedAdded {
		return existing
	}
	return existing + "\n\n" + added
}

func appendUniquePlans(existing []conv.Plan, added []conv.Plan) []conv.Plan {
	if len(added) == 0 {
		return existing
	}
	if len(existing) == 0 {
		return append(existing, added...)
	}

	seen := make(map[string]struct{}, len(existing))
	for _, plan := range existing {
		seen[plan.FilePath] = struct{}{}
	}
	for _, plan := range added {
		if _, ok := seen[plan.FilePath]; ok {
			continue
		}
		seen[plan.FilePath] = struct{}{}
		existing = append(existing, plan)
	}
	return existing
}

func extractCompletedPlan(raw json.RawMessage, ts time.Time) (conv.Plan, bool) {
	if len(raw) == 0 {
		return conv.Plan{}, false
	}

	var item completedItemPayload
	if err := json.Unmarshal(raw, &item); err != nil {
		return conv.Plan{}, false
	}
	if item.Type != eventItemTypePlan || strings.TrimSpace(item.Text) == "" {
		return conv.Plan{}, false
	}

	filePath := strings.TrimSpace(item.ID)
	if filePath == "" {
		filePath = "plan"
	}
	if !strings.Contains(filepath.Base(filePath), ".") {
		filePath += ".md"
	}

	return conv.Plan{
		FilePath:  filePath,
		Content:   strings.TrimSpace(item.Text),
		Timestamp: ts,
	}, true
}
