package elements

import (
	conv "github.com/rkuska/carn/internal/conversation"
)

// LedgerSegmentKind tells PlanContextLedger what a transcript segment is.
type LedgerSegmentKind int

const (
	LedgerSegmentOther LedgerSegmentKind = iota
	LedgerSegmentHeader
	LedgerSegmentToolCall
	LedgerSegmentToolResult
)

// LedgerSegment is what the Turn Ledger needs to know about one transcript
// segment, in transcript order.
type LedgerSegment struct {
	Kind     LedgerSegmentKind
	Role     conv.Role // headers only
	MsgIndex int       // message that produced the segment
	// Sidechain is set for segments of sidechain messages the main-thread series
	// does not describe; the ledger leaves them undecorated.
	Sidechain bool
}

// LedgerItemKind says how a planned item relates to the input segments.
type LedgerItemKind int

const (
	// LedgerItemSegment is an input segment, possibly decorated.
	LedgerItemSegment LedgerItemKind = iota
	// LedgerItemDivider is an inserted Context Reset divider.
	LedgerItemDivider
	// LedgerItemHeader is an inserted role header that repeats the group a
	// divider interrupted.
	LedgerItemHeader
)

// LedgerItem is one entry of the planned transcript: an input segment with its
// decoration, or an item the ledger inserts.
type LedgerItem struct {
	Kind LedgerItemKind
	// Source is the input segment index of a LedgerItemSegment.
	Source   int
	Role     conv.Role
	MsgIndex int

	Annotation    LedgerAnnotation // Assistant header decoration
	HasAnnotation bool

	Growth      int // Growth Tag tokens
	HasGrowth   bool
	GrowthLarge bool

	From, To  int  // LedgerItemDivider: prompt before and after the reset
	Compacted bool // LedgerItemDivider: the conversation recorded a compaction

	sidechain bool
	kind      LedgerSegmentKind // input segment kind; inserted headers are headers too
}

// PlanContextLedger decides where the Turn Ledger decorates a transcript: a
// divider before every Context Reset (with a repeated header when it interrupts
// a group), an annotation on every Assistant header and a Growth Tag on the
// last tool call of every step with notable growth (its last tool result when
// calls are hidden). Every number comes from series, so the plan is independent
// of which segments are hidden. messageCount bounds the last header's group.
func PlanContextLedger(segments []LedgerSegment, series conv.ContextSeries, messageCount int) []LedgerItem {
	if len(series.Steps) == 0 {
		return segmentItems(segments)
	}
	items := insertResetDividers(segments, series)
	annotateLedgerHeaders(items, series, messageCount)
	tagLedgerGrowth(items, series)
	return items
}

func segmentItems(segments []LedgerSegment) []LedgerItem {
	items := make([]LedgerItem, len(segments))
	for i, seg := range segments {
		items[i] = segmentItem(i, seg)
	}
	return items
}

func segmentItem(source int, seg LedgerSegment) LedgerItem {
	return LedgerItem{
		Kind: LedgerItemSegment, Source: source, Role: seg.Role, MsgIndex: seg.MsgIndex,
		sidechain: seg.Sidechain, kind: seg.Kind,
	}
}

// insertResetDividers puts a divider before the first segment of every Context
// Reset step. When the divider interrupts a group, a header is repeated after
// it so the reader sees who continues.
func insertResetDividers(segments []LedgerSegment, series conv.ContextSeries) []LedgerItem {
	if series.Resets == 0 {
		return segmentItems(segments)
	}
	items := make([]LedgerItem, 0, len(segments)+2*series.Resets)
	role := conv.RoleAssistant
	step := nextResetStep(series, 0)
	for i, seg := range segments {
		if seg.Kind == LedgerSegmentHeader {
			role = seg.Role
		}
		inserted := false
		for ; step >= 0 && seg.MsgIndex >= series.Steps[step].MessageIndex; step = nextResetStep(series, step+1) {
			items = append(items, LedgerItem{
				Kind: LedgerItemDivider, Source: i, MsgIndex: seg.MsgIndex,
				From: series.Steps[step-1].PromptTokens, To: series.Steps[step].PromptTokens,
				Compacted: series.Compacted,
			})
			inserted = true
		}
		if inserted && seg.Kind != LedgerSegmentHeader {
			items = append(items, LedgerItem{
				Kind: LedgerItemHeader, Source: i, Role: role, MsgIndex: seg.MsgIndex,
				sidechain: seg.Sidechain, kind: LedgerSegmentHeader,
			})
		}
		items = append(items, segmentItem(i, seg))
	}
	return items
}

func nextResetStep(series conv.ContextSeries, from int) int {
	for i := from; i < len(series.Steps); i++ {
		if series.Steps[i].Reset {
			return i
		}
	}
	return -1
}

// annotateLedgerHeaders decorates every Assistant header from the first and
// last Context Step of its group: the messages from the header up to the next
// header.
func annotateLedgerHeaders(items []LedgerItem, series conv.ContextSeries, messageCount int) {
	previous := -1
	for i := range items {
		if items[i].kind != LedgerSegmentHeader {
			continue
		}
		if previous >= 0 {
			annotateHeader(&items[previous], items[i].MsgIndex, series)
		}
		previous = i
	}
	if previous >= 0 {
		annotateHeader(&items[previous], messageCount, series)
	}
}

func annotateHeader(header *LedgerItem, groupEnd int, series conv.ContextSeries) {
	if header.Role != conv.RoleAssistant || header.sidechain {
		return
	}
	last, ok := series.StepAt(max(groupEnd-1, header.MsgIndex))
	if !ok {
		return
	}
	first, ok := series.StepAt(header.MsgIndex)
	if !ok {
		first = 0
	}
	end := series.Steps[last].PromptTokens
	header.HasAnnotation = true
	header.Annotation = LedgerAnnotation{
		StartTokens: series.Steps[first].PromptTokens,
		EndTokens:   end,
		Share:       series.Share(end),
		Pressure:    series.UnderPressure(end),
	}
}

// growthTarget remembers the items a step's Growth Tag may sit on.
type growthTarget struct{ call, result int }

// tagLedgerGrowth puts each Growth Tag on the last tool call of its step, or on
// the last tool result when no call is shown.
func tagLedgerGrowth(items []LedgerItem, series conv.ContextSeries) {
	large := series.LargeGrowth()
	for step, target := range growthTargets(items, series) {
		at := target.call
		if at < 0 {
			at = target.result
		}
		if at < 0 {
			continue
		}
		growth := series.Steps[step].Growth
		items[at].HasGrowth, items[at].Growth, items[at].GrowthLarge = true, growth, growth >= large
	}
}

// growthTargets finds, per step with notable growth, its last main-thread tool
// call and tool result.
func growthTargets(items []LedgerItem, series conv.ContextSeries) []growthTarget {
	minGrowth := series.MinGrowth()
	targets := make([]growthTarget, len(series.Steps))
	for i := range targets {
		targets[i] = growthTarget{call: -1, result: -1}
	}
	for i, item := range items {
		if item.Kind != LedgerItemSegment || item.sidechain {
			continue
		}
		if item.kind != LedgerSegmentToolCall && item.kind != LedgerSegmentToolResult {
			continue
		}
		step, ok := series.StepAt(item.MsgIndex)
		if !ok || series.Steps[step].Growth < minGrowth {
			continue
		}
		if item.kind == LedgerSegmentToolCall {
			targets[step].call = i
		} else {
			targets[step].result = i
		}
	}
	return targets
}
