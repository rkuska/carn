package canonical

import conv "github.com/rkuska/carn/internal/conversation"

type role = conv.Role
type conversationProvider = conv.Provider
type conversationRef = conv.Ref
type project = conv.Project
type tokenUsage = conv.TokenUsage
type diffHunk = conv.DiffHunk
type toolCall = conv.ToolCall
type toolResult = conv.ToolResult
type messageVisibility = conv.MessageVisibility
type plan = conv.Plan
type message = conv.Message
type sessionMeta = conv.SessionMeta
type sessionFull = conv.Session
type conversation = conv.Conversation

var countPlansInMessages = conv.CountPlans
