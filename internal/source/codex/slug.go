package codex

import "strings"

const codexSlugPrefixLen = 12

func slugFromThreadID(threadID string) string {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return ""
	}
	if len(threadID) <= codexSlugPrefixLen {
		return threadID
	}
	return threadID[:codexSlugPrefixLen]
}
