package browser

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func scrollAnchorSession(id string) conv.Session {
	msgs := make([]conv.Message, 0, 40)
	for i := range 10 {
		msgs = append(msgs,
			conv.Message{Role: conv.RoleUser, Text: fmt.Sprintf("user question %d", i)},
			conv.Message{Role: conv.RoleAssistant, Text: fmt.Sprintf("assistant reply %d", i)},
		)
	}
	return conv.Session{
		Meta: conv.SessionMeta{
			ID:        id,
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "scroll-anchor"},
		},
		Messages: msgs,
	}
}

func TestRenderContentCapturesTurnAnchors(t *testing.T) {
	t.Parallel()

	m := newTestViewer(scrollAnchorSession("turn-anchors"), 120, 40)

	require.NotEmpty(t, m.turnAnchors)
	// 10 user + 10 assistant alternating = 20 role-header boundaries.
	assert.Len(t, m.turnAnchors, 20)

	for i := 1; i < len(m.turnAnchors); i++ {
		assert.Greater(t, m.turnAnchors[i], m.turnAnchors[i-1],
			"anchor[%d]=%d should exceed anchor[%d]=%d", i, m.turnAnchors[i], i-1, m.turnAnchors[i-1])
	}
}

func TestRenderPreservesScrollOnToggle(t *testing.T) {
	t.Parallel()

	session := scrollAnchorSession("preserve-scroll")
	session.Messages[5].Thinking = "inner thought that only shows when t is on"

	m := newTestViewer(session, 120, 20)
	require.GreaterOrEqual(t, len(m.turnAnchors), 4)

	m.viewport.SetYOffset(m.turnAnchors[3] + 1)
	beforeTurnIdx, beforeDelta := findAnchorContext(m.turnAnchors, m.viewport.YOffset())

	m.opts.showThinking = true
	m = m.renderContentPreservingScroll()

	afterTurnIdx, afterDelta := findAnchorContext(m.turnAnchors, m.viewport.YOffset())
	assert.Equal(t, beforeTurnIdx, afterTurnIdx)
	assert.Equal(t, beforeDelta, afterDelta)
}

func TestFindAnchorContextReturnsNegativeWhenBeforeFirstAnchor(t *testing.T) {
	t.Parallel()

	turnIdx, delta := findAnchorContext([]int{10, 20, 30}, 5)
	assert.Equal(t, -1, turnIdx)
	assert.Equal(t, 0, delta)
}

func TestFindAnchorContextReturnsLastAnchorAtOrBeforeOffset(t *testing.T) {
	t.Parallel()

	turnIdx, delta := findAnchorContext([]int{10, 20, 30}, 25)
	assert.Equal(t, 1, turnIdx)
	assert.Equal(t, 5, delta)
}

func TestComputeTurnAnchorsMapsByteOffsetsToLines(t *testing.T) {
	t.Parallel()

	content := "line0\nline1\nline2\nline3\n"
	// Byte offsets at starts of "line1" and "line3".
	offsets := []int{6, 18}
	anchors := computeTurnAnchors(content, offsets)
	assert.Equal(t, []int{1, 3}, anchors)
}
