package claude

import conv "github.com/rkuska/carn/internal/conversation"

type role = conv.Role

const (
	roleUser      = conv.RoleUser
	roleAssistant = conv.RoleAssistant

	contentTypeToolResult = conv.ContentTypeToolResult
)

type project = conv.Project
type tokenUsage = conv.TokenUsage
type diffHunk = conv.DiffHunk
type toolCall = conv.ToolCall
type toolResult = conv.ToolResult
type plan = conv.Plan
type message = conv.Message
type sessionMeta = conv.SessionMeta
type sessionFull = conv.Session
type conversation = conv.Conversation
