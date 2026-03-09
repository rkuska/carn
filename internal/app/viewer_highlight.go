package app

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// searchOccurrence identifies a single match by line index and byte
// offset within the ANSI-stripped, lowercased version of that line.
type searchOccurrence struct {
	line      int
	byteStart int
}

// lineOccurrence describes one match on a single line for highlighting.
type lineOccurrence struct {
	byteStart      int
	isCurrentMatch bool
}

// applyViewportContent sets the viewport content from baseContent,
// applying search highlights if an active search exists.
func (m *viewerModel) applyViewportContent() {
	content := highlightSearchMatches(m.baseContent, m.searchQuery, m.matches, m.currentMatch)
	m.viewport.SetContent(content)
}

// findMatchRanges finds all occurrences of queryLower in the stripped text
// and returns lipgloss.Range values with cell-width positions suitable for
// lipgloss.StyleRanges. stripped must be ANSI-free; queryLower must be
// pre-lowered.
func findMatchRanges(stripped, queryLower string, style lipgloss.Style) []lipgloss.Range {
	if queryLower == "" || stripped == "" {
		return nil
	}

	strippedLower := strings.ToLower(stripped)

	// If lowercasing changed byte length, byte offsets from strippedLower
	// cannot index into stripped safely. This is extremely rare in practice
	// (e.g. Turkish İ). Skip highlighting for such lines.
	if len(strippedLower) != len(stripped) {
		return nil
	}

	queryLen := len(queryLower)
	var ranges []lipgloss.Range
	offset := 0

	for {
		idx := strings.Index(strippedLower[offset:], queryLower)
		if idx < 0 {
			break
		}
		byteStart := offset + idx
		byteEnd := byteStart + queryLen

		cellStart := ansi.StringWidth(stripped[:byteStart])
		cellEnd := ansi.StringWidth(stripped[:byteEnd])

		ranges = append(ranges, lipgloss.NewRange(cellStart, cellEnd, style))
		offset = byteEnd
	}

	return ranges
}

// highlightLine applies search highlight styling to all occurrences of
// queryLower in line. The line may contain ANSI escape codes; matching
// is done on stripped text and styling is applied via lipgloss.StyleRanges
// which preserves existing ANSI codes.
func highlightLine(line, queryLower string, style lipgloss.Style) string {
	if queryLower == "" {
		return line
	}

	stripped := ansi.Strip(line)
	ranges := findMatchRanges(stripped, queryLower, style)
	if len(ranges) == 0 {
		return line
	}

	return lipgloss.StyleRanges(line, ranges...)
}

// highlightLineOccurrences applies per-occurrence highlight styling.
// Each lineOccurrence selects styleCurrentMatch or styleSearchMatch
// based on its isCurrentMatch flag. The line may contain ANSI codes;
// matching is done on stripped text via lipgloss.StyleRanges.
func highlightLineOccurrences(line, queryLower string, occs []lineOccurrence) string {
	if len(occs) == 0 || queryLower == "" {
		return line
	}

	stripped := ansi.Strip(line)
	strippedLower := strings.ToLower(stripped)

	if len(strippedLower) != len(stripped) {
		return line
	}

	// Build a set of byte offsets that should get the current-match style.
	currentSet := make(map[int]bool, len(occs))
	for _, occ := range occs {
		if occ.isCurrentMatch {
			currentSet[occ.byteStart] = true
		}
	}

	queryLen := len(queryLower)
	var ranges []lipgloss.Range
	offset := 0

	for {
		idx := strings.Index(strippedLower[offset:], queryLower)
		if idx < 0 {
			break
		}
		byteStart := offset + idx
		byteEnd := byteStart + queryLen

		cellStart := ansi.StringWidth(stripped[:byteStart])
		cellEnd := ansi.StringWidth(stripped[:byteEnd])

		style := styleSearchMatch
		if currentSet[byteStart] {
			style = styleCurrentMatch
		}

		ranges = append(ranges, lipgloss.NewRange(cellStart, cellEnd, style))
		offset = byteEnd
	}

	if len(ranges) == 0 {
		return line
	}

	return lipgloss.StyleRanges(line, ranges...)
}

// highlightSearchMatches highlights individual occurrences in content.
// The occurrence at matches[currentMatch] gets styleCurrentMatch; all
// others get styleSearchMatch. Returns content unchanged when query is
// empty or matches is nil.
func highlightSearchMatches(content, query string, matches []searchOccurrence, currentMatch int) string {
	if query == "" || len(matches) == 0 {
		return content
	}

	queryLower := strings.ToLower(query)
	lines := strings.Split(content, "\n")

	// Group occurrences by line.
	lineOccs := make(map[int][]lineOccurrence, len(matches))
	for i, m := range matches {
		lineOccs[m.line] = append(lineOccs[m.line], lineOccurrence{
			byteStart:      m.byteStart,
			isCurrentMatch: i == currentMatch,
		})
	}

	for lineIdx, occs := range lineOccs {
		if lineIdx < len(lines) {
			lines[lineIdx] = highlightLineOccurrences(lines[lineIdx], queryLower, occs)
		}
	}

	return strings.Join(lines, "\n")
}
