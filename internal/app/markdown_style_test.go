package app

import (
	"testing"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubduedMarkdownStyleConfigUsesMinimalHeadingTreatment(t *testing.T) {
	t.Parallel()

	style := subduedMarkdownStyleConfig(true)

	require.NotNil(t, style.H1.Bold)
	assert.True(t, *style.H1.Bold)
	assert.Nil(t, style.H1.BackgroundColor)
	assert.Equal(t, "# ", style.H1.Prefix)
	assert.Nil(t, style.CodeBlock.Chroma)
}

func TestSubduedMarkdownStyleConfigUsesSingleAccentLinks(t *testing.T) {
	t.Parallel()

	style := subduedMarkdownStyleConfig(false)

	require.NotNil(t, style.Link.Color)
	require.NotNil(t, style.Link.Underline)
	require.NotNil(t, style.LinkText.Color)
	assert.True(t, *style.Link.Underline)
	assert.Equal(t, "28", *style.Link.Color)
	assert.Equal(t, *style.Link.Color, *style.LinkText.Color)
}

func TestViewerMarkdownRenderingDropsStandardH1Badge(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "markdown-heading",
			Timestamp: testConv("markdown-heading").Sessions[0].Timestamp,
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "show heading"},
			{Role: conv.RoleAssistant, Text: "# Heading"},
		},
	}

	m := newTestViewer(session, 120, 40)

	assert.Contains(t, m.baseContent, "Heading")
	assert.NotContains(t, m.baseContent, "48;5;63m")
}
