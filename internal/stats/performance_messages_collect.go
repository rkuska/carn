package stats

import (
	"strings"
	"time"
	"unicode/utf8"

	conv "github.com/rkuska/carn/internal/conversation"
)

func (s PerformanceSequenceSession) sessionTimestamp() time.Time {
	return s.Timestamp
}

func CollectPerformanceSequenceSessions(sessions []conv.Session) []PerformanceSequenceSession {
	if len(sessions) == 0 {
		return nil
	}

	collected := make([]PerformanceSequenceSession, 0, len(sessions))
	for _, session := range sessions {
		collected = append(collected, collectPerformanceSequenceSession(session))
	}
	return collected
}

type performanceSequenceCollector struct {
	summary               PerformanceSequenceSession
	lastReadByTarget      map[string]int
	lastSearchIndex       int
	lastPattern           string
	actionIndex           int
	firstMutationIndex    int
	anchorSet             bool
	postMutationFailure   bool
	userTurns             int
	tokensBeforeMutation  int
	actionsBeforeMutation int
}

func collectPerformanceSequenceSession(session conv.Session) PerformanceSequenceSession {
	collector := newPerformanceSequenceCollector(session.Meta.Timestamp)
	for _, message := range session.Messages {
		collector.consumeMessage(message)
	}
	return collector.finish()
}

func newPerformanceSequenceCollector(timestamp time.Time) performanceSequenceCollector {
	return performanceSequenceCollector{
		summary:            PerformanceSequenceSession{Timestamp: timestamp},
		lastReadByTarget:   make(map[string]int),
		lastSearchIndex:    -1000,
		firstMutationIndex: -1,
	}
}

func (c *performanceSequenceCollector) consumeMessage(message conv.Message) {
	c.consumeMessageMeta(message)
	for _, call := range message.ToolCalls {
		c.consumeToolCall(call)
	}
	for _, result := range message.ToolResults {
		c.consumeToolResult(result)
	}
}

func (c *performanceSequenceCollector) consumeMessageMeta(message conv.Message) {
	if message.Role == conv.RoleUser && strings.TrimSpace(message.Text) != "" {
		c.userTurns++
		if c.anchorSet && !c.summary.VerificationPassed {
			c.summary.CorrectionFollowups++
		}
	}
	if message.Role == conv.RoleAssistant {
		c.summary.AssistantTurns++
		c.summary.VisibleReasoningChars += visibleReasoningChars(message.Thinking)
		if message.HasHiddenThinking && strings.TrimSpace(message.Thinking) == "" {
			c.summary.HiddenThinkingTurns++
		}
	}
	if c.firstMutationIndex == -1 {
		c.tokensBeforeMutation += message.Usage.TotalTokens()
	}
}

func visibleReasoningChars(thinking string) int {
	trimmed := strings.TrimSpace(thinking)
	return utf8.RuneCountInString(trimmed)
}

func (c *performanceSequenceCollector) consumeToolCall(call conv.ToolCall) {
	if call.Action.IsZero() {
		return
	}

	c.actionIndex++
	c.summary.ActionCount++
	if c.firstMutationIndex == -1 {
		c.actionsBeforeMutation++
	}
	c.consumeActionPattern(call.Action)

	switch call.Action.Type {
	case conv.NormalizedActionRead:
		rememberTargets(c.lastReadByTarget, call.Action, c.actionIndex)
	case conv.NormalizedActionSearch:
		c.lastSearchIndex = c.actionIndex
	case conv.NormalizedActionMutate, conv.NormalizedActionRewrite:
		c.consumeMutation(call.Action)
	case conv.NormalizedActionExecute,
		conv.NormalizedActionTest,
		conv.NormalizedActionBuild,
		conv.NormalizedActionWeb,
		conv.NormalizedActionPlan,
		conv.NormalizedActionDelegate,
		conv.NormalizedActionOther:
	}
}

