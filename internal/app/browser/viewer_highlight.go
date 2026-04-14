package browser

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

type searchLineIndex struct {
	lower string
}

// lineOccurrence describes one visible match on a single line.
type lineOccurrence struct {
	isCurrentMatch bool
}

func buildSearchLineIndex(line string, _ int) searchLineIndex {
	return searchLineIndex{
		lower: strings.ToLower(ansi.Strip(line)),
	}
}

func collectSearchOccurrences(lines []searchLineIndex, query string) []searchOccurrence {
	if query == "" {
		return nil
	}

	queryLower := strings.ToLower(query)
	if queryLower == "" {
		return nil
	}

	matches := make([]searchOccurrence, 0)
	for lineIdx, line := range lines {
		offset := 0
		for {
			idx := strings.Index(line.lower[offset:], queryLower)
			if idx < 0 {
				break
			}
			byteStart := offset + idx
			matches = append(matches, searchOccurrence{
				line:      lineIdx,
				byteStart: byteStart,
			})
			offset = byteStart + len(queryLower)
		}
	}

	return matches
}

// highlightLineOccurrences applies per-occurrence highlight styling.
// The line may contain ANSI codes; matching is done on stripped text via
// lipgloss.StyleRanges.
func highlightLineOccurrences(line, queryLower string, occs []lineOccurrence) string {
	if len(occs) == 0 || queryLower == "" {
		return line
	}

	stripped := ansi.Strip(line)
	strippedLower := strings.ToLower(stripped)

	if len(strippedLower) != len(stripped) {
		return line
	}

	queryLen := len(queryLower)
	var ranges []lipgloss.Range
	offset := 0
	occIdx := 0

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
		if occIdx < len(occs) && occs[occIdx].isCurrentMatch {
			style = styleCurrentMatch
		}

		ranges = append(ranges, lipgloss.NewRange(cellStart, cellEnd, style))
		offset = byteEnd
		occIdx++
	}

	if len(ranges) == 0 {
		return line
	}

	return lipgloss.StyleRanges(line, ranges...)
}

func highlightViewportMatches(
	content string,
	query string,
	matches []searchOccurrence,
	currentMatch int,
	lineOffset int,
) string {
	if query == "" || len(matches) == 0 {
		return content
	}

	queryLower := strings.ToLower(query)
	lines := strings.Split(content, "\n")
	lineOccs := make(map[int][]lineOccurrence, len(lines))
	for i, match := range matches {
		if match.line < lineOffset {
			continue
		}
		visibleLine := match.line - lineOffset
		if visibleLine < 0 || visibleLine >= len(lines) {
			continue
		}
		lineOccs[visibleLine] = append(lineOccs[visibleLine], lineOccurrence{
			isCurrentMatch: i == currentMatch,
		})
	}

	for visibleLine, occs := range lineOccs {
		lines[visibleLine] = highlightLineOccurrences(lines[visibleLine], queryLower, occs)
	}

	return strings.Join(lines, "\n")
}
