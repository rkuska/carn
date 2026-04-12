package conversation

import (
	"fmt"
	"strings"
	"sync"
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
	ReasoningOutputTokens    int
}

func (u TokenUsage) TotalTokens() int {
	return u.InputTokens +
		u.CacheCreationInputTokens +
		u.CacheReadInputTokens +
		u.OutputTokens +
		u.ReasoningOutputTokens
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
	Action  NormalizedAction
}

type ToolResult struct {
	ToolName        string
	ToolSummary     string
	Content         string
	IsError         bool
	StructuredPatch []DiffHunk
	Action          NormalizedAction
}

func (r ToolResult) IsRejected() bool {
	if !r.IsError {
		return false
	}

	return IsRejectedToolResultContent(r.Content)
}

func IsRejectedToolResultContent(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	return strings.Contains(lower, "the user doesn't want to proceed with this tool use") ||
		strings.Contains(lower, "the user does not want to proceed with this tool use") ||
		strings.Contains(lower, "tool use was rejected") ||
		strings.Contains(lower, "user rejected tool use")
}

type SessionMeta struct {
	ID                    string
	Provider              Provider
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
	ActionCounts          map[string]int
	ActionErrorCounts     map[string]int
	ActionRejectCounts    map[string]int
	Performance           SessionPerformanceMeta
	IsSubagent            bool
}

const maxSlugFromMessage = 40
const untitledDisplayName = "untitled"

var displayNow = time.Now
var displayNowMu sync.RWMutex

func currentDisplayNow() time.Time {
	displayNowMu.RLock()
	now := displayNow
	displayNowMu.RUnlock()
	return now()
}

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

func FormatRelativeTime(t time.Time, now time.Time) string {
	if t.IsZero() {
		return ""
	}

	duration := now.Sub(t)
	if duration < time.Minute {
		return "now"
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	}
	if duration < 30*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	}
	return ""
}

func (s SessionMeta) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s", s.Project.DisplayName, s.DisplaySlug(), s.FirstMessage, s.GitBranch)
}

func (s SessionMeta) Title() string {
	return buildDisplayTitle(
		s.Project.DisplayName,
		s.DisplaySlug(),
		s.Timestamp.Format("2006-01-02 15:04"),
		FormatRelativeTime(s.Timestamp, currentDisplayNow()),
		s.IsSubagent,
		s.GitBranch,
		0,
	)
}

func SetNowForTesting(now func() time.Time) func() {
	displayNowMu.Lock()
	previous := displayNow
	if now == nil {
		displayNow = time.Now
	} else {
		displayNow = now
	}
	displayNowMu.Unlock()
	return func() {
		displayNowMu.Lock()
		displayNow = previous
		displayNowMu.Unlock()
	}
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
	Performance       MessagePerformanceMeta
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
