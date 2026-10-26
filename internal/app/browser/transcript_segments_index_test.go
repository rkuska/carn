package browser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func toolCallTexts(msg conv.Message) []string {
	texts := make([]string, len(msg.ToolCalls))
	for i, call := range msg.ToolCalls {
		texts[i] = formatToolCall(call)
	}
	return texts
}

func TestSegmentsRecordTheMessageThatProducedThem(t *testing.T) {
	t.Parallel()

	optionSets := []struct {
		name string
		opts transcriptOptions
	}{
		{name: "defaults", opts: transcriptOptions{}},
		{name: "everything shown", opts: allTranscriptOptions()},
		{name: "sidechain hidden", opts: transcriptOptions{showTools: true, showToolResults: true, hideSidechain: true}},
		{name: "thinking only", opts: transcriptOptions{showThinking: true}},
		{name: "system only", opts: transcriptOptions{showSystem: true}},
	}

	for _, tt := range optionSets {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session := contextFixtureSession()
			segments := renderTranscriptSegmented(session, tt.opts)

			require.NotEmpty(t, segments)
			previous := 0
			for _, seg := range segments {
				require.GreaterOrEqual(t, seg.msgIndex, previous, "segment indices never move backwards")
				require.Less(t, seg.msgIndex, len(session.Messages))
				previous = seg.msgIndex

				msg := session.Messages[seg.msgIndex]
				switch seg.kind {
				case segmentRoleHeader:
					assert.Equal(t, msg.Role, seg.role)
				case segmentThinking:
					assert.Equal(t, msg.Thinking, seg.text)
				case segmentToolCall:
					assert.Contains(t, toolCallTexts(msg), seg.text)
				case segmentToolResult:
					assert.Contains(t, msg.ToolResults, seg.result)
				case segmentMarkdown:
					assertMarkdownSegmentBelongsTo(t, seg, msg)
				case segmentThinkingUnavailable:
					assert.True(t, msg.HasHiddenThinking)
				case segmentContextReset:
					assert.Fail(t, "reset dividers are added by the ledger, not by segmenting")
				}
			}
		})
	}
}

func assertMarkdownSegmentBelongsTo(t *testing.T, seg transcriptSegment, msg conv.Message) {
	t.Helper()

	if strings.TrimSpace(seg.text) == "" {
		return
	}
	assert.Contains(t, seg.text, msg.Text, "markdown segment must start with the text of its message")
}

func TestRoleHeaderSegmentIndexIsTheMessageThatOpenedTheGroup(t *testing.T) {
	t.Parallel()

	session := contextFixtureSession()
	segments := renderTranscriptSegmented(session, allTranscriptOptions())

	var headers []int
	for _, seg := range segments {
		if seg.kind == segmentRoleHeader {
			headers = append(headers, seg.msgIndex)
		}
	}

	// m00 user, m01 assistant group (m01, m02), m03 tool results (user),
	// m04 assistant group, m05 results, m06 hidden system, m07 sidechain
	// assistant, then the divider breaks the group.
	assert.Equal(t, []int{0, 1, 3, 4, 5, 6, 7, 9, 10, 11, 12, 13, 14, 16, 17}, headers)
}

func TestMarkdownSegmentIndexIgnoresLeadingWhitespaceFromAnEarlierMessage(t *testing.T) {
	t.Parallel()

	// A tool call segment leaves a bare newline in the markdown buffer. The
	// text of the next message in the same group is appended to it, and the
	// resulting segment must be attributed to the message that wrote the text.
	session := conv.Session{Messages: []conv.Message{
		{Role: conv.RoleUser, Text: "ask"},
		{Role: conv.RoleAssistant, Text: "first", ToolCalls: []conv.ToolCall{{Name: "Read", Summary: "a.go"}}},
		{Role: conv.RoleAssistant, Text: "second"},
	}}

	segments := renderTranscriptSegmented(session, transcriptOptions{showTools: true})

	var texts []string
	var indices []int
	for _, seg := range segments {
		if seg.kind == segmentMarkdown {
			texts = append(texts, strings.TrimSpace(seg.text))
			indices = append(indices, seg.msgIndex)
		}
	}
	assert.Equal(t, []string{"ask", "first", "second"}, texts)
	assert.Equal(t, []int{0, 1, 2}, indices)
}
