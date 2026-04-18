package codex

import (
	"path/filepath"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

type commandClassification struct {
	actionType conv.NormalizedActionType
	targets    []conv.ActionTarget
}

func classifyCommand(raw string) commandClassification {
	command := strings.TrimSpace(raw)
	if cmd, ok := extractJSONStringField(command, "cmd"); ok {
		command = cmd
	}
	command = unwrapCommand(command)
	words := strings.Fields(command)
	if len(words) == 0 {
		return commandClassification{actionType: conv.NormalizedActionExecute}
	}

	switch {
	case isSearchCommand(words):
		return commandClassification{
			actionType: conv.NormalizedActionSearch,
			targets:    extractSearchTargets(words),
		}
	case isReadCommand(words):
		return commandClassification{
			actionType: conv.NormalizedActionRead,
			targets:    extractReadTargets(words),
		}
	case isTestCommand(words):
		return commandClassification{
			actionType: conv.NormalizedActionTest,
			targets:    []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: command}},
		}
	case isBuildCommand(words):
		return commandClassification{
			actionType: conv.NormalizedActionBuild,
			targets:    []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: command}},
		}
	default:
		return commandClassification{
			actionType: conv.NormalizedActionExecute,
			targets:    []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: command}},
		}
	}
}

func unwrapCommand(command string) string {
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

func isSearchCommand(words []string) bool {
	switch filepath.Base(words[0]) {
	case "rg", "grep", "find", "fd", "ag":
		return true
	default:
		return false
	}
}

func isReadCommand(words []string) bool {
	switch filepath.Base(words[0]) {
	case "cat", "sed", "ls", "head", "tail", "wc", "awk", "jq", "tree", "stat":
		return true
	default:
		return false
	}
}

func isTestCommand(words []string) bool {
	base := filepath.Base(words[0])
	if base == "pytest" || base == "gotestsum" {
		return true
	}
	if isCommandWithVerb(base, words, commandVerbTest) {
		return true
	}
	if isCommandRunVerb(base, words, commandVerbTest) {
		return true
	}
	return isPythonTestRunner(base, words)
}

func isBuildCommand(words []string) bool {
	base := filepath.Base(words[0])
	switch base {
	case "tsc":
		return true
	case "go", "cargo":
		return hasCommandVerb(words, commandVerbBuild)
	case commandRunnerNPM, commandRunnerPNPM, commandRunnerYarn, commandRunnerBun:
		return hasCommandVerb(words, commandVerbBuild) ||
			isCommandRunVerb(base, words, commandVerbBuild)
	default:
		return false
	}
}

func isCommandWithVerb(base string, words []string, verb string) bool {
	switch base {
	case "go", "cargo", commandRunnerNPM, commandRunnerPNPM, commandRunnerYarn, commandRunnerBun:
		return hasCommandVerb(words, verb)
	default:
		return false
	}
}

func isCommandRunVerb(base string, words []string, verb string) bool {
	switch base {
	case commandRunnerNPM, commandRunnerPNPM, commandRunnerYarn, commandRunnerBun:
		return len(words) > 2 && words[1] == commandVerbRun && words[2] == verb
	default:
		return false
	}
}

func isPythonTestRunner(base string, words []string) bool {
	switch base {
	case "python", "python3", "uv":
		return strings.Contains(strings.Join(words, " "), "pytest")
	default:
		return false
	}
}

func hasCommandVerb(words []string, verb string) bool {
	return len(words) > 1 && words[1] == verb
}

func extractSearchTargets(words []string) []conv.ActionTarget {
	switch filepath.Base(words[0]) {
	case "find":
		return appendTargets(
			searchPatternTargets(findPatterns(words)),
			fileTargets(findRoots(words)),
		)
	default:
		return appendTargets(
			searchPatternTargets(searchPatterns(words[1:])),
			fileTargets(fileOperands(words[1:])),
		)
	}
}

func extractReadTargets(words []string) []conv.ActionTarget {
	return fileTargets(fileOperands(words[1:]))
}

func appendTargets(groups ...[]conv.ActionTarget) []conv.ActionTarget {
	seen := make(map[string]struct{})
	targets := make([]conv.ActionTarget, 0)
	for _, group := range groups {
		for _, target := range group {
			if target.Value == "" {
				continue
			}
			key := string(target.Type) + "\x00" + target.Value
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			targets = append(targets, target)
		}
	}
	return targets
}

func fileTargets(paths []string) []conv.ActionTarget {
	targets := make([]conv.ActionTarget, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		targets = append(targets, conv.ActionTarget{
			Type:  conv.ActionTargetFilePath,
			Value: path,
		})
	}
	return targets
}

func searchPatternTargets(patterns []string) []conv.ActionTarget {
	targets := make([]conv.ActionTarget, 0, len(patterns))
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		targets = append(targets, conv.ActionTarget{
			Type:  conv.ActionTargetPattern,
			Value: pattern,
		})
	}
	return targets
}

func searchPatterns(words []string) []string {
	for _, word := range words {
		if strings.HasPrefix(word, "-") {
			continue
		}
		return []string{strings.Trim(word, `"'`)}
	}
	return nil
}

func findPatterns(words []string) []string {
	patterns := make([]string, 0, 1)
	for i := 0; i < len(words)-1; i++ {
		switch words[i] {
		case "-name", "-path", "-wholename":
			patterns = append(patterns, strings.Trim(words[i+1], `"'`))
		}
	}
	return patterns
}

func findRoots(words []string) []string {
	roots := make([]string, 0, 1)
	for _, word := range words[1:] {
		if strings.HasPrefix(word, "-") {
			break
		}
		trimmed := strings.Trim(word, `"'`)
		if trimmed == "" {
			continue
		}
		roots = append(roots, trimmed)
	}
	return roots
}

func fileOperands(words []string) []string {
	paths := make([]string, 0, len(words))
	for _, word := range words {
		if word == "" || strings.HasPrefix(word, "-") {
			continue
		}
		trimmed := strings.Trim(word, `"'`)
		if trimmed == "" || strings.Contains(trimmed, "=") {
			continue
		}
		if looksLikePath(trimmed) {
			paths = append(paths, trimmed)
		}
	}
	return paths
}

func looksLikePath(value string) bool {
	return strings.HasPrefix(value, "/") ||
		strings.HasPrefix(value, ".") ||
		strings.Contains(value, "/") ||
		strings.HasSuffix(value, ".go") ||
		strings.HasSuffix(value, ".md") ||
		strings.HasSuffix(value, ".json") ||
		strings.HasSuffix(value, ".yaml") ||
		strings.HasSuffix(value, ".yml") ||
		strings.HasSuffix(value, ".txt")
}
