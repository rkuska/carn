package conversation

import (
	"fmt"
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

type SessionMeta struct {
	ID               string
	Project          Project
	Slug             string
	Timestamp        time.Time
	LastTimestamp    time.Time
	CWD              string
	GitBranch        string
	Version          string
	Model            string
	FirstMessage     string
	MessageCount     int
	MainMessageCount int
	FilePath         string
	TotalUsage       TokenUsage
	ToolCounts       map[string]int
	IsSubagent       bool
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
	date := s.Timestamp.Format("2006-01-02 15:04")
	title := fmt.Sprintf("%s / %s  %s", s.Project.DisplayName, s.DisplaySlug(), date)
	if s.IsSubagent {
		title = "[sub] " + title
	}
	if s.GitBranch != "" {
		title += "  " + s.GitBranch
	}
	return title
}

func (s SessionMeta) Description() string {
	desc := fmt.Sprintf("%s  %d msgs", s.Model, s.MessageCount)
	if s.MainMessageCount > 0 && s.MainMessageCount != s.MessageCount {
		desc = fmt.Sprintf("%s  %d msgs (%d main)", s.Model, s.MessageCount, s.MainMessageCount)
	}
	if s.Version != "" {
		desc = s.Version + "  " + desc
	}
	if total := s.TotalUsage.TotalTokens(); total > 0 {
		desc += fmt.Sprintf("  %dk tokens", total/1000)
	}
	if d := s.Duration(); d > 0 {
		desc += "  " + FormatDuration(d)
	}
	if len(s.ToolCounts) > 0 {
		desc += "  " + FormatToolCounts(s.ToolCounts)
	}
	if s.FirstMessage != "" {
		desc += "\n" + s.FirstMessage
	}
	return desc
}

type Message struct {
	Role           Role
	Text           string
	Thinking       string
	ToolCalls      []ToolCall
	ToolResults    []ToolResult
	Plans          []Plan
	Visibility     MessageVisibility
	IsSidechain    bool
	IsAgentDivider bool
}

func (m Message) IsVisible() bool {
	return m.Visibility != MessageVisibilityHiddenSystem
}

type Session struct {
	Meta     SessionMeta
	Messages []Message
}
