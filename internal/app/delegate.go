package app

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const listItemEllipsis = "..."

type conversationDelegate struct {
	list.DefaultDelegate
}

type matchRangeItem interface {
	MatchRanges() itemMatchRanges
}

func hasMatchRanges(ranges itemMatchRanges) bool {
	return len(ranges.title) > 0 || len(ranges.desc) > 0
}

func newDelegate() conversationDelegate {
	d := conversationDelegate{DefaultDelegate: list.NewDefaultDelegate()}
	d.ShowDescription = true
	d.SetSpacing(1)
	d.SetHeight(3)

	d.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(colorNormalTitle).
		Padding(0, 0, 0, 2)

	d.Styles.NormalDesc = lipgloss.NewStyle().
		Foreground(colorNormalDesc).
		Padding(0, 0, 0, 2)

	d.Styles.FilterMatch = lipgloss.NewStyle().
		Background(colorHighlight).
		Bold(true)

	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(colorAccent).
		Foreground(colorSecondary).
		Padding(0, 0, 0, 1)

	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(colorAccent).
		Foreground(colorSelectedFg).
		Padding(0, 0, 0, 1)

	return d
}

func (d conversationDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	defaultItem, ok := item.(list.DefaultItem)
	if !ok || m.Width() <= 0 {
		return
	}

	s := &d.Styles
	textWidth := m.Width() -
		s.NormalTitle.GetPaddingLeft() -
		s.NormalTitle.GetPaddingRight()
	if textWidth <= 0 {
		return
	}

	title := defaultItem.Title()
	desc := defaultItem.Description()
	matchRanges := splitItemMatches(title, desc, m.MatchesForItem(index))
	if highlighted, ok := item.(matchRangeItem); ok {
		matchRanges = highlighted.MatchRanges()
	}

	title, titleMatches := truncateLineWithMatches(title, matchRanges.title, textWidth)
	desc, descMatches := truncateDescriptionWithMatches(
		desc,
		matchRanges.desc,
		max(d.Height()-1, 0),
		textWidth,
	)

	isSelected := index == m.Index()
	emptyFilter := m.FilterState() == list.Filtering && m.FilterValue() == ""
	isFiltered := m.FilterState() == list.Filtering ||
		m.FilterState() == list.FilterApplied ||
		hasMatchRanges(matchRanges)

	switch {
	case emptyFilter:
		title = s.DimmedTitle.Render(title)
		desc = s.DimmedDesc.Render(desc)

	case isSelected && m.FilterState() != list.Filtering:
		if isFiltered {
			title = renderMatchedText(title, titleMatches, s.SelectedTitle, s.FilterMatch)
			desc = renderMatchedText(desc, descMatches, s.SelectedDesc, s.FilterMatch)
		}
		title = s.SelectedTitle.Render(title)
		desc = s.SelectedDesc.Render(desc)

	default:
		if isFiltered {
			title = renderMatchedText(title, titleMatches, s.NormalTitle, s.FilterMatch)
			desc = renderMatchedText(desc, descMatches, s.NormalDesc, s.FilterMatch)
		}
		title = s.NormalTitle.Render(title)
		desc = s.NormalDesc.Render(desc)
	}

	if d.ShowDescription {
		fmt.Fprintf(w, "%s\n%s", title, desc) //nolint:errcheck
		return
	}
	fmt.Fprintf(w, "%s", title) //nolint:errcheck
}

func renderMatchedText(
	text string,
	matches []int,
	baseStyle lipgloss.Style,
	matchStyle lipgloss.Style,
) string {
	if text == "" || len(matches) == 0 {
		return text
	}

	unmatched := baseStyle.Inline(true)
	matched := unmatched.Inherit(matchStyle)
	return lipgloss.StyleRunes(text, matches, matched, unmatched)
}

func truncateLineWithMatches(line string, matches []int, width int) (string, []int) {
	truncated := ansi.Truncate(line, width, listItemEllipsis)
	visibleRunes := utf8.RuneCountInString(truncated)
	visibleMatches := make([]int, 0, len(matches))
	for _, idx := range matches {
		if idx < visibleRunes {
			visibleMatches = append(visibleMatches, idx)
		}
	}
	return truncated, visibleMatches
}

func truncateDescriptionWithMatches(
	desc string,
	matches []int,
	lineLimit int,
	width int,
) (string, []int) {
	if desc == "" || lineLimit <= 0 || width <= 0 {
		return "", nil
	}

	lines := strings.Split(desc, "\n")
	visibleLines := make([]string, 0, min(len(lines), lineLimit))
	visibleMatches := make([]int, 0, len(matches))
	origOffset := 0
	visibleOffset := 0

	for i, line := range lines {
		if i >= lineLimit {
			break
		}

		lineMatches := matchesForLine(matches, origOffset, line)
		truncatedLine, truncatedMatches := truncateLineWithMatches(line, lineMatches, width)
		visibleLines = append(visibleLines, truncatedLine)
		for _, idx := range truncatedMatches {
			visibleMatches = append(visibleMatches, visibleOffset+idx)
		}

		origOffset += utf8.RuneCountInString(line)
		if i < len(lines)-1 {
			origOffset++
		}

		visibleOffset += utf8.RuneCountInString(truncatedLine)
		if i < min(len(lines), lineLimit)-1 {
			visibleOffset++
		}
	}

	return strings.Join(visibleLines, "\n"), visibleMatches
}

func matchesForLine(matches []int, offset int, line string) []int {
	lineRunes := utf8.RuneCountInString(line)
	lineMatches := make([]int, 0, len(matches))
	for _, idx := range matches {
		local := idx - offset
		if local >= 0 && local < lineRunes {
			lineMatches = append(lineMatches, local)
		}
	}
	return lineMatches
}
