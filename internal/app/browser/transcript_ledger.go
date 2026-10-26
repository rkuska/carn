package browser

import (
	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

// applyContextLedger decorates segments with the Turn Ledger of series: Context
// Reset dividers, annotations on Assistant role headers and Growth Tags. The
// plan lives in elements; this maps it back onto segments. Segments of messages
// off the series' thread are left alone.
func applyContextLedger(
	segments []transcriptSegment,
	messages []conv.Message,
	series conv.ContextSeries,
) []transcriptSegment {
	if len(series.Steps) == 0 {
		return segments
	}
	input := make([]el.LedgerSegment, len(segments))
	for i, seg := range segments {
		input[i] = el.LedgerSegment{
			Kind:      ledgerSegmentKind(seg.kind),
			Role:      seg.role,
			MsgIndex:  seg.msgIndex,
			Sidechain: !series.OnThread(messages[seg.msgIndex]),
		}
	}
	items := el.PlanContextLedger(input, series, len(messages))
	out := make([]transcriptSegment, len(items))
	for i, item := range items {
		switch item.Kind {
		case el.LedgerItemSegment:
			out[i] = segments[item.Source]
			out[i].ledger = item
		case el.LedgerItemDivider:
			out[i] = transcriptSegment{kind: segmentContextReset, msgIndex: item.MsgIndex, ledger: item}
		case el.LedgerItemHeader:
			out[i] = transcriptSegment{kind: segmentRoleHeader, role: item.Role, msgIndex: item.MsgIndex, ledger: item}
		}
	}
	return out
}

func ledgerSegmentKind(kind segmentKind) el.LedgerSegmentKind {
	switch kind {
	case segmentRoleHeader:
		return el.LedgerSegmentHeader
	case segmentToolCall:
		return el.LedgerSegmentToolCall
	case segmentToolResult:
		return el.LedgerSegmentToolResult
	case segmentMarkdown, segmentThinking, segmentThinkingUnavailable, segmentContextReset:
		return el.LedgerSegmentOther
	}
	return el.LedgerSegmentOther
}

// renderContextResetSegment renders the Turn Ledger divider of a Context Reset.
func renderContextResetSegment(theme *el.Theme, item el.LedgerItem, width int) string {
	label := "context reset"
	if item.Compacted {
		label = "context compacted"
	}
	return theme.RenderContextResetDivider(label, item.From, item.To, width) + "\n\n"
}

// renderToolCallSegment renders a tool call, with its Growth Tag when the Turn
// Ledger put one on it.
func renderToolCallSegment(theme *el.Theme, seg transcriptSegment, width int) string {
	if !seg.ledger.HasGrowth {
		return renderStyledToolCall(theme, seg.text)
	}
	tag := theme.RenderGrowthTag(seg.ledger.Growth, seg.ledger.GrowthLarge)
	return theme.RenderGrowthTagLine(seg.text, tag, width) + "\n"
}

// renderToolResultSegment renders a tool result, with its Growth Tag when the
// Turn Ledger put one on it.
func renderToolResultSegment(theme *el.Theme, seg transcriptSegment, width int) string {
	block := renderStyledToolResult(theme, seg.result, width)
	if !seg.ledger.HasGrowth {
		return block
	}
	return theme.AppendGrowthTag(block, theme.RenderGrowthTag(seg.ledger.Growth, seg.ledger.GrowthLarge), width)
}
