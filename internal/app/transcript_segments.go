package app

import (
	"fmt"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

type transcriptOptions struct {
	showThinking    bool
	showTools       bool
	showToolResults bool
	showSystem      bool
	hideSidechain   bool
}

type segmentKind int

const (
	segmentMarkdown   segmentKind = iota
	segmentToolResult segmentKind = iota
	segmentRoleHeader segmentKind = iota
	segmentThinking   segmentKind = iota
	segmentToolCall   segmentKind = iota
)

type transcriptSegment struct {
	kind   segmentKind
	text   string
	result conv.ToolResult
	role   conv.Role
}

type transcriptRenderState struct {
	lastRole        conv.Role
	hasVisibleGroup bool
	forceHeader     bool
}

func renderTranscriptSegmented(session conv.Session, opts transcriptOptions) []transcriptSegment {
	var segments []transcriptSegment
	var md strings.Builder
	state := transcriptRenderState{forceHeader: true}

	flush := func() {
		if md.Len() == 0 {
			return
		}
		segments = append(segments, transcriptSegment{kind: segmentMarkdown, text: md.String()})
		md.Reset()
	}

	for _, msg := range session.Messages {
		appendTranscriptMessage(&segments, &md, flush, &state, msg, opts)
	}

	flush()
	return segments
}

func appendTranscriptMessage(
	segments *[]transcriptSegment,
	md *strings.Builder,
	flush func(),
	state *transcriptRenderState,
	msg conv.Message,
	opts transcriptOptions,
) {
	if shouldSkipSidechain(msg, opts) {
		state.breakGroup()
		return
	}
	if appendAgentDividerSegments(md, flush, state, msg) {
		return
	}
	if appendSuppressedMessageSegments(segments, md, flush, state, msg, opts) {
		return
	}
	appendVisibleMessageSegments(segments, md, flush, state, msg, opts)
}

func shouldSkipSidechain(msg conv.Message, opts transcriptOptions) bool {
	return opts.hideSidechain && msg.IsSidechain
}

func appendAgentDividerSegments(
	md *strings.Builder,
	flush func(),
	state *transcriptRenderState,
	msg conv.Message,
) bool {
	if !msg.IsAgentDivider {
		return false
	}
	flush()
	md.WriteString("---\n### Subagent\n")
	md.WriteString(msg.Text)
	md.WriteString("\n---\n\n")
	state.breakGroup()
	return true
}

func appendSuppressedMessageSegments(
	segments *[]transcriptSegment,
	md *strings.Builder,
	flush func(),
	state *transcriptRenderState,
	msg conv.Message,
	opts transcriptOptions,
) bool {
	if msg.IsVisible() || opts.showSystem {
		return false
	}
	appendHiddenMessageSegments(segments, md, flush, state, msg, opts)
	return true
}

func appendVisibleMessageSegments(
	segments *[]transcriptSegment,
	md *strings.Builder,
	flush func(),
	state *transcriptRenderState,
	msg conv.Message,
	opts transcriptOptions,
) {
	switch msg.Role {
	case conv.RoleUser:
		appendUserSegments(segments, md, flush, state, msg, opts)
	case conv.RoleAssistant:
		appendAssistantSegments(segments, md, flush, state, msg, opts)
	case conv.RoleSystem:
		appendSystemSegments(segments, md, flush, state, msg, opts)
	}
}

func userHasContent(msg conv.Message, userText string, opts transcriptOptions) bool {
	return userText != "" || (opts.showToolResults && len(msg.ToolResults) > 0)
}

func appendUserSegments(
	segments *[]transcriptSegment,
	md *strings.Builder,
	flush func(),
	state *transcriptRenderState,
	msg conv.Message,
	opts transcriptOptions,
) {
	if !userHasContent(msg, msg.Text, opts) {
		// Hidden message kinds should not create a visible role boundary.
		return
	}

	appendRoleHeader(segments, flush, state, conv.RoleUser)
	if msg.Text != "" {
		md.WriteString(msg.Text)
		md.WriteString("\n\n")
	}
	if opts.showToolResults && len(msg.ToolResults) > 0 {
		flush()
		appendToolResultSegments(segments, msg.ToolResults)
		md.WriteString("\n")
	}
}

func appendSystemSegments(
	segments *[]transcriptSegment,
	md *strings.Builder,
	flush func(),
	state *transcriptRenderState,
	msg conv.Message,
	opts transcriptOptions,
) {
	if msg.Text == "" && (!opts.showToolResults || len(msg.ToolResults) == 0) {
		return
	}

	appendRoleHeader(segments, flush, state, conv.RoleSystem)
	if msg.Text != "" {
		md.WriteString(msg.Text)
		md.WriteString("\n\n")
	}
	if opts.showToolResults && len(msg.ToolResults) > 0 {
		flush()
		appendToolResultSegments(segments, msg.ToolResults)
		md.WriteString("\n")
	}
}

func appendHiddenMessageSegments(
	segments *[]transcriptSegment,
	md *strings.Builder,
	flush func(),
	state *transcriptRenderState,
	msg conv.Message,
	opts transcriptOptions,
) {
	if opts.showThinking && msg.Thinking != "" {
		flush()
		*segments = append(*segments, transcriptSegment{kind: segmentThinking, text: msg.Thinking})
	}
	if opts.showTools && len(msg.ToolCalls) > 0 {
		flush()
		appendToolCallSegments(segments, msg.ToolCalls)
		md.WriteString("\n")
	}
	if opts.showToolResults && len(msg.ToolResults) > 0 {
		flush()
		appendToolResultSegments(segments, msg.ToolResults)
		md.WriteString("\n")
	}
	state.breakGroup()
}

func appendToolResultSegments(segments *[]transcriptSegment, results []conv.ToolResult) {
	for _, result := range results {
		*segments = append(*segments, transcriptSegment{kind: segmentToolResult, result: result})
	}
}

func assistantHasContent(msg conv.Message, opts transcriptOptions) bool {
	return msg.Text != "" ||
		(opts.showThinking && msg.Thinking != "") ||
		(opts.showTools && len(msg.ToolCalls) > 0)
}

func appendAssistantSegments(
	segments *[]transcriptSegment,
	md *strings.Builder,
	flush func(),
	state *transcriptRenderState,
	msg conv.Message,
	opts transcriptOptions,
) {
	if !assistantHasContent(msg, opts) {
		// Hidden message kinds should not create a visible role boundary.
		return
	}

	appendRoleHeader(segments, flush, state, conv.RoleAssistant)
	if opts.showThinking && msg.Thinking != "" {
		flush()
		*segments = append(*segments, transcriptSegment{kind: segmentThinking, text: msg.Thinking})
	}
	if msg.Text != "" {
		md.WriteString(msg.Text)
		md.WriteString("\n\n")
	}
	if opts.showTools && len(msg.ToolCalls) > 0 {
		flush()
		appendToolCallSegments(segments, msg.ToolCalls)
		md.WriteString("\n")
	}
}

func appendRoleHeader(
	segments *[]transcriptSegment,
	flush func(),
	state *transcriptRenderState,
	role conv.Role,
) {
	if !state.shouldStartGroup(role) {
		return
	}
	flush()
	*segments = append(*segments, transcriptSegment{kind: segmentRoleHeader, role: role})
	state.startGroup(role)
}

func (s *transcriptRenderState) shouldStartGroup(role conv.Role) bool {
	return s.forceHeader || !s.hasVisibleGroup || s.lastRole != role
}

func (s *transcriptRenderState) startGroup(role conv.Role) {
	s.lastRole = role
	s.hasVisibleGroup = true
	s.forceHeader = false
}

func (s *transcriptRenderState) breakGroup() {
	s.forceHeader = true
}

func appendToolCallSegments(segments *[]transcriptSegment, toolCalls []conv.ToolCall) {
	for _, call := range toolCalls {
		*segments = append(*segments, transcriptSegment{kind: segmentToolCall, text: formatToolCall(call)})
	}
}

func flattenSegments(segments []transcriptSegment) string {
	var sb strings.Builder
	for _, seg := range segments {
		switch seg.kind {
		case segmentMarkdown:
			sb.WriteString(seg.text)
		case segmentToolResult:
			sb.WriteString(formatToolResult(seg.result))
			sb.WriteString("\n")
		case segmentRoleHeader:
			appendRoleHeaderSegment(&sb, seg.role)
		case segmentThinking:
			sb.WriteString("*Thinking:*\n")
			sb.WriteString(seg.text)
			sb.WriteString("\n\n")
		case segmentToolCall:
			sb.WriteString(seg.text)
			sb.WriteString("\n")
		}
	}
	return sb.String()
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

func renderTranscript(session conv.Session, opts transcriptOptions) string {
	return renderVisibleConversation(session, opts, false)
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
