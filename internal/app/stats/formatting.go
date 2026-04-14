package stats

import (
	"strconv"
	"strings"
)

func joinSections(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	totalLen := 0
	count := 0
	for _, part := range parts {
		if part == "" {
			continue
		}
		totalLen += len(part)
		count++
	}
	if count == 0 {
		return ""
	}

	var body strings.Builder
	body.Grow(totalLen + (count-1)*2)
	written := 0
	for _, part := range parts {
		if part == "" {
			continue
		}
		if written > 0 {
			body.WriteString("\n\n")
		}
		body.WriteString(part)
		written++
	}
	return body.String()
}

func formatSignedPercent(value int) string {
	if value >= 0 {
		return "+" + strconv.Itoa(value) + "%"
	}
	return strconv.Itoa(value) + "%"
}

func formatFractionInt(left, right int) string {
	return strconv.Itoa(left) + "/" + strconv.Itoa(right)
}
