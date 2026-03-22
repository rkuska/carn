package stats

import (
	"fmt"
	"strconv"
	"strings"
)

func FormatNumber(n int) string {
	switch {
	case n < 1000:
		return strconv.Itoa(n)
	case n < 100000:
		return formatWithCommas(n)
	case n < 1000000:
		return formatAbbreviated(float64(n)/1000, "k")
	default:
		return formatAbbreviated(float64(n)/1000000, "M")
	}
}

func formatWithCommas(n int) string {
	raw := strconv.Itoa(n)
	if len(raw) <= 3 {
		return raw
	}

	var builder strings.Builder
	prefix := len(raw) % 3
	if prefix == 0 {
		prefix = 3
	}
	builder.WriteString(raw[:prefix])
	for i := prefix; i < len(raw); i += 3 {
		builder.WriteByte(',')
		builder.WriteString(raw[i : i+3])
	}
	return builder.String()
}

func formatAbbreviated(value float64, suffix string) string {
	formatted := fmt.Sprintf("%.1f", value)
	formatted = strings.TrimSuffix(formatted, ".0")
	return formatted + suffix
}
