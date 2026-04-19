package browser

func (m viewerModel) renderContentPreservingScroll() viewerModel {
	turnIdx, delta := findAnchorContext(m.turnAnchors, m.viewport.YOffset())
	m = m.renderContent()
	if turnIdx < 0 || turnIdx >= len(m.turnAnchors) {
		return m
	}
	target := max(m.turnAnchors[turnIdx]+delta, 0)
	m.viewport.SetYOffset(target)
	return m
}

func findAnchorContext(anchors []int, yOffset int) (int, int) {
	turnIdx := -1
	for i, a := range anchors {
		if a <= yOffset {
			turnIdx = i
			continue
		}
		break
	}
	if turnIdx < 0 {
		return -1, 0
	}
	return turnIdx, yOffset - anchors[turnIdx]
}

func computeTurnAnchors(content string, byteOffsets []int) []int {
	if len(byteOffsets) == 0 {
		return nil
	}
	anchors := make([]int, 0, len(byteOffsets))
	line := 0
	pos := 0
	for _, offset := range byteOffsets {
		if offset > len(content) {
			offset = len(content)
		}
		for pos < offset {
			if content[pos] == '\n' {
				line++
			}
			pos++
		}
		anchors = append(anchors, line)
	}
	return anchors
}
