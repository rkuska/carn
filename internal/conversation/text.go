package conversation

import (
	"path/filepath"
	"strings"
)

func Truncate(s string, maxLen int) string {
	return truncateWithSuffix(s, maxLen, "...", false)
}

func TruncatePreserveNewlines(s string, maxLen int) string {
	return truncateWithSuffix(s, maxLen, "\n...", true)
}

func CompactCWD(cwd string) string {
	return compactPath(cwd, 2)
}

func ProjectName(cwd string) string {
	return compactPath(cwd, 1)
}

func truncateWithSuffix(s string, maxLen int, suffix string, preserveNewlines bool) string {
	s = strings.ReplaceAll(s, "\r", "")
	if preserveNewlines {
		if len(s) <= maxLen {
			return s
		}
		return s[:maxLen] + suffix
	}

	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + suffix
}

func compactPath(cwd string, count int) string {
	if cwd == "" {
		return ""
	}

	normalized := strings.ReplaceAll(cwd, "\\", "/")
	clean := filepath.ToSlash(filepath.Clean(normalized))
	parts := strings.Split(clean, "/")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" && part != "." {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		return clean
	}
	if len(filtered) <= count {
		return strings.Join(filtered, "/")
	}
	return strings.Join(filtered[len(filtered)-count:], "/")
}
