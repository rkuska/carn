package codex

import (
	"regexp"
	"strconv"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

var hunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

type patchParser struct {
	hunks   []conv.DiffHunk
	current conv.DiffHunk
	inHunk  bool
}

func parseStructuredPatch(input string) []conv.DiffHunk {
	if !strings.Contains(input, "*** Begin Patch") {
		return nil
	}

	parser := patchParser{current: conv.DiffHunk{OldStart: 1, NewStart: 1}}
	for _, line := range normalizedPatchLines(input) {
		parser.consume(line)
	}
	parser.flush()
	return parser.hunks
}

func normalizedPatchLines(input string) []string {
	return strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
}

func (p *patchParser) consume(line string) {
	if strings.HasPrefix(line, "@@") {
		p.startHunk(line)
		return
	}
	if !p.inHunk || strings.HasPrefix(line, "*** ") {
		return
	}
	if isPatchBodyLine(line) {
		p.current.Lines = append(p.current.Lines, line)
	}
}

func (p *patchParser) startHunk(line string) {
	p.flush()
	p.current = parseHunkHeader(line)
	p.inHunk = true
}

func (p *patchParser) flush() {
	if !p.inHunk || len(p.current.Lines) == 0 {
		return
	}

	ensureHunkLineCounts(&p.current)
	p.hunks = append(p.hunks, p.current)
}

func isPatchBodyLine(line string) bool {
	return strings.HasPrefix(line, "+") ||
		strings.HasPrefix(line, "-") ||
		strings.HasPrefix(line, " ")
}

func ensureHunkLineCounts(hunk *conv.DiffHunk) {
	if hunk.OldLines != 0 && hunk.NewLines != 0 {
		return
	}

	for _, line := range hunk.Lines {
		switch {
		case strings.HasPrefix(line, "-"):
			hunk.OldLines++
		case strings.HasPrefix(line, "+"):
			hunk.NewLines++
		default:
			hunk.OldLines++
			hunk.NewLines++
		}
	}
}

func parseHunkHeader(line string) conv.DiffHunk {
	match := hunkHeaderPattern.FindStringSubmatch(line)
	if len(match) == 0 {
		return conv.DiffHunk{OldStart: 1, NewStart: 1}
	}

	hunk := conv.DiffHunk{
		OldStart: parseInt(match[1], 1),
		NewStart: parseInt(match[3], 1),
	}
	hunk.OldLines = parseInt(match[2], 0)
	hunk.NewLines = parseInt(match[4], 0)
	return hunk
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}
