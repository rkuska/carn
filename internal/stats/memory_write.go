package stats

import (
	"regexp"

	conv "github.com/rkuska/carn/internal/conversation"
)

var memoryWritePathPattern = regexp.MustCompile(`(?i)(/memory/[^/]+\.md|/MEMORY\.md)$`)

// IsMemoryWriteCall reports whether a tool call mutates a Claude-style memory
// file. The call must classify as mutate or rewrite and target a file-path
// ending in `/memory/<name>.md` or `/MEMORY.md`.
func IsMemoryWriteCall(call conv.ToolCall) bool {
	if !isMemoryWriteAction(call.Action.Type) {
		return false
	}
	for _, target := range call.Action.Targets {
		if target.Type != conv.ActionTargetFilePath {
			continue
		}
		if memoryWritePathPattern.MatchString(target.Value) {
			return true
		}
	}
	return false
}

func isMemoryWriteAction(actionType conv.NormalizedActionType) bool {
	return actionType == conv.NormalizedActionMutate || actionType == conv.NormalizedActionRewrite
}
