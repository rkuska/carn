package canonical

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestWithEncodedSessionBlobRoundTrip(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		Meta: sessionMeta{
			ID:                    "sess-1",
			Project:               project{DisplayName: "test-project"},
			Slug:                  "test-slug",
			Timestamp:             time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			LastTimestamp:         time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
			CWD:                   "/home/user/project",
			GitBranch:             "main",
			Version:               "1.0.0",
			Model:                 "claude-3",
			FirstMessage:          "hello",
			MessageCount:          5,
			MainMessageCount:      3,
			UserMessageCount:      2,
			AssistantMessageCount: 3,
			FilePath:              "/tmp/sess.jsonl",
			TotalUsage: tokenUsage{
				InputTokens:              1000,
				CacheCreationInputTokens: 100,
				CacheReadInputTokens:     200,
				OutputTokens:             500,
			},
			ToolCounts:      map[string]int{"Read": 3, "Write": 1},
			ToolErrorCounts: map[string]int{"Write": 1},
			IsSubagent:      false,
		},
		Messages: []message{
			{
				Role:       conv.RoleUser,
				Text:       "hello world",
				Visibility: conv.MessageVisibilityVisible,
			},
			{
				Role:     conv.RoleAssistant,
				Text:     "I'll help you",
				Thinking: "let me think about this",
				ToolCalls: []toolCall{
					{Name: "Read", Summary: "Read /tmp/test.go"},
				},
				ToolResults: []toolResult{
					{
						ToolName:    "Read",
						ToolSummary: "Read /tmp/test.go",
						Content:     "package main\n\nfunc main() {}",
						IsError:     false,
						StructuredPatch: []diffHunk{
							{
								OldStart: 1, OldLines: 3,
								NewStart: 1, NewLines: 4,
								Lines: []string{" package main", "+", " func main() {}", "+// end"},
							},
						},
					},
				},
				Plans: []plan{
					{
						FilePath:  "/tmp/plan.md",
						Content:   "step 1: do something",
						Timestamp: time.Date(2025, 1, 15, 10, 45, 0, 0, time.UTC),
					},
				},
				IsSidechain:    true,
				IsAgentDivider: false,
				Usage: tokenUsage{
					InputTokens:              600,
					CacheCreationInputTokens: 80,
					CacheReadInputTokens:     120,
					OutputTokens:             240,
				},
			},
		},
	}

	var blob []byte
	err := withEncodedSessionBlob(session, func(data []byte) error {
		blob = make([]byte, len(data))
		copy(blob, data)
		return nil
	})
	require.NoError(t, err)

	decoded, err := decodeSessionBlob(blob)
	require.NoError(t, err)

	assert.Equal(t, session.Meta, decoded.Meta)
	require.Len(t, decoded.Messages, 2)

	// First message: simple user message — codec deserializes empty slices as non-nil
	assert.Equal(t, session.Messages[0].Role, decoded.Messages[0].Role)
	assert.Equal(t, session.Messages[0].Text, decoded.Messages[0].Text)
	assert.Equal(t, session.Messages[0].Visibility, decoded.Messages[0].Visibility)

	// Second message: complex assistant message with tools and plans
	assert.Equal(t, session.Messages[1].Role, decoded.Messages[1].Role)
	assert.Equal(t, session.Messages[1].Text, decoded.Messages[1].Text)
	assert.Equal(t, session.Messages[1].Thinking, decoded.Messages[1].Thinking)
	assert.Equal(t, session.Messages[1].ToolCalls, decoded.Messages[1].ToolCalls)
	assert.Equal(t, session.Messages[1].ToolResults, decoded.Messages[1].ToolResults)
	assert.Equal(t, session.Messages[1].Plans, decoded.Messages[1].Plans)
	assert.Equal(t, session.Messages[1].IsSidechain, decoded.Messages[1].IsSidechain)
	assert.Equal(t, session.Messages[1].Usage, decoded.Messages[1].Usage)
}

func TestWithEncodedSessionBlobMinimal(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		Meta: sessionMeta{
			ID:        "min-sess",
			Timestamp: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		Messages: []message{
			{Role: conv.RoleUser, Text: "hi"},
		},
	}

	var blob []byte
	err := withEncodedSessionBlob(session, func(data []byte) error {
		blob = make([]byte, len(data))
		copy(blob, data)
		return nil
	})
	require.NoError(t, err)

	decoded, err := decodeSessionBlob(blob)
	require.NoError(t, err)
	assert.Equal(t, session.Meta.ID, decoded.Meta.ID)
	assert.Equal(t, session.Meta.Timestamp, decoded.Meta.Timestamp)
	require.Len(t, decoded.Messages, 1)
	assert.Equal(t, conv.RoleUser, decoded.Messages[0].Role)
	assert.Equal(t, "hi", decoded.Messages[0].Text)
}

func TestWithEncodedSessionBlobEmptyFields(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		Meta: sessionMeta{
			ID:        "empty-sess",
			Timestamp: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		Messages: []message{
			{
				Role: conv.RoleUser,
				Text: "",
			},
		},
	}

	var blob []byte
	err := withEncodedSessionBlob(session, func(data []byte) error {
		blob = make([]byte, len(data))
		copy(blob, data)
		return nil
	})
	require.NoError(t, err)

	decoded, err := decodeSessionBlob(blob)
	require.NoError(t, err)
	assert.Equal(t, session.Meta.ID, decoded.Meta.ID)
	require.Len(t, decoded.Messages, 1)
	assert.Equal(t, conv.RoleUser, decoded.Messages[0].Role)
	assert.Equal(t, "", decoded.Messages[0].Text)
}

func TestWithEncodedSessionBlobMessageUsageRoundTrip(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		Meta: sessionMeta{
			ID:        "usage-sess",
			Timestamp: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		Messages: []message{
			{
				Role:  conv.RoleAssistant,
				Text:  "first reply",
				Usage: tokenUsage{InputTokens: 120, OutputTokens: 30},
			},
			{
				Role:  conv.RoleAssistant,
				Text:  "second reply",
				Usage: tokenUsage{InputTokens: 240, CacheReadInputTokens: 18, OutputTokens: 55},
			},
		},
	}

	var blob []byte
	err := withEncodedSessionBlob(session, func(data []byte) error {
		blob = make([]byte, len(data))
		copy(blob, data)
		return nil
	})
	require.NoError(t, err)

	decoded, err := decodeSessionBlob(blob)
	require.NoError(t, err)
	require.Len(t, decoded.Messages, 2)
	assert.Equal(t, session.Messages[0].Usage, decoded.Messages[0].Usage)
	assert.Equal(t, session.Messages[1].Usage, decoded.Messages[1].Usage)
}

func TestMarshalUnmarshalToolCountsRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		counts map[string]int
	}{
		{name: "nil", counts: nil},
		{name: "single", counts: map[string]int{"Read": 5}},
		{name: "multiple", counts: map[string]int{"Read": 5, "Write": 3, "Bash": 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw := marshalToolCountsCached(tt.counts)
			got, err := unmarshalToolCounts(raw)
			require.NoError(t, err)
			if len(tt.counts) == 0 {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.counts, got)
			}
		})
	}
}

func TestMarshalToolCountsCachedStableOrder(t *testing.T) {
	t.Parallel()

	raw := marshalToolCountsCached(map[string]int{
		"Write": 3,
		"Read":  5,
		"Bash":  10,
	})

	assert.Equal(t, `{"Bash":10,"Read":5,"Write":3}`, raw)
}
