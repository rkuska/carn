package elements

import (
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

const NoDataLabel = "No data"

func FitToWidth(s string, width int) string {
	sw := lipgloss.Width(s)
	if sw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-sw)
}

func ShortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if rest, ok := strings.CutPrefix(path, home); ok {
		return "~" + rest
	}
	return path
}

func RenderWrappedTokens(tokens []string, width int) string {
	if len(tokens) == 0 {
		return ""
	}

	const sep = "  "
	lines := make([]string, 0, len(tokens))
	current := tokens[0]
	for _, token := range tokens[1:] {
		if lipgloss.Width(current+sep+token) <= width {
			current += sep + token
			continue
		}
		lines = append(lines, current)
		current = token
	}
	lines = append(lines, current)
	return strings.Join(lines, "\n")
}

func (t *Theme) RenderSingleChip(label, value string) string {
	return t.StyleMetaLabel.Render(label) + " " + t.StyleMetaValue.Render(value)
}
