package browser

import (
	"fmt"
	"strings"

	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

func flattenSegments(segments []transcriptSegment) string {
	return flattenSegmentsOffsets(segments, nil)
}

// flattenSegmentsOffsets flattens segments and, when offsets is non-nil, records
// where each segment starts.
func flattenSegmentsOffsets(segments []transcriptSegment, offsets *[]el.SegmentOffset) string {
	size := 0
	for _, seg := range segments {
		size += len(seg.text) + 50
	}
	var sb strings.Builder
	sb.Grow(size)
	for _, seg := range segments {
		if offsets != nil {
			*offsets = append(*offsets, el.SegmentOffset{Offset: sb.Len(), MsgIndex: seg.msgIndex})
		}
		writeFlattenedSegment(&sb, seg)
	}
	return sb.String()
}

func writeFlattenedSegment(sb *strings.Builder, seg transcriptSegment) {
	switch seg.kind {
	case segmentMarkdown:
		sb.WriteString(seg.text)
	case segmentToolResult:
		sb.WriteString(formatToolResult(seg.result))
		sb.WriteString("\n")
	case segmentRoleHeader:
		appendRoleHeaderSegment(sb, seg.role)
	case segmentThinking:
		sb.WriteString("*Thinking:*\n")
		sb.WriteString(seg.text)
		sb.WriteString("\n\n")
	case segmentThinkingUnavailable:
		sb.WriteString("*Thinking unavailable:*\n")
		sb.WriteString(hiddenThinkingUnavailableText)
		sb.WriteString("\n\n")
	case segmentToolCall:
		sb.WriteString(seg.text)
		sb.WriteString("\n")
	case segmentContextReset:
		// render-only: dividers never reach copied or exported text
	}
}

func appendRoleHeaderSegment(sb *strings.Builder, r conv.Role) {
	switch r {
	case conv.RoleUser:
		sb.WriteString("## You\n\n")
	case conv.RoleAssistant:
		sb.WriteString("## Assistant\n\n")
	case conv.RoleSystem:
		sb.WriteString("## System\n\n")
	}
}

func formatToolCall(tc conv.ToolCall) string {
	if tc.Summary != "" {
		return fmt.Sprintf("[%s: %s]", tc.Name, tc.Summary)
	}
	return fmt.Sprintf("[%s]", tc.Name)
}

func formatToolResult(tr conv.ToolResult) string {
	var sb strings.Builder

	header := "Result"
	if tr.ToolName != "" {
		header = tr.ToolName
	}

	if tr.ToolSummary != "" {
		fmt.Fprintf(&sb, "**%s**: `%s`\n", header, tr.ToolSummary)
	} else {
		fmt.Fprintf(&sb, "**%s**\n", header)
	}

	if len(tr.StructuredPatch) > 0 {
		sb.WriteString("```diff\n")
		for _, hunk := range tr.StructuredPatch {
			fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n",
				hunk.OldStart, hunk.OldLines,
				hunk.NewStart, hunk.NewLines)
			for _, line := range hunk.Lines {
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
		sb.WriteString("```")
		return sb.String()
	}

	if tr.Content != "" {
		sb.WriteString("```\n")
		sb.WriteString(tr.Content)
		if !strings.HasSuffix(tr.Content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("```")
	}

	return sb.String()
}
