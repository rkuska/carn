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

// scrollToMessage puts the top line delta lines below the first line of msg.
func scrollToMessage(t *testing.T, m viewerModel, msg, delta int) viewerModel {
	t.Helper()
	line, ok := m.contextLines.LineOf(msg)
	require.True(t, ok, "message %d is rendered", msg)
	m.viewport.SetYOffset(line + delta)
	require.Equal(t, line+delta, m.viewport.YOffset(), "the viewport can scroll there")
	return m
}

// scrollPosition is the message on the top line and how far below the
// message's first line the top line is.
func scrollPosition(t *testing.T, m viewerModel) (int, int) {
	t.Helper()
	top := m.viewport.YOffset()
	msg := m.messageAtLine(top)
	first, ok := m.contextLines.LineOf(msg)
	require.True(t, ok)
	return msg, top - first
}

func TestRenderContentMapsEveryMessageToAnAscendingLine(t *testing.T) {
	t.Parallel()

	m := newTestViewer(scrollAnchorSession("turn-anchors"), 120, 40)

	previous := -1
	for msg := range 20 {
		line, ok := m.contextLines.LineOf(msg)
		require.True(t, ok, "message %d", msg)
		assert.Equal(t, msg, m.messageAtLine(line))
		assert.Greater(t, line, previous, "message %d starts below message %d", msg, msg-1)
		previous = line
	}
}

func TestRenderPreservesScrollOnToggle(t *testing.T) {
	t.Parallel()

	session := scrollAnchorSession("preserve-scroll")
	// The thinking block sits above the anchored message, so toggling it moves it.
	session.Messages[1].Thinking = "inner thought that only shows when t is on"

	m := scrollToMessage(t, newTestViewer(session, 120, 20), 3, 1)
	beforeMsg, beforeDelta := scrollPosition(t, m)
	require.Equal(t, 3, beforeMsg)

	m.opts.showThinking = true
	m = m.renderContentPreservingScroll()

	afterMsg, afterDelta := scrollPosition(t, m)
	assert.Equal(t, beforeMsg, afterMsg)
	assert.Equal(t, beforeDelta, afterDelta)
}
