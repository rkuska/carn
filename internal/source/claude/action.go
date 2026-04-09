package claude

import (
	"encoding/json"
	"path/filepath"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

const (
	claudeCommandVerbBuild = "build"
	claudeCommandVerbTest  = "test"
)

func classifyClaudeToolAction(name string, input json.RawMessage) normalizedAction {
	if action, ok := classifyClaudeCoreToolAction(name, input); ok {
		return action
	}
	if action, ok := classifyClaudeMCPAction(name, input); ok {
		return action
	}
	return normalizedAction{Type: conv.NormalizedActionOther}
}

func classifyClaudeCoreToolAction(name string, input json.RawMessage) (normalizedAction, bool) {
	if action, ok := classifyClaudeDirectToolAction(name, input); ok {
		return action, true
	}
	if name == "Bash" {
		command, _ := extractTopLevelJSONStringFieldFast(input, "command")
		classified := classifyClaudeCommand(command)
		return normalizedAction{Type: classified.actionType, Targets: classified.targets}, true
	}
	if isClaudePlanTool(name) {
		return normalizedAction{Type: conv.NormalizedActionPlan}, true
	}
	if isClaudeDelegateTool(name) {
		return normalizedAction{
			Type:    conv.NormalizedActionDelegate,
			Targets: inputTargets(input, conv.ActionTargetPlanPath, "task_id", "taskId", "subject", "description"),
		}, true
	}
	return normalizedAction{}, false
}

func classifyClaudeDirectToolAction(name string, input json.RawMessage) (normalizedAction, bool) {
	switch name {
	case "Read":
		return normalizedAction{
			Type:    conv.NormalizedActionRead,
			Targets: inputTargets(input, conv.ActionTargetFilePath, "file_path"),
		}, true
	case "Grep", "Glob", "ToolSearch":
		return normalizedAction{
			Type:    conv.NormalizedActionSearch,
			Targets: inputTargets(input, conv.ActionTargetPattern, "pattern"),
		}, true
	case "Edit", "NotebookEdit":
		return normalizedAction{
			Type:    conv.NormalizedActionMutate,
			Targets: inputTargets(input, conv.ActionTargetFilePath, "file_path", "notebook_path"),
		}, true
	case "Write":
		return normalizedAction{
			Type:    conv.NormalizedActionRewrite,
			Targets: inputTargets(input, conv.ActionTargetFilePath, "file_path"),
		}, true
	case "WebSearch":
		return normalizedAction{
			Type:    conv.NormalizedActionWeb,
			Targets: inputTargets(input, conv.ActionTargetQuery, "query"),
		}, true
	case "WebFetch":
		return normalizedAction{
			Type:    conv.NormalizedActionWeb,
			Targets: inputTargets(input, conv.ActionTargetURL, "url"),
		}, true
	default:
		return normalizedAction{}, false
	}
}

func classifyClaudeMCPAction(name string, input json.RawMessage) (normalizedAction, bool) {
	if strings.HasPrefix(name, "mcp__context7__") {
		return normalizedAction{
			Type:    conv.NormalizedActionWeb,
			Targets: inputTargets(input, conv.ActionTargetQuery, "query", "libraryName"),
		}, true
	}
	if strings.HasPrefix(name, "mcp__") {
		return normalizedAction{Type: conv.NormalizedActionOther}, true
	}
	return normalizedAction{}, false
}

func isClaudePlanTool(name string) bool {
	switch name {
	case "EnterPlanMode", "ExitPlanMode", "TaskList":
		return true
	default:
		return false
	}
}

func isClaudeDelegateTool(name string) bool {
	switch name {
	case "Agent", "Task", "TaskCreate", "TaskUpdate", "TaskGet", "TaskOutput":
		return true
	default:
		return false
	}
}

type claudeCommandClassification struct {
	actionType normalizedActionType
	targets    []actionTarget
}

func classifyClaudeCommand(raw string) claudeCommandClassification {
	command := unwrapClaudeCommand(raw)
	words := strings.Fields(command)
	if len(words) == 0 {
		return claudeCommandClassification{actionType: conv.NormalizedActionExecute}
	}

	switch {
	case isClaudeSearchCommand(words):
		return claudeCommandClassification{
			actionType: conv.NormalizedActionSearch,
			targets:    extractClaudeSearchTargets(words, command),
		}
	case isClaudeReadCommand(words):
		return claudeCommandClassification{
			actionType: conv.NormalizedActionRead,
			targets:    extractClaudeReadTargets(words),
		}
	case isClaudeTestCommand(words):
		return claudeCommandClassification{
			actionType: conv.NormalizedActionTest,
			targets:    []actionTarget{{Type: conv.ActionTargetCommand, Value: command}},
		}
	case isClaudeBuildCommand(words):
		return claudeCommandClassification{
			actionType: conv.NormalizedActionBuild,
			targets:    []actionTarget{{Type: conv.ActionTargetCommand, Value: command}},
		}
	default:
		return claudeCommandClassification{
			actionType: conv.NormalizedActionExecute,
			targets:    []actionTarget{{Type: conv.ActionTargetCommand, Value: command}},
		}
	}
}

func classifyClaudeToolActionMetadata(name string, input json.RawMessage) normalizedAction {
	if action, ok := classifyClaudeMetadataCoreToolAction(name, input); ok {
		return action
	}
	if action, ok := classifyClaudeMetadataMCPAction(name); ok {
		return action
	}
	return normalizedAction{Type: conv.NormalizedActionOther}
}

func classifyClaudeMetadataCoreToolAction(name string, input json.RawMessage) (normalizedAction, bool) {
	if action, ok := classifyClaudeMetadataDirectToolAction(name, input); ok {
		return action, true
	}
	if isClaudePlanTool(name) {
		return normalizedAction{Type: conv.NormalizedActionPlan}, true
	}
	if isClaudeDelegateTool(name) {
		return normalizedAction{Type: conv.NormalizedActionDelegate}, true
	}
	return normalizedAction{}, false
}

func classifyClaudeMetadataDirectToolAction(name string, input json.RawMessage) (normalizedAction, bool) {
	switch name {
	case "Read":
		return normalizedAction{Type: conv.NormalizedActionRead}, true
	case "Grep", "Glob", "ToolSearch":
		return normalizedAction{Type: conv.NormalizedActionSearch}, true
	case "Edit", "NotebookEdit":
		return normalizedAction{Type: conv.NormalizedActionMutate}, true
	case "Write":
		return normalizedAction{Type: conv.NormalizedActionRewrite}, true
	case "Bash":
		command, _ := extractTopLevelJSONStringFieldFast(input, "command")
		return normalizedAction{Type: classifyClaudeCommandType(command)}, true
	case "WebSearch", "WebFetch":
		return normalizedAction{Type: conv.NormalizedActionWeb}, true
	default:
		return normalizedAction{}, false
	}
}

func classifyClaudeMetadataMCPAction(name string) (normalizedAction, bool) {
	if strings.HasPrefix(name, "mcp__context7__") {
		return normalizedAction{Type: conv.NormalizedActionWeb}, true
	}
	if strings.HasPrefix(name, "mcp__") {
		return normalizedAction{Type: conv.NormalizedActionOther}, true
	}
	return normalizedAction{}, false
}

func classifyClaudeCommandType(raw string) normalizedActionType {
	command := unwrapClaudeCommand(raw)
	words := strings.Fields(command)
	if len(words) == 0 {
		return conv.NormalizedActionExecute
	}

	switch {
	case isClaudeSearchCommand(words):
		return conv.NormalizedActionSearch
	case isClaudeReadCommand(words):
		return conv.NormalizedActionRead
	case isClaudeTestCommand(words):
		return conv.NormalizedActionTest
	case isClaudeBuildCommand(words):
		return conv.NormalizedActionBuild
	default:
		return conv.NormalizedActionExecute
	}
}

func inputTargets(input json.RawMessage, targetType actionTargetType, fields ...string) []actionTarget {
	for _, field := range fields {
		if value, ok := extractTopLevelJSONStringFieldFast(input, field); ok && value != "" {
			return []actionTarget{{Type: targetType, Value: value}}
		}
	}
	return nil
}

func unwrapClaudeCommand(command string) string {
	command = strings.TrimSpace(command)
	for _, marker := range []string{" -lc ", " -ic ", " -c "} {
		_, after, ok := strings.Cut(command, marker)
		if !ok {
			continue
		}
		inner := strings.TrimSpace(after)
		return strings.Trim(inner, `"'`)
	}
	return command
}

func isClaudeSearchCommand(words []string) bool {
	switch filepath.Base(words[0]) {
	case "rg", "grep", "find", "fd", "ag":
		return true
	default:
		return false
	}
}

func isClaudeReadCommand(words []string) bool {
	switch filepath.Base(words[0]) {
	case "cat", "sed", "ls", "head", "tail", "wc", "awk", "jq", "tree", "stat":
		return true
	default:
		return false
	}
}

func isClaudeTestCommand(words []string) bool {
	base := filepath.Base(words[0])
	switch base {
	case "go":
		return hasCommandVerb(words, claudeCommandVerbTest)
	case "pytest", "gotestsum", "cargo":
		return base == "pytest" ||
			base == "gotestsum" ||
			hasCommandVerb(words, claudeCommandVerbTest)
	case "npm", "pnpm", "yarn":
		return hasCommandVerb(words, claudeCommandVerbTest)
	default:
		return false
	}
}

func isClaudeBuildCommand(words []string) bool {
	switch filepath.Base(words[0]) {
	case "go":
		return hasCommandVerb(words, claudeCommandVerbBuild)
	case "cargo":
		return hasCommandVerb(words, claudeCommandVerbBuild)
	case "tsc":
		return true
	case "npm", "pnpm", "yarn":
		return hasCommandVerb(words, claudeCommandVerbBuild)
	default:
		return false
	}
}

func hasCommandVerb(words []string, verb string) bool {
	return len(words) > 1 && words[1] == verb
}

func extractClaudeReadTargets(words []string) []actionTarget {
	if len(words) < 2 {
		return nil
	}
	targets := make([]actionTarget, 0, len(words)-1)
	for _, word := range words[1:] {
		if shouldSkipCommandOperand(word) {
			continue
		}
		targets = append(targets, actionTarget{Type: conv.ActionTargetFilePath, Value: word})
	}
	return targets
}

func extractClaudeSearchTargets(words []string, command string) []actionTarget {
	if len(words) < 2 {
		return []actionTarget{{Type: conv.ActionTargetCommand, Value: command}}
	}
	for i := 1; i < len(words); i++ {
		word := words[i]
		if shouldSkipCommandOperand(word) {
			continue
		}
		return []actionTarget{{Type: conv.ActionTargetPattern, Value: word}}
	}
	return []actionTarget{{Type: conv.ActionTargetCommand, Value: command}}
}

func shouldSkipCommandOperand(word string) bool {
	return word == "" ||
		strings.HasPrefix(word, "-") ||
		strings.Contains(word, "=") ||
		word == "--"
}
