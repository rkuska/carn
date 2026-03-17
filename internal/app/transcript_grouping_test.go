package app

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestRenderTranscriptGroupsConsecutiveAssistantMessages(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "group-assistant",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleUser, Text: "Question"},
			{Role: conv.RoleAssistant, Text: "First reply"},
			{Role: conv.RoleAssistant, Text: "Second reply"},
		},
	}

	rendered := renderTranscript(session, transcriptOptions{})

	assert.Equal(t, 1, strings.Count(rendered, "## Assistant"))
	assert.Contains(t, rendered, "First reply")
	assert.Contains(t, rendered, "Second reply")
}

func TestRenderTranscriptStartsNewRoleGroupAfterHiddenBoundary(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "group-boundary",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleAssistant, Text: "Visible reply"},
			{Role: conv.RoleAssistant, Text: "Hidden sidechain", IsSidechain: true},
			{Role: conv.RoleAssistant, Text: "Visible after boundary"},
		},
	}

	rendered := renderTranscript(session, transcriptOptions{hideSidechain: true})

	assert.Equal(t, 2, strings.Count(rendered, "## Assistant"))
}

func TestRenderTranscriptStartsNewRoleGroupAfterAgentDivider(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "group-divider",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{Role: conv.RoleAssistant, Text: "Before divider"},
			{Role: conv.RoleUser, Text: "delegate", IsAgentDivider: true},
			{Role: conv.RoleAssistant, Text: "After divider"},
		},
	}

	rendered := renderTranscript(session, transcriptOptions{})

	assert.Equal(t, 2, strings.Count(rendered, "## Assistant"))
	assert.Contains(t, rendered, "### Subagent")
}

func TestRenderTranscriptGroupsAssistantMessagesAcrossHiddenToolResults(t *testing.T) {
	t.Parallel()

	session := conv.Session{
		Meta: conv.SessionMeta{
			ID:        "group-hidden-results",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "test"},
		},
		Messages: []conv.Message{
			{
				Role: conv.RoleAssistant,
				Text: "Let me check the file.",
				ToolCalls: []conv.ToolCall{
					{Name: "Read", Summary: "/tmp/file.go"},
				},
			},
			{
				Role: conv.RoleUser,
				ToolResults: []conv.ToolResult{
					{ToolName: "Read", ToolSummary: "/tmp/file.go", Content: "package main"},
				},
			},
			{Role: conv.RoleAssistant, Text: "Now I have the answer."},
		},
	}

	rendered := renderTranscript(session, transcriptOptions{})

	assert.Equal(t, 1, strings.Count(rendered, "## Assistant"))
	assert.Contains(t, rendered, "Let me check the file.")
	assert.Contains(t, rendered, "Now I have the answer.")
}
