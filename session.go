package main

import "slices"

// systemInterrupts are messages injected by Claude Code when a session
// is interrupted and later resumed. They are not actual user input.
var systemInterrupts = []string{
	"[Request interrupted by user for tool use]",
	"[Request interrupted by user]",
}

// isSystemInterrupt returns true if text is a Claude Code system message
// rather than actual user input.
func isSystemInterrupt(text string) bool {
	return slices.Contains(systemInterrupts, text)
}
