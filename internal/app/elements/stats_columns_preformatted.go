package elements

import "strings"

func (t *Theme) RenderPreformattedColumns(left, right string, leftWidth, rightWidth int, stacked bool) string {
	if stacked || leftWidth <= 0 || rightWidth <= 0 {
		return strings.TrimSpace(left) + "\n\n" + strings.TrimSpace(right)
	}

	leftLines := splitPreformattedLines(left)
	rightLines := splitPreformattedLines(right)
	lineCount := max(len(leftLines), len(rightLines))
	leftBlank := strings.Repeat(" ", leftWidth)
	rightBlank := strings.Repeat(" ", rightWidth)
	separator := " " + t.StyleRuleHR.Render("│") + " "

	var body strings.Builder
	body.Grow(lineCount * (leftWidth + rightWidth + len(separator) + 1))
	for i := range lineCount {
		if i > 0 {
			body.WriteByte('\n')
		}
		if i < len(leftLines) {
			body.WriteString(leftLines[i])
		} else {
			body.WriteString(leftBlank)
		}
		body.WriteString(separator)
		if i < len(rightLines) {
			body.WriteString(rightLines[i])
		} else {
			body.WriteString(rightBlank)
		}
	}
	return body.String()
}

func splitPreformattedLines(content string) []string {
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}
