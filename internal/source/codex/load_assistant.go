package codex

import (
	"errors"
	"strings"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

func (s *loadState) applyClassifiedMessage(message visibleMessage, timestamp time.Time) {
	switch {
	case message.isAgentDivider:
		s.flushAssistant("", time.Time{}, "")
		s.messages = appendParsedDividerMessage(s.messages, message.text, timestamp)
	case message.role == conv.RoleSystem:
		s.flushAssistant("", time.Time{}, "")
		s.messages = appendParsedSystemMessage(s.messages, message.text, message.visibility, timestamp)
	case message.role == conv.RoleUser:
		s.flushAssistant("", time.Time{}, "")
		s.messages = appendParsedUserMessage(s.messages, message.text, timestamp)
		s.openRequestWindow()
	case message.role == conv.RoleAssistant:
		s.flushAssistant(message.text, timestamp, message.phase)
	}
}

func (s *loadState) appendThinking(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if len(s.thinkingParts) > 0 && s.thinkingParts[len(s.thinkingParts)-1] == text {
		return
	}
	s.thinkingParts = append(s.thinkingParts, text)
}

func (s *loadState) applyPlan(plan conv.Plan, timestamp time.Time) {
	if len(s.messages) > 0 &&
		s.messages[len(s.messages)-1].role == conv.RoleAssistant &&
		len(s.thinkingParts) == 0 &&
		len(s.pendingCalls) == 0 &&
		len(s.pendingResults) == 0 &&
		len(s.pendingPlans) == 0 {
		s.messages[len(s.messages)-1].plans = appendUniquePlans(
			s.messages[len(s.messages)-1].plans,
			[]conv.Plan{plan},
		)
		return
	}
	s.markPendingTimestamp(timestamp)
	s.pendingPlans = appendUniquePlans(s.pendingPlans, []conv.Plan{plan})
}

func (s *loadState) applyTokenCount(payload []byte) {
	if _, ok := extractTopLevelRawJSONFieldByMarker(payload, rateLimitsFieldMarker); ok {
		s.meta.Performance.RateLimitSnapshotCount++
	}
	info, infoOK := extractTopLevelRawJSONFieldByMarker(payload, infoFieldMarker)
	if !infoOK {
		return
	}
	if raw, rawOK := extractTopLevelRawJSONFieldByMarker(info, modelContextWindowFieldMarker); rawOK {
		if window, parsed := readRawJSONInt(raw, 0); parsed {
			s.meta.Performance.ModelContextWindow = max(s.meta.Performance.ModelContextWindow, window)
		}
	}
	usage, ok := scanLastTokenUsageInfo(info)
	if !ok || !s.isNewRequest(info) {
		return
	}
	s.attachRequestUsage(usage)
}

func (s *loadState) flushAssistant(text string, timestamp time.Time, phase string) {
	thinking := joinText(s.thinkingParts)
	hasHiddenThinking := s.pendingHiddenThinking && strings.TrimSpace(thinking) == ""
	performance := s.pendingPerformance
	if performance.Effort == "" {
		performance.Effort = s.currentEffort
	}
	if phase != "" {
		performance.Phase = phase
	}
	var index int
	var appended bool
	s.messages, index, appended = appendParsedAssistantMessage(s.messages, assistantContent{
		thinking:          thinking,
		hasHiddenThinking: hasHiddenThinking,
		calls:             s.pendingCalls,
		results:           s.pendingResults,
		plans:             s.pendingPlans,
		text:              text,
		timestamp:         maxTime(s.pendingTimestamp, timestamp),
		performance:       performance,
	})
	if appended && s.windowFirst < 0 && index >= s.windowStart {
		s.windowFirst = index
	}
	s.thinkingParts = s.thinkingParts[:0]
	s.pendingHiddenThinking = false
	s.pendingCalls = s.pendingCalls[:0]
	s.pendingResults = s.pendingResults[:0]
	s.pendingPlans = s.pendingPlans[:0]
	s.pendingTimestamp = time.Time{}
	s.pendingPerformance = conv.MessagePerformanceMeta{}
}

func (s *loadState) transcript() rolloutTranscript {
	s.flushAssistant("", time.Time{}, "")
	return rolloutTranscript{
		meta:     s.meta,
		link:     s.link,
		messages: s.messages,
	}
}

var errMissingPath = errors.New("missing conversation file path")

func (s *loadState) markPendingTimestamp(timestamp time.Time) {
	if timestamp.After(s.pendingTimestamp) {
		s.pendingTimestamp = timestamp
	}
}

func (s *loadState) hasPendingAssistantContent() bool {
	return len(s.thinkingParts) > 0 ||
		s.pendingHiddenThinking ||
		len(s.pendingCalls) > 0 ||
		len(s.pendingResults) > 0 ||
		len(s.pendingPlans) > 0
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// A request window spans the assistant messages created since the previous
// counted token_count (or the previous user message). The token_count that
// closes the window reports the usage of exactly one model request, so it is
// attached once, to the first assistant message of the window.
func (s *loadState) openRequestWindow() {
	s.windowStart = len(s.messages)
	s.windowFirst = -1
}

// isNewRequest reports whether info describes a model request that has not
// been counted yet. Codex repeats token_count events while the cumulative
// total_token_usage is unchanged.
func (s *loadState) isNewRequest(info []byte) bool {
	total, ok := scanTokenUsageInfoField(info, totalTokenUsageFieldMarker)
	if !ok {
		return true
	}
	tokens := total.TotalTokens()
	if tokens == s.lastTotalTokens {
		return false
	}
	s.lastTotalTokens = tokens
	return true
}

func (s *loadState) attachRequestUsage(usage conv.TokenUsage) {
	if s.hasPendingAssistantContent() {
		s.flushAssistant("", time.Time{}, "")
	}
	if s.windowFirst >= 0 {
		s.messages[s.windowFirst].usage = usage
	}
	s.openRequestWindow()
}
