package conversation

import (
	"fmt"
	"strings"
	"time"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"

	ContentTypeToolResult = "tool_result"
)

type MessageVisibility string

const (
	MessageVisibilityVisible      MessageVisibility = ""
	MessageVisibilityHiddenSystem MessageVisibility = "hidden_system"
)

type Project struct {
	DisplayName string
}

type ResumeTarget struct {
	Provider Provider
	ID       string
	CWD      string
}

type TokenUsage struct {
	InputTokens              int
	CacheCreationInputTokens int
	CacheReadInputTokens     int
	OutputTokens             int
}

func (u TokenUsage) TotalTokens() int {
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens + u.OutputTokens
}

type DiffHunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Lines    []string
}

type ToolCall struct {
	Name    string
	Summary string
}

type ToolResult struct {
	ToolName        string
	ToolSummary     string
	Content         string
	IsError         bool
	StructuredPatch []DiffHunk
}

func (r ToolResult) IsRejected() bool {
	if !r.IsError {
		return false
	}

	lower := strings.ToLower(strings.TrimSpace(r.Content))
	return strings.Contains(lower, "the user doesn't want to proceed with this tool use") ||
		strings.Contains(lower, "the user does not want to proceed with this tool use") ||
		strings.Contains(lower, "tool use was rejected") ||
		strings.Contains(lower, "user rejected tool use")
}

type SessionMeta struct {
	ID                    string
	Project               Project
	Slug                  string
	Timestamp             time.Time
	LastTimestamp         time.Time
	CWD                   string
	GitBranch             string
	Version               string
	Model                 string
	FirstMessage          string
	MessageCount          int
	MainMessageCount      int
	UserMessageCount      int
	AssistantMessageCount int
	FilePath              string
	TotalUsage            TokenUsage
	ToolCounts            map[string]int
	ToolErrorCounts       map[string]int
	ToolRejectCounts      map[string]int
	IsSubagent            bool
}

const maxSlugFromMessage = 40
const untitledDisplayName = "untitled"

func (s SessionMeta) DisplaySlug() string {
	if s.Slug != "" {
		return s.Slug
	}
	if s.FirstMessage != "" {
		return Truncate(s.FirstMessage, maxSlugFromMessage)
	}
	return untitledDisplayName
}

func (s SessionMeta) Duration() time.Duration {
	if s.LastTimestamp.IsZero() || s.Timestamp.IsZero() {
		return 0
	}
	return s.LastTimestamp.Sub(s.Timestamp)
}

func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

func (s SessionMeta) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s", s.Project.DisplayName, s.DisplaySlug(), s.FirstMessage, s.GitBranch)
}

func (s SessionMeta) Title() string {
	return buildDisplayTitle(
		s.Project.DisplayName,
		s.DisplaySlug(),
		s.Timestamp.Format("2006-01-02 15:04"),
		s.IsSubagent,
		s.GitBranch,
		0,
	)
}

func (s SessionMeta) Description() string {
	return buildDisplayDescription(
		s.Version,
		s.Model,
		s.MessageCount,
		s.MainMessageCount,
		s.TotalUsage.TotalTokens(),
		formatDisplayDuration(s.Duration()),
		s.ToolCounts,
		s.FirstMessage,
	)
}

type Message struct {
	Role              Role
	Text              string
	Thinking          string
	HasHiddenThinking bool
	ToolCalls         []ToolCall
	ToolResults       []ToolResult
	Plans             []Plan
	Visibility        MessageVisibility
	IsSidechain       bool
	IsAgentDivider    bool
	Usage             TokenUsage
}

func (m Message) IsVisible() bool {
	return m.Visibility != MessageVisibilityHiddenSystem
}

func (m Message) HasThinking() bool {
	return m.Thinking != "" || m.HasHiddenThinking
}

type Session struct {
	Meta     SessionMeta
	Messages []Message
}
