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
		s.clearAssistantUsageTarget()
		s.messages = appendParsedDividerMessage(s.messages, message.text, timestamp)
	case message.role == conv.RoleSystem:
		s.flushAssistant("", time.Time{}, "")
		s.clearAssistantUsageTarget()
		s.messages = appendParsedSystemMessage(s.messages, message.text, message.visibility, timestamp)
	case message.role == conv.RoleUser:
		s.flushAssistant("", time.Time{}, "")
		s.clearAssistantUsageTarget()
		s.messages = appendParsedUserMessage(s.messages, message.text, timestamp)
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
	if !ok {
		return
	}
	s.attachAssistantUsage(usage)
}

func (s *loadState) attachAssistantUsage(usage conv.TokenUsage) {
	if s.hasPendingAssistantContent() {
		s.pendingUsage = usage
		return
	}
	if s.usageTargetIndex < 0 || s.usageTargetIndex >= len(s.messages) {
		return
	}
	target := &s.messages[s.usageTargetIndex]
	if target.role != conv.RoleAssistant || target.isAgentDivider {
		return
	}
	target.usage = usage
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
	var appended bool
	s.messages, s.usageTargetIndex, appended = appendParsedAssistantMessage(s.messages, assistantContent{
		thinking:          thinking,
		hasHiddenThinking: hasHiddenThinking,
		calls:             s.pendingCalls,
		results:           s.pendingResults,
		plans:             s.pendingPlans,
		usage:             s.pendingUsage,
		text:              text,
		timestamp:         maxTime(s.pendingTimestamp, timestamp),
		performance:       performance,
	})
	if !appended && text == "" && !s.hasPendingAssistantContent() {
		s.clearAssistantUsageTarget()
	}
	s.thinkingParts = s.thinkingParts[:0]
	s.pendingHiddenThinking = false
	s.pendingCalls = s.pendingCalls[:0]
	s.pendingResults = s.pendingResults[:0]
	s.pendingPlans = s.pendingPlans[:0]
	s.pendingUsage = conv.TokenUsage{}
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

func (s *loadState) clearAssistantUsageTarget() {
	s.usageTargetIndex = -1
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
