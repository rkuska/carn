package browser

import "strings"

// markdownBuffer collects markdown text between non-markdown segments and
// remembers which message wrote its first visible text, so the flushed
// segment can be attributed to that message. A bare newline left behind by a
// previous message does not count.
type markdownBuffer struct {
	strings.Builder
	segments *[]transcriptSegment // where flush appends the markdown segment
	current  int                  // index of the message being rendered
	index    int                  // index of the message that wrote the first visible text
	hasText  bool
	// continued is set when a message starts writing visible text while the
	// buffer still holds the visible text of an earlier message: the buffer is
	// split there, and the next flush marks its segment as a continuation.
	continued bool
}

func (b *markdownBuffer) WriteString(s string) {
	visible := strings.TrimSpace(s) != ""
	if visible && b.hasText && b.current != b.index {
		b.flush()
		b.continued = true
	}
	if b.Len() == 0 {
		b.index = b.current
		b.hasText = false
	}
	if !b.hasText && visible {
		b.index = b.current
		b.hasText = true
	}
	b.Builder.WriteString(s)
}

// flush appends the buffered text as a markdown segment and empties the buffer.
func (b *markdownBuffer) flush() {
	if b.Len() == 0 {
		return
	}
	*b.segments = append(*b.segments, transcriptSegment{
		kind:         segmentMarkdown,
		text:         b.String(),
		msgIndex:     b.index,
		continuation: b.continued,
	})
	b.Reset()
}

func (b *markdownBuffer) Reset() {
	b.Builder.Reset()
	b.hasText = false
	b.continued = false
}

// stampSegmentIndex records msgIndex on the non-markdown segments created while
// rendering one message. Markdown segments carry the index of the message that
// wrote their text instead.
func stampSegmentIndex(segments []transcriptSegment, msgIndex int) {
	for i := range segments {
		if segments[i].kind != segmentMarkdown {
			segments[i].msgIndex = msgIndex
		}
	}
}
