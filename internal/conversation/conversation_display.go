package conversation

import (
	"fmt"
	"sort"
	"strings"
)

type conversationDisplayCache struct {
	title       string
	description string
	toolCounts  map[string]int
}

func (c Conversation) DisplayName() string {
	if c.Name != "" {
		return c.Name
	}
	if slug := c.firstSession().DisplaySlug(); slug != "" && slug != untitledDisplayName {
		return slug
	}
	if fm := c.FirstMessage(); fm != "" {
		return Truncate(fm, maxSlugFromMessage)
	}
	return untitledDisplayName
}

// PrecomputeDisplay eagerly caches derived list-display fields for a value
// conversation. This is the deliberate exception to the value-receiver rule.
func (c *Conversation) PrecomputeDisplay() {
	if c == nil {
		return
	}

	if c.displayCache == nil {
		c.displayCache = &conversationDisplayCache{}
	}
	c.displayCache.toolCounts = c.computeTotalToolCounts()
	c.displayCache.title = c.computeTitle()
	c.displayCache.description = c.computeDescription()
}

func (c *Conversation) SetSearchPreview(preview string) {
	if c == nil {
		return
	}

	c.SearchPreview = preview
	if c.displayCache != nil {
		c.displayCache.description = ""
	}
}

func (c Conversation) FilterValue() string {
	return c.Title() + "\n" + c.Description()
}

func (c Conversation) Title() string {
	if c.displayCache != nil && c.displayCache.title != "" {
		return c.displayCache.title
	}
	return c.computeTitle()
}

func (c Conversation) computeTitle() string {
	date := c.Timestamp().Format("2006-01-02 15:04")
	title := fmt.Sprintf("%s / %s  %s", c.Project.DisplayName, c.DisplayName(), date)
	if c.IsSubagent() {
		title = "[sub] " + title
	}
	if branch := c.GitBranch(); branch != "" {
		title += "  " + branch
	}
	if parts := c.PartCount(); parts > 1 {
		title += fmt.Sprintf("  (%d parts)", parts)
	}
	return title
}

func (c Conversation) Description() string {
	if c.displayCache != nil && c.displayCache.description != "" {
		return c.displayCache.description
	}
	return c.computeDescription()
}

func (c Conversation) computeDescription() string {
	msgCount, mainCount := c.messageCounts()
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

func (c Conversation) messageCounts() (total, main int) {
	for _, s := range c.Sessions {
		total += s.MessageCount
		main += s.MainMessageCount
	}
	return total, main
}

func (c Conversation) TotalToolCounts() map[string]int {
	if c.displayCache != nil && c.displayCache.toolCounts != nil {
		return c.displayCache.toolCounts
	}
	return c.computeTotalToolCounts()
}

func (c Conversation) computeTotalToolCounts() map[string]int {
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
