package elements

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

// contextSeriesOf builds the series of a Claude conversation with one assistant
// message per prompt.
func contextSeriesOf(perf conv.SessionPerformanceMeta, prompts ...int) conv.ContextSeries {
	messages := make([]conv.Message, len(prompts))
	for i, prompt := range prompts {
		messages[i] = conv.Message{
			Role: conv.RoleAssistant, Text: "m", Usage: conv.TokenUsage{CacheReadInputTokens: prompt},
		}
	}
	conversation := conv.Conversation{
		Ref:      conv.Ref{Provider: conv.ProviderClaude, ID: "series"},
		Sessions: []conv.SessionMeta{{Performance: perf}},
	}
	return conv.NewContextSeries(conversation, messages)
}

func planSeries(prompts ...int) conv.ContextSeries {
	return contextSeriesOf(conv.SessionPerformanceMeta{ModelContextWindow: 200_000}, prompts...)
}

func header(role conv.Role, msgIndex int) LedgerSegment {
	return LedgerSegment{Kind: LedgerSegmentHeader, Role: role, MsgIndex: msgIndex}
}

func other(msgIndex int) LedgerSegment {
	return LedgerSegment{Kind: LedgerSegmentOther, MsgIndex: msgIndex}
}

func call(msgIndex int) LedgerSegment {
	return LedgerSegment{Kind: LedgerSegmentToolCall, MsgIndex: msgIndex}
}

func result(msgIndex int) LedgerSegment {
	return LedgerSegment{Kind: LedgerSegmentToolResult, MsgIndex: msgIndex}
}

func kinds(items []LedgerItem) []LedgerItemKind {
	out := make([]LedgerItemKind, len(items))
	for i, item := range items {
		out[i] = item.Kind
	}
	return out
}

func TestPlanContextLedgerWithoutStepsKeepsEverySegmentUndecorated(t *testing.T) {
	t.Parallel()

	segments := []LedgerSegment{header(conv.RoleUser, 0), other(0), header(conv.RoleAssistant, 1), call(1)}

	items := PlanContextLedger(segments, planSeries(), 2)

	require.Len(t, items, 4)
	for i, item := range items {
		assert.Equal(t, LedgerItemSegment, item.Kind)
		assert.Equal(t, i, item.Source)
		assert.False(t, item.HasAnnotation)
		assert.False(t, item.HasGrowth)
	}
}

func TestPlanContextLedgerAnnotatesAssistantHeaderGroups(t *testing.T) {
	t.Parallel()

	// m0 user, m1 assistant (step 0), m2 user, m3 assistant (step 1), m4 and m5 assistant (steps 2, 3)
	series := conv.NewContextSeries(conv.Conversation{
		Ref:      conv.Ref{Provider: conv.ProviderClaude, ID: "plan"},
		Sessions: []conv.SessionMeta{{Performance: conv.SessionPerformanceMeta{ModelContextWindow: 200_000}}},
	}, []conv.Message{
		{Role: conv.RoleUser, Text: "q"},
		{Role: conv.RoleAssistant, Usage: conv.TokenUsage{CacheReadInputTokens: 30_000}},
		{Role: conv.RoleUser, Text: "q"},
		{Role: conv.RoleAssistant, Usage: conv.TokenUsage{CacheReadInputTokens: 50_000}},
		{Role: conv.RoleAssistant, Usage: conv.TokenUsage{CacheReadInputTokens: 60_000}},
		{Role: conv.RoleAssistant, Usage: conv.TokenUsage{CacheReadInputTokens: 170_000}},
	})
	segments := []LedgerSegment{
		header(conv.RoleUser, 0),
		header(conv.RoleAssistant, 1),
		header(conv.RoleUser, 2),
		header(conv.RoleAssistant, 3),
	}

	items := PlanContextLedger(segments, series, 6)

	assert.False(t, items[0].HasAnnotation, "user headers are not annotated")
	assert.Equal(t, LedgerAnnotation{StartTokens: 30_000, EndTokens: 30_000, Share: 0.15}, items[1].Annotation)
	assert.False(t, items[2].HasAnnotation)
	assert.Equal(t, LedgerAnnotation{
		StartTokens: 50_000, EndTokens: 170_000, Share: 0.85, Pressure: true,
	}, items[3].Annotation, "the last header's group runs to the end of the messages")
}

func TestPlanContextLedgerCarriesTheStepOverAHeaderWithoutOne(t *testing.T) {
	t.Parallel()

	series := planSeries(30_000)
	segments := []LedgerSegment{header(conv.RoleUser, 0), header(conv.RoleAssistant, 0)}

	items := PlanContextLedger(segments, series, 1)

	assert.True(t, items[1].HasAnnotation)
	assert.Equal(t, 30_000, items[1].Annotation.EndTokens)
}

func TestPlanContextLedgerSkipsSidechainHeaders(t *testing.T) {
	t.Parallel()

	series := planSeries(30_000)
	side := header(conv.RoleAssistant, 0)
	side.Sidechain = true

	items := PlanContextLedger([]LedgerSegment{side}, series, 1)

	assert.False(t, items[0].HasAnnotation)
}

