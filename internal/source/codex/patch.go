package codex

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

var hunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

type patchParser struct {
	metadata patchMetadata
	current  conv.DiffHunk
	inHunk   bool
}

type patchMetadata struct {
	Files            []string
	Hunks            []conv.DiffHunk
	changedLineCount int
	contextLineCount int
	addedFileCount   int
}

func parseStructuredPatch(input string) []conv.DiffHunk {
	return parsePatchMetadata(input).Hunks
}

func parsePatchMetadata(input string) patchMetadata {
	if !strings.Contains(input, "*** Begin Patch") {
		return patchMetadata{}
	}

	parser := patchParser{current: conv.DiffHunk{OldStart: 1, NewStart: 1}}
	for _, line := range normalizedPatchLines(input) {
		parser.consume(line)
	}
	parser.flush()
	return parser.metadata
}

func normalizedPatchLines(input string) []string {
	return strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
}

func (p *patchParser) consume(line string) {
	switch {
	case strings.HasPrefix(line, "*** Add File: "):
		p.flush()
		p.appendFile(strings.TrimSpace(strings.TrimPrefix(line, "*** Add File: ")))
		p.metadata.addedFileCount++
		return
	case strings.HasPrefix(line, "*** Update File: "):
		p.flush()
		p.appendFile(strings.TrimSpace(strings.TrimPrefix(line, "*** Update File: ")))
		return
	case strings.HasPrefix(line, "*** Delete File: "):
		p.flush()
		p.appendFile(strings.TrimSpace(strings.TrimPrefix(line, "*** Delete File: ")))
		return
	case strings.HasPrefix(line, "*** Move to: "):
		p.appendFile(strings.TrimSpace(strings.TrimPrefix(line, "*** Move to: ")))
		return
	}

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
		p.inHunk = false
		return
	}

	ensureHunkLineCounts(&p.current)
	for _, line := range p.current.Lines {
		switch {
		case strings.HasPrefix(line, "-"), strings.HasPrefix(line, "+"):
			p.metadata.changedLineCount++
		case strings.HasPrefix(line, " "):
			p.metadata.contextLineCount++
		}
	}
	p.metadata.Hunks = append(p.metadata.Hunks, p.current)
	p.inHunk = false
}

func (p *patchParser) appendFile(path string) {
	if path == "" {
		return
	}
	if slices.Contains(p.metadata.Files, path) {
		return
	}
	p.metadata.Files = append(p.metadata.Files, path)
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
