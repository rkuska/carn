package conversation

import (
	"strconv"
	"strings"
	"time"
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
	return buildDisplayTitle(
		c.Project.DisplayName,
		c.DisplayName(),
		c.Timestamp().Format("2006-01-02 15:04"),
		FormatRelativeTime(c.Timestamp(), currentDisplayNow()),
		c.IsSubagent(),
		c.GitBranch(),
		c.PartCount(),
	)
}

func (c Conversation) Description() string {
	if c.displayCache != nil && c.displayCache.description != "" {
		return c.displayCache.description
	}
	return c.computeDescription()
}

func (c Conversation) computeDescription() string {
	msgCount, mainCount := c.messageCounts()
	trailing := c.FirstMessage()
	if preview := c.SearchPreview; preview != "" {
		trailing = preview
	}
	return buildDisplayDescription(
		c.Version(),
		c.Model(),
		msgCount,
		mainCount,
		c.TotalTokenUsage().TotalTokens(),
		formatDisplayDuration(c.Duration()),
		c.TotalToolCounts(),
		trailing,
	)
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
	var merged map[string]int
	for _, s := range c.Sessions {
		if len(s.ToolCounts) == 0 {
			continue
		}
		if merged == nil {
			merged = make(map[string]int, len(s.ToolCounts))
		}
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

	top := topToolCounts(counts)
	if len(top) == 0 {
		return ""
	}

	var totalLen int
	for i, tc := range top {
		totalLen += len(tc.name) + len(strconv.Itoa(tc.count)) + 1
		if i > 0 {
			totalLen++
		}
	}

	var builder strings.Builder
	builder.Grow(totalLen)
	for i, tc := range top {
		if i > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(tc.name)
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(tc.count))
	}
	return builder.String()
}

type toolCount struct {
	name  string
	count int
}

func buildDisplayTitle(
	projectName string,
	displayName string,
	date string,
	relativeHint string,
	isSubagent bool,
	branch string,
	partCount int,
) string {
	var builder strings.Builder
	builder.Grow(len(projectName) + len(displayName) + len(date) + len(relativeHint) + len(branch) + 28)
	if isSubagent {
		builder.WriteString("[sub] ")
	}
	builder.WriteString(projectName)
	builder.WriteString(" / ")
	builder.WriteString(displayName)
	builder.WriteString("  ")
	builder.WriteString(date)
	if relativeHint != "" {
		builder.WriteString(" (")
		builder.WriteString(relativeHint)
		builder.WriteByte(')')
	}
	if branch != "" {
		builder.WriteString("  ")
		builder.WriteString(branch)
	}
	if partCount > 1 {
		builder.WriteString("  (")
		builder.WriteString(strconv.Itoa(partCount))
		builder.WriteString(" parts)")
	}
	return builder.String()
}

func buildDisplayDescription(
	version string,
	model string,
	messageCount int,
	mainMessageCount int,
	totalTokens int,
	durationText string,
	toolCounts map[string]int,
	trailing string,
) string {
	toolSummary := FormatToolCounts(toolCounts)

	var builder strings.Builder
	builder.Grow(len(version) + len(model) + len(durationText) + len(toolSummary) + len(trailing) + 48)
	if version != "" {
		builder.WriteString(version)
		builder.WriteString("  ")
	}
	builder.WriteString(model)
	builder.WriteString("  ")
	builder.WriteString(strconv.Itoa(messageCount))
	builder.WriteString(" msgs")
	if mainMessageCount > 0 && mainMessageCount != messageCount {
		builder.WriteString(" (")
		builder.WriteString(strconv.Itoa(mainMessageCount))
		builder.WriteString(" main)")
	}
	if totalTokens > 0 {
		builder.WriteString("  ")
		builder.WriteString(strconv.Itoa(totalTokens / 1000))
		builder.WriteString("k tokens")
	}
	if durationText != "" {
		builder.WriteString("  ")
		builder.WriteString(durationText)
	}
	if toolSummary != "" {
		builder.WriteString("  ")
		builder.WriteString(toolSummary)
	}
	if trailing != "" {
		builder.WriteByte('\n')
		builder.WriteString(trailing)
	}
	return builder.String()
}

func topToolCounts(counts map[string]int) []toolCount {
	top := make([]toolCount, 0, 3)
	for name, count := range counts {
		entry := toolCount{name: name, count: count}
		insertAt := len(top)
		for i := range top {
			if toolCountPrecedes(entry, top[i]) {
				insertAt = i
				break
			}
		}
		if insertAt >= 3 {
			continue
		}
		if len(top) < 3 {
			top = append(top, toolCount{})
		}
		copy(top[insertAt+1:], top[insertAt:])
		top[insertAt] = entry
	}
	return top
}

func toolCountPrecedes(left toolCount, right toolCount) bool {
	if left.count != right.count {
		return left.count > right.count
	}
	return left.name < right.name
}

func formatDisplayDuration(duration time.Duration) string {
	if duration <= 0 {
		return ""
	}
	return FormatDuration(duration)
}
