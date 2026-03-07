package app

import (
	"slices"
	"strings"
)

// systemInterrupts are messages injected by Claude Code when a session
// is interrupted and later resumed. They are not actual user input.
var systemInterrupts = []string{
	"[Request interrupted by user for tool use]",
	"[Request interrupted by user]",
}

// systemMessagePrefixes are prefixes of system-injected messages
// such as slash command invocations and local command output.
var systemMessagePrefixes = []string{
	"<command-name>",
	"<local-command-stdout>",
	"<local-command-caveat>",
}

// isSystemInterrupt returns true if text is a Claude Code system message
// rather than actual user input.
func isSystemInterrupt(text string) bool {
	if slices.Contains(systemInterrupts, text) {
		return true
	}
	for _, prefix := range systemMessagePrefixes {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}
