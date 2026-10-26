package browser

import (
	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

// defaultTranscriptOptions are the toggles a transcript starts with: only the
// Turn Ledger is on.
func defaultTranscriptOptions() transcriptOptions {
	return transcriptOptions{showContextLedger: true}
}

type transcriptOptions struct {
	showThinking      bool
	showTools         bool
	showToolResults   bool
	showSystem        bool
	hideSidechain     bool
	showContextLedger bool
}

type segmentKind int

const (
	segmentMarkdown            segmentKind = iota
	segmentToolResult          segmentKind = iota
	segmentRoleHeader          segmentKind = iota
	segmentThinking            segmentKind = iota
	segmentThinkingUnavailable segmentKind = iota
	segmentToolCall            segmentKind = iota
	segmentContextReset        segmentKind = iota
)

type transcriptSegment struct {
	kind     segmentKind
	text     string
	result   conv.ToolResult
	role     conv.Role
	msgIndex int
	// continuation marks a markdown segment split off from the previous markdown
	// segment at a message boundary so that each segment belongs to one message.
	continuation bool
	// ledger carries the Turn Ledger decoration of the segment.
	ledger el.LedgerItem
}

type transcriptRenderState struct {
	lastRole        conv.Role
	hasVisibleGroup bool
	forceHeader     bool
}

const hiddenThinkingUnavailableText = "Codex recorded reasoning for this reply, " +
	"but no readable thinking summary was stored."

func renderTranscriptSegmented(session conv.Session, opts transcriptOptions) []transcriptSegment {
	var segments []transcriptSegment
	md := markdownBuffer{segments: &segments}
	flush := md.flush
	state := transcriptRenderState{forceHeader: true}

	for i, msg := range session.Messages {
		md.current = i
		start := len(segments)
		appendTranscriptMessage(&segments, &md, flush, &state, msg, opts)
		stampSegmentIndex(segments[start:], i)
	}

	flush()
	return segments
}

func appendTranscriptMessage(
	segments *[]transcriptSegment,
	md *markdownBuffer,
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
	md *markdownBuffer,
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
	md *markdownBuffer,
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
	md *markdownBuffer,
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
	md *markdownBuffer,
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
	md *markdownBuffer,
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
	md *markdownBuffer,
	flush func(),
	state *transcriptRenderState,
	msg conv.Message,
	opts transcriptOptions,
) {
	if opts.showThinking {
		appendThinkingSegments(segments, flush, msg)
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
		(opts.showThinking && msg.HasThinking()) ||
		(opts.showTools && len(msg.ToolCalls) > 0)
}

func appendAssistantSegments(
	segments *[]transcriptSegment,
	md *markdownBuffer,
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
	if opts.showThinking {
		appendThinkingSegments(segments, flush, msg)
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

func appendThinkingSegments(
	segments *[]transcriptSegment,
	flush func(),
	msg conv.Message,
) {
	if msg.Thinking != "" {
		flush()
		*segments = append(*segments, transcriptSegment{kind: segmentThinking, text: msg.Thinking})
		return
	}
	if !msg.HasHiddenThinking {
		return
	}
	flush()
	*segments = append(*segments, transcriptSegment{kind: segmentThinkingUnavailable})
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