func (c *performanceSequenceCollector) consumeActionPattern(action conv.NormalizedAction) {
	pattern := actionPattern(action)
	if pattern == "" {
		return
	}
	if pattern == c.lastPattern {
		c.summary.ReasoningLoopCount++
	}
	c.lastPattern = pattern
}

func (c *performanceSequenceCollector) consumeMutation(action conv.NormalizedAction) {
	c.summary.Mutated = true
	c.summary.MutationCount++
	if action.Type == conv.NormalizedActionRewrite {
		c.summary.RewriteCount++
	}
	if c.firstMutationIndex == -1 {
		c.firstMutationIndex = c.actionIndex
		c.summary.ActionsBeforeFirstMutation = c.actionsBeforeMutation - 1
		c.summary.TokensBeforeFirstMutation = c.tokensBeforeMutation
		c.summary.UserTurnsBeforeFirstMutation = c.userTurns
		c.anchorSet = true
	}
	targeted, blind, distinct := classifyMutationTargets(
		action,
		c.lastReadByTarget,
		c.lastSearchIndex,
		c.actionIndex,
	)
	c.summary.TargetedMutationCount += targeted
	c.summary.BlindMutationCount += blind
	c.summary.DistinctMutationTargets += distinct
}

func (c *performanceSequenceCollector) consumeToolResult(result conv.ToolResult) {
	if len(result.StructuredPatch) > 0 {
		c.summary.PatchHunkCount += len(result.StructuredPatch)
	}
	if result.Action.IsZero() {
		return
	}
	if result.IsError && resultAnchorsCorrection(result, c.summary.Mutated) {
		c.anchorSet = true
	}
	if !c.summary.Mutated {
		return
	}
	if result.IsError {
		c.postMutationFailure = true
	}
	if resultPassedVerification(result) {
		c.summary.VerificationPassed = true
		c.anchorSet = false
	}
}

func resultAnchorsCorrection(result conv.ToolResult, mutated bool) bool {
	return mutated ||
		result.Action.Type == conv.NormalizedActionMutate ||
		result.Action.Type == conv.NormalizedActionRewrite
}

func resultPassedVerification(result conv.ToolResult) bool {
	return !result.IsError &&
		(result.Action.Type == conv.NormalizedActionTest || result.Action.Type == conv.NormalizedActionBuild)
}

func (c performanceSequenceCollector) finish() PerformanceSequenceSession {
	if c.summary.Mutated {
		c.summary.FirstPassResolved = c.summary.CorrectionFollowups == 0 &&
			c.summary.ReasoningLoopCount == 0 &&
			c.summary.MutationCount <= 2 &&
			c.summary.BlindMutationCount == 0 &&
			!c.postMutationFailure
	}
	return c.summary
}

func classifyMutationTargets(
	action conv.NormalizedAction,
	lastReadByTarget map[string]int,
	lastSearchIndex, actionIndex int,
) (int, int, int) {
	seen := make(map[string]struct{}, 2)
	targeted := 0
	blind := 0
	for _, target := range action.Targets {
		if target.Type != conv.ActionTargetFilePath || target.Value == "" {
			continue
		}
		targeted++
		if _, ok := seen[target.Value]; !ok {
			seen[target.Value] = struct{}{}
		}
		if lastReadByTarget[target.Value] == 0 && actionIndex-lastSearchIndex > 3 {
			blind++
		}
	}
	return targeted, blind, len(seen)
}

func rememberTargets(index map[string]int, action conv.NormalizedAction, actionIndex int) {
	for _, target := range action.Targets {
		if target.Type == conv.ActionTargetFilePath && target.Value != "" {
			index[target.Value] = actionIndex
		}
	}
}

func actionPattern(action conv.NormalizedAction) string {
	if action.IsZero() {
		return ""
	}
	if len(action.Targets) == 0 || action.Targets[0].Value == "" {
		return string(action.Type)
	}
	return string(action.Type) + ":" + action.Targets[0].Value
}
