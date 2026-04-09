package claude

import conv "github.com/rkuska/carn/internal/conversation"

type role = conv.Role

const (
	roleUser      = conv.RoleUser
	roleAssistant = conv.RoleAssistant
	roleSystem    = conv.RoleSystem

	contentTypeToolResult = conv.ContentTypeToolResult
)

type project = conv.Project
type tokenUsage = conv.TokenUsage
type diffHunk = conv.DiffHunk
type toolCall = conv.ToolCall
type toolResult = conv.ToolResult
type actionTarget = conv.ActionTarget
type actionTargetType = conv.ActionTargetType
type normalizedAction = conv.NormalizedAction
type normalizedActionType = conv.NormalizedActionType
type messageVisibility = conv.MessageVisibility
type plan = conv.Plan
type message = conv.Message
type messagePerformanceMeta = conv.MessagePerformanceMeta
type sessionPerformanceMeta = conv.SessionPerformanceMeta
type sessionMeta = conv.SessionMeta
type sessionFull = conv.Session
type conversation = conv.Conversation
type conversationRef = conv.Ref