func TestPlanContextLedgerTagsTheLastToolCallOfAStep(t *testing.T) {
	t.Parallel()

	// step 0 at m1 grows by 40k (large), step 1 at m4.
	series := planSeries(0, 30_000, 0, 0, 70_000)
	segments := []LedgerSegment{
		header(conv.RoleAssistant, 1), call(1), call(2), result(3), header(conv.RoleAssistant, 4), call(4),
	}

	items := PlanContextLedger(segments, series, 5)

	tagged := []int{}
	for i, item := range items {
		if item.HasGrowth {
			tagged = append(tagged, i)
		}
	}
	assert.Equal(t, []int{2}, tagged, "the last call of the step, not the results or the next step")
	assert.Equal(t, 40_000, items[2].Growth)
	assert.True(t, items[2].GrowthLarge)
}

func TestPlanContextLedgerFallsBackToTheLastToolResult(t *testing.T) {
	t.Parallel()

	series := planSeries(0, 30_000, 0, 48_000)
	segments := []LedgerSegment{header(conv.RoleAssistant, 1), result(2), other(2), result(2)}

	items := PlanContextLedger(segments, series, 4)

	assert.False(t, items[1].HasGrowth)
	assert.True(t, items[3].HasGrowth)
	assert.Equal(t, 18_000, items[3].Growth)
	assert.False(t, items[3].GrowthLarge)
}

func TestPlanContextLedgerHasNoTagWithoutToolSegments(t *testing.T) {
	t.Parallel()

	series := planSeries(0, 30_000, 0, 90_000)
	segments := []LedgerSegment{header(conv.RoleAssistant, 1), other(1), other(2)}

	for _, item := range PlanContextLedger(segments, series, 4) {
		assert.False(t, item.HasGrowth)
	}
}

func TestPlanContextLedgerIgnoresSmallGrowthAndSidechainTools(t *testing.T) {
	t.Parallel()

	series := planSeries(0, 30_000, 34_000, 60_000)
	sideCall := call(2)
	sideCall.Sidechain = true
	segments := []LedgerSegment{header(conv.RoleAssistant, 1), call(1), sideCall, call(2)}

	items := PlanContextLedger(segments, series, 4)

	assert.False(t, items[1].HasGrowth, "growth of 4k is below the threshold")
	assert.False(t, items[2].HasGrowth)
	assert.True(t, items[3].HasGrowth, "the main-thread call of step 1 carries the 26k growth")
	assert.Equal(t, 26_000, items[3].Growth)
}

func TestPlanContextLedgerInsertsDividersBeforeResets(t *testing.T) {
	t.Parallel()

	series := planSeries(0, 150_000, 0, 30_000)
	segments := []LedgerSegment{
		header(conv.RoleAssistant, 1), other(1), header(conv.RoleUser, 2), other(2), header(conv.RoleAssistant, 3), other(3),
	}

	items := PlanContextLedger(segments, series, 4)

	assert.Equal(t, []LedgerItemKind{
		LedgerItemSegment, LedgerItemSegment, LedgerItemSegment, LedgerItemSegment,
		LedgerItemDivider, LedgerItemSegment, LedgerItemSegment,
	}, kinds(items))
	divider := items[4]
	assert.Equal(t, 150_000, divider.From)
	assert.Equal(t, 30_000, divider.To)
	assert.Equal(t, 3, divider.MsgIndex)
	assert.False(t, divider.Compacted)
}

func TestPlanContextLedgerRepeatsTheHeaderWhenADividerInterruptsAGroup(t *testing.T) {
	t.Parallel()

	series := planSeries(0, 150_000, 30_000, 31_000)
	segments := []LedgerSegment{header(conv.RoleAssistant, 1), other(1), other(2), other(3)}

	items := PlanContextLedger(segments, series, 4)

	assert.Equal(t, []LedgerItemKind{
		LedgerItemSegment, LedgerItemSegment, LedgerItemDivider, LedgerItemHeader, LedgerItemSegment, LedgerItemSegment,
	}, kinds(items))
	assert.Equal(t, conv.RoleAssistant, items[3].Role)
	assert.Equal(t, 2, items[3].MsgIndex)
	assert.Equal(t, 150_000, items[0].Annotation.StartTokens)
	assert.True(t, items[3].HasAnnotation)
	assert.Equal(t, LedgerAnnotation{StartTokens: 30_000, EndTokens: 31_000, Share: 0.155}, items[3].Annotation)
}

func TestPlanContextLedgerDropsDividersWhenTheResetMessageIsGone(t *testing.T) {
	t.Parallel()

	series := planSeries(0, 150_000, 30_000)
	segments := []LedgerSegment{header(conv.RoleAssistant, 1), other(1)}

	items := PlanContextLedger(segments, series, 3)

	assert.Equal(t, []LedgerItemKind{LedgerItemSegment, LedgerItemSegment}, kinds(items))
}

func TestPlanContextLedgerMarksCompaction(t *testing.T) {
	t.Parallel()

	messages := []conv.Message{
		{Role: conv.RoleAssistant, Usage: conv.TokenUsage{CacheReadInputTokens: 150_000}},
		{Role: conv.RoleAssistant, Usage: conv.TokenUsage{CacheReadInputTokens: 30_000}},
	}
	series := conv.NewContextSeries(conv.Conversation{
		Ref:      conv.Ref{Provider: conv.ProviderClaude, ID: "plan"},
		Sessions: []conv.SessionMeta{{Performance: conv.SessionPerformanceMeta{CompactionCount: 1}}},
	}, messages)

	items := PlanContextLedger([]LedgerSegment{header(conv.RoleAssistant, 0), header(conv.RoleAssistant, 1)}, series, 2)

	assert.True(t, items[1].Compacted)
}
