package browser

// renderContentPreservingScroll re-renders after a toggle and keeps the reader on
// the same spot: the message on the top line stays there, and when the toggle
// hides it the nearest following message takes its place. Above the first
// message (the conversation header) nothing moves, so the offset is kept.
func (m viewerModel) renderContentPreservingScroll() viewerModel {
	top := m.viewport.YOffset()
	msg := m.messageAtLine(top)
	delta := 0
	if first, ok := m.contextLines.LineOf(msg); ok && msg >= 0 {
		delta = top - first
	}
	m = m.renderContent()
	if msg < 0 {
		return m
	}
	if line, ok := m.contextLines.ScrollTarget(msg, delta, m.viewport.TotalLineCount()); ok {
		m.viewport.SetYOffset(line)
	}
	return m
}
