package browser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestViewerRenderKeyChangesWithLayoutAndToggles(t *testing.T) {
	t.Parallel()

	m := newTestViewer(testSession("viewer-render-key"), 120, 40)
	baseKey := m.renderKey()

	m.opts.showThinking = true
	assert.NotEqual(t, baseKey, m.renderKey())

	m.opts.showThinking = false
	m.planExpanded = true
	assert.NotEqual(t, baseKey, m.renderKey())

	m.planExpanded = false
	m.width = 100
	assert.NotEqual(t, baseKey, m.renderKey())
}

func TestRenderContentCachesByRenderKey(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "viewer-render-cache",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "hello"},
			{Role: conv.RoleAssistant, Text: "hi there", Thinking: "private reasoning"},
		},
	}

	m := newTestViewer(session, 120, 40)
	firstKey := m.renderKey()
	require.Len(t, m.renderCache, 1)
	firstValue, ok := m.renderCache[firstKey]
	require.True(t, ok)

	m = m.renderContent()
	require.Len(t, m.renderCache, 1)
	assert.Equal(t, firstValue.baseContent, m.baseContent)

	m.opts.showThinking = true
	m = m.renderContent()
	require.Len(t, m.renderCache, 2)

	m.opts.showThinking = false
	m = m.renderContent()
	require.Len(t, m.renderCache, 2)
	assert.Equal(t, firstValue.baseContent, m.baseContent)
}
