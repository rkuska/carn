package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// conversation groups one or more sessionMeta entries that share
// the same slug and project into a single logical unit.
type conversation struct {
	name     string // session slug (session name)
	project  project
	sessions []sessionMeta // chronologically ordered, len >= 1
}

// FilterValue implements list.Item for fuzzy filtering.
func (c conversation) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s", c.project.displayName, c.name, c.firstMessage(), c.gitBranch())
}

// Title implements list.DefaultItem.
func (c conversation) Title() string {
	date := c.timestamp().Format("2006-01-02 15:04")
	title := fmt.Sprintf("%s / %s  %s", c.project.displayName, c.name, date)
	if c.isSubagent() {
		title = "[sub] " + title
	}
	if branch := c.gitBranch(); branch != "" {
		title += "  " + branch
	}
	if len(c.sessions) > 1 {
		title += fmt.Sprintf("  (%d parts)", len(c.sessions))
	}
	return title
}

// Description implements list.DefaultItem.
func (c conversation) Description() string {
	msgCount := c.totalMessageCount()
	mainCount := c.mainMessageCount()
	desc := fmt.Sprintf("%s  %d msgs", c.model(), msgCount)
	if mainCount > 0 && mainCount != msgCount {
		desc = fmt.Sprintf("%s  %d msgs (%d main)", c.model(), msgCount, mainCount)
	}
	if v := c.version(); v != "" {
		desc = v + "  " + desc
	}
	if total := c.totalTokenUsage().totalTokens(); total > 0 {
		desc += fmt.Sprintf("  %dk tokens", total/1000)
	}
	if d := c.duration(); d > 0 {
		desc += "  " + formatDuration(d)
	}
	if counts := c.totalToolCounts(); len(counts) > 0 {
		desc += "  " + formatToolCounts(counts)
	}
	if fm := c.firstMessage(); fm != "" {
		desc += "\n" + fm
	}
	return desc
}

// id returns the primary (first) session's ID — used as a stable cache key.
func (c conversation) id() string {
	return c.sessions[0].id
}

// resumeID returns the latest session's ID — for `claude --resume`.
func (c conversation) resumeID() string {
	return c.sessions[len(c.sessions)-1].id
}

// timestamp returns the earliest timestamp across all sessions.
func (c conversation) timestamp() time.Time {
	return c.sessions[0].timestamp
}

// filePaths returns all file paths in chronological order.
func (c conversation) filePaths() []string {
	paths := make([]string, len(c.sessions))
	for i, s := range c.sessions {
		paths[i] = s.filePath
	}
	return paths
}

// latestFilePath returns the most recent session's file path — for editor.
func (c conversation) latestFilePath() string {
	return c.sessions[len(c.sessions)-1].filePath
}

// firstMessage returns the first non-interrupt user message from the primary session.
func (c conversation) firstMessage() string {
	return c.sessions[0].firstMessage
}

// totalMessageCount sums message counts across all sessions.
func (c conversation) totalMessageCount() int {
	total := 0
	for _, s := range c.sessions {
		total += s.messageCount
	}
	return total
}

// mainMessageCount sums main (non-sidechain) message counts across all sessions.
func (c conversation) mainMessageCount() int {
	total := 0
	for _, s := range c.sessions {
		total += s.mainMessageCount
	}
	return total
}

// totalTokenUsage sums token usage across all sessions.
func (c conversation) totalTokenUsage() tokenUsage {
	var total tokenUsage
	for _, s := range c.sessions {
		total.inputTokens += s.totalUsage.inputTokens
		total.cacheCreationInputTokens += s.totalUsage.cacheCreationInputTokens
		total.cacheReadInputTokens += s.totalUsage.cacheReadInputTokens
		total.outputTokens += s.totalUsage.outputTokens
	}
	return total
}

// duration returns the total duration from the earliest session start
// to the latest session end across all sessions in the conversation.
func (c conversation) duration() time.Duration {
	earliest := c.sessions[0].timestamp
	var latest time.Time
	for _, s := range c.sessions {
		if s.lastTimestamp.After(latest) {
			latest = s.lastTimestamp
		}
	}
	if earliest.IsZero() || latest.IsZero() {
		return 0
	}
	return latest.Sub(earliest)
}

// totalToolCounts aggregates tool invocation counts across all sessions.
func (c conversation) totalToolCounts() map[string]int {
	merged := make(map[string]int)
	for _, s := range c.sessions {
		for name, count := range s.toolCounts {
			merged[name] += count
		}
	}
	return merged
}

// formatToolCounts returns a compact summary of the top 3 tools by count,
// e.g. "Bash:12 Read:8 Edit:5".
func formatToolCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}

	type toolCount struct {
		name  string
		count int
	}
	sorted := make([]toolCount, 0, len(counts))
	for name, count := range counts {
		sorted = append(sorted, toolCount{name, count})
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

// isSubagent returns true if the primary session is a subagent.
func (c conversation) isSubagent() bool {
	return c.sessions[0].isSubagent
}

// model returns the model from the primary session, falling back to
// the latest session with a model set.
func (c conversation) model() string {
	if m := c.sessions[0].model; m != "" {
		return m
	}
	for i := len(c.sessions) - 1; i >= 0; i-- {
		if m := c.sessions[i].model; m != "" {
			return m
		}
	}
	return ""
}

// version returns the version from the latest session.
func (c conversation) version() string {
	for i := len(c.sessions) - 1; i >= 0; i-- {
		if v := c.sessions[i].version; v != "" {
			return v
		}
	}
	return ""
}

// gitBranch returns the git branch from the primary session.
func (c conversation) gitBranch() string {
	return c.sessions[0].gitBranch
}

// groupKey is the key used for grouping sessions into conversations.
type groupKey struct {
	dirName string
	slug    string
}

// groupConversations groups sessions by (project.dirName, slug) into conversations.
// Sessions with empty slug or subagent sessions are not grouped — each becomes
// its own single-session conversation. Within each group, sessions are sorted
// by timestamp ascending. The returned conversations are unsorted — caller sorts.
func groupConversations(sessions []sessionMeta) []conversation {
	groups := make(map[groupKey][]sessionMeta)
	var ungrouped []sessionMeta

	for _, s := range sessions {
		if s.isSubagent || s.slug == "" {
			ungrouped = append(ungrouped, s)
			continue
		}
		key := groupKey{dirName: s.project.dirName, slug: s.slug}
		groups[key] = append(groups[key], s)
	}

	conversations := make([]conversation, 0, len(groups)+len(ungrouped))

	for key, members := range groups {
		sort.Slice(members, func(i, j int) bool {
			return members[i].timestamp.Before(members[j].timestamp)
		})
		conversations = append(conversations, conversation{
			name:     key.slug,
			project:  members[0].project,
			sessions: members,
		})
	}

	for _, s := range ungrouped {
		conversations = append(conversations, conversation{
			name:     s.slug,
			project:  s.project,
			sessions: []sessionMeta{s},
		})
	}

	return conversations
}
