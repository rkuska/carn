package elements

import (
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi"

	conv "github.com/rkuska/carn/internal/conversation"
)

// ContextLineAnchor maps the first rendered line of a transcript segment to the
// index of the message that produced it.
type ContextLineAnchor struct {
	Line     int
	MsgIndex int
}

// ContextLineMap is the per-render mapping from rendered transcript line to
// message index, ascending in both. It lets the Turn Ledger and the Context
// Dock find the Context Step under any line.
type ContextLineMap []ContextLineAnchor

// SegmentOffset is the byte offset at which a transcript segment starts in
// rendered content, and the message that produced the segment.
type SegmentOffset struct {
	Offset   int
	MsgIndex int
}

// NewContextLineMap converts segment byte offsets in content into line anchors.
// Segments starting on the same line collapse into the last one, so lookups by
// line are unambiguous.
func NewContextLineMap(content string, offsets []SegmentOffset) ContextLineMap {
	if len(offsets) == 0 {
		return nil
	}
	anchors := make(ContextLineMap, 0, len(offsets))
	line, pos := 0, 0
	for _, entry := range offsets {
		offset := min(entry.Offset, len(content))
		line += strings.Count(content[pos:offset], "\n")
		pos = offset
		if n := len(anchors); n > 0 && anchors[n-1].Line == line {
			anchors[n-1].MsgIndex = entry.MsgIndex
			continue
		}
		anchors = append(anchors, ContextLineAnchor{Line: line, MsgIndex: entry.MsgIndex})
	}
	return anchors
}

// MessageAt returns the index of the message whose content is on line, or -1
// when line lies before the first message (the conversation header).
func (m ContextLineMap) MessageAt(line int) int {
	next := sort.Search(len(m), func(i int) bool { return m[i].Line > line })
	if next == 0 {
		return -1
	}
	return m[next-1].MsgIndex
}

// LineOf returns the first line of the first message at or after msgIndex, so a
// message hidden by a toggle resolves to the next visible one.
func (m ContextLineMap) LineOf(msgIndex int) (int, bool) {
	next := sort.Search(len(m), func(i int) bool { return m[i].MsgIndex >= msgIndex })
	if next == len(m) {
		return 0, false
	}
	return m[next].Line, true
}

// NextLine returns the first of the ascending lines below top.
func NextLine(lines []int, top int) (int, bool) {
	next := sort.Search(len(lines), func(i int) bool { return lines[i] > top })
	if next == len(lines) {
		return 0, false
	}
	return lines[next], true
}

// PrevLine returns the last of the ascending lines above top.
func PrevLine(lines []int, top int) (int, bool) {
	before := sort.Search(len(lines), func(i int) bool { return lines[i] >= top })
	if before == 0 {
		return 0, false
	}
	return lines[before-1], true
}

// CollapseLeadingBlankLines keeps at most one leading blank line of rendered
// markdown. Glamour separates blocks of one document with a single blank line,
// but every render starts with its own document margin, so a segment split off
// at a message boundary would otherwise gain an extra blank line.
func CollapseLeadingBlankLines(rendered string) string {
	blank, pos := 0, 0
	for {
		end := strings.IndexByte(rendered[pos:], '\n')
		if end < 0 || strings.TrimSpace(ansi.Strip(rendered[pos:pos+end])) != "" {
			break
		}
		blank++
		pos += end + 1
	}
	if blank <= 1 {
		return rendered
	}
	return rendered[strings.LastIndex(rendered[:pos-1], "\n")+1:]
}

// ScrollTarget returns the top line that keeps the reader on message msg after
// a re-render: delta lines below the message's first line, kept inside the
// message. A message the toggles hide resolves to the first line of the next
// visible one. total is the number of rendered lines.
func (m ContextLineMap) ScrollTarget(msg, delta, total int) (int, bool) {
	line, ok := m.LineOf(msg)
	if !ok {
		return 0, false
	}
	if m.MessageAt(line) != msg {
		return line, true
	}
	last := total - 1
	if next := sort.Search(len(m), func(i int) bool { return m[i].MsgIndex > msg }); next < len(m) {
		last = m[next].Line - 1
	}
	return min(line+delta, max(last, line)), true
}

// NotableLines returns the lines the Notable Steps of series start on,
// ascending. A step starts on the first line of the first message at or after
// it, so the result works whichever messages the toggles hide.
func NotableLines(series conv.ContextSeries, lines ContextLineMap) []int {
	var notable []int
	for i, step := range series.Steps {
		if !series.IsNotable(i) {
			continue
		}
		line, ok := lines.LineOf(step.MessageIndex)
		if ok && (len(notable) == 0 || notable[len(notable)-1] != line) {
			notable = append(notable, line)
		}
	}
	return notable
}
