package conversation

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Conversation struct {
	Ref           Ref
	Name          string
	Project       Project
	Sessions      []SessionMeta
	PlanCount     int
	SearchPreview string
}

func (c Conversation) DisplayName() string {
	if c.Name != "" {
		return c.Name
	}
	if fm := c.FirstMessage(); fm != "" {
		return Truncate(fm, maxSlugFromMessage)
	}
	return "untitled"
}

func (c Conversation) FilterValue() string {
	return c.Title() + "\n" + c.Description()
}

func (c Conversation) Title() string {
	date := c.Timestamp().Format("2006-01-02 15:04")
	title := fmt.Sprintf("%s / %s  %s", c.Project.DisplayName, c.DisplayName(), date)
	if c.IsSubagent() {
		title = "[sub] " + title
	}
	if branch := c.GitBranch(); branch != "" {
		title += "  " + branch
	}
	if len(c.Sessions) > 1 {
		title += fmt.Sprintf("  (%d parts)", len(c.Sessions))
	}
	return title
}

func (c Conversation) Description() string {
	msgCount := c.TotalMessageCount()
	mainCount := c.MainMessageCount()
	desc := fmt.Sprintf("%s  %d msgs", c.Model(), msgCount)
	if mainCount > 0 && mainCount != msgCount {
		desc = fmt.Sprintf("%s  %d msgs (%d main)", c.Model(), msgCount, mainCount)
	}
	if v := c.Version(); v != "" {
		desc = v + "  " + desc
	}
	if total := c.TotalTokenUsage().TotalTokens(); total > 0 {
		desc += fmt.Sprintf("  %dk tokens", total/1000)
	}
	if d := c.Duration(); d > 0 {
		desc += "  " + FormatDuration(d)
	}
	if counts := c.TotalToolCounts(); len(counts) > 0 {
		desc += "  " + FormatToolCounts(counts)
	}
	if preview := c.SearchPreview; preview != "" {
		desc += "\n" + preview
	} else if fm := c.FirstMessage(); fm != "" {
		desc += "\n" + fm
	}
	return desc
}

func (c Conversation) ID() string {
	if c.Ref.ID != "" {
		return c.Ref.ID
	}
	return c.Sessions[0].ID
}

func (c Conversation) CacheKey() string {
	if key := c.Ref.CacheKey(); key != "" {
		return key
	}
	return c.ID()
}

func (c Conversation) ResumeID() string {
	return c.Sessions[len(c.Sessions)-1].ID
}

func (c Conversation) ResumeCWD() string {
	return c.Sessions[len(c.Sessions)-1].CWD
}

func (c Conversation) Timestamp() time.Time {
	return c.Sessions[0].Timestamp
}

func (c Conversation) FilePaths() []string {
	paths := make([]string, len(c.Sessions))
	for i, s := range c.Sessions {
		paths[i] = s.FilePath
	}
	return paths
}

func (c Conversation) LatestFilePath() string {
	return c.Sessions[len(c.Sessions)-1].FilePath
}

func (c Conversation) FirstMessage() string {
	return c.Sessions[0].FirstMessage
}

func (c Conversation) TotalMessageCount() int {
	total := 0
	for _, s := range c.Sessions {
		total += s.MessageCount
	}
	return total
}

func (c Conversation) MainMessageCount() int {
	total := 0
	for _, s := range c.Sessions {
		total += s.MainMessageCount
	}
	return total
}

func (c Conversation) TotalTokenUsage() TokenUsage {
	var total TokenUsage
	for _, s := range c.Sessions {
		total.InputTokens += s.TotalUsage.InputTokens
		total.CacheCreationInputTokens += s.TotalUsage.CacheCreationInputTokens
		total.CacheReadInputTokens += s.TotalUsage.CacheReadInputTokens
		total.OutputTokens += s.TotalUsage.OutputTokens
	}
	return total
}

func (c Conversation) Duration() time.Duration {
	earliest := c.Sessions[0].Timestamp
	var latest time.Time
	for _, s := range c.Sessions {
		if s.LastTimestamp.After(latest) {
			latest = s.LastTimestamp
		}
	}
	if earliest.IsZero() || latest.IsZero() {
		return 0
	}
	return latest.Sub(earliest)
}

func (c Conversation) TotalToolCounts() map[string]int {
	merged := make(map[string]int)
	for _, s := range c.Sessions {
		for name, count := range s.ToolCounts {
			merged[name] += count
		}
	}
	return merged
}

func FormatToolCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}

	type toolCount struct {
		name  string
		count int
	}

	sorted := make([]toolCount, 0, len(counts))
	for name, count := range counts {
		sorted = append(sorted, toolCount{name: name, count: count})
	}

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].count != sorted[j].count {
			return sorted[i].count > sorted[j].count
		}
		return sorted[i].name < sorted[j].name
	})

	limit := min(len(sorted), 3)
	parts := make([]string, limit)
	for i := range limit {
		parts[i] = fmt.Sprintf("%s:%d", sorted[i].name, sorted[i].count)
	}
	return strings.Join(parts, " ")
}

func (c Conversation) IsSubagent() bool {
	return c.Sessions[0].IsSubagent
}

func (c Conversation) Model() string {
	if m := c.Sessions[0].Model; m != "" {
		return m
	}
	for i := len(c.Sessions) - 1; i >= 0; i-- {
		if m := c.Sessions[i].Model; m != "" {
			return m
		}
	}
	return ""
}

func (c Conversation) Version() string {
	for i := len(c.Sessions) - 1; i >= 0; i-- {
		if v := c.Sessions[i].Version; v != "" {
			return v
		}
	}
	return ""
}

func (c Conversation) GitBranch() string {
	return c.Sessions[0].GitBranch
}
