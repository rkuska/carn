package main

import (
	"fmt"
	"time"
)

type role string

const (
	roleUser      role = "user"
	roleAssistant role = "assistant"
)

type project struct {
	dirName     string
	displayName string
	path        string
}

type tokenUsage struct {
	inputTokens              int
	cacheCreationInputTokens int
	cacheReadInputTokens     int
	outputTokens             int
}

func (u tokenUsage) totalTokens() int {
	return u.inputTokens + u.outputTokens
}

type toolResult struct {
	toolUseID       string
	toolName        string
	toolSummary     string
	content         string
	structuredPatch []diffHunk
}

type diffHunk struct {
	oldStart int
	oldLines int
	newStart int
	newLines int
	lines    []string
}

type sessionMeta struct {
	id               string
	project          project
	slug             string
	timestamp        time.Time
	cwd              string
	gitBranch        string
	version          string
	model            string
	firstMessage     string
	messageCount     int
	mainMessageCount int
	filePath         string
	totalUsage       tokenUsage
	isSubagent       bool
	parentSessionID  string
}

// FilterValue implements list.Item for fuzzy filtering.
func (s sessionMeta) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s", s.project.displayName, s.slug, s.firstMessage, s.gitBranch)
}

// Title implements list.DefaultItem.
func (s sessionMeta) Title() string {
	date := s.timestamp.Format("2006-01-02 15:04")
	title := fmt.Sprintf("%s / %s  %s", s.project.displayName, s.slug, date)
	if s.isSubagent {
		title = "[sub] " + title
	}
	if s.gitBranch != "" {
		title += "  " + s.gitBranch
	}
	return title
}

// Description implements list.DefaultItem.
func (s sessionMeta) Description() string {
	desc := fmt.Sprintf("%s  %d msgs", s.model, s.messageCount)
	if s.mainMessageCount > 0 && s.mainMessageCount != s.messageCount {
		desc = fmt.Sprintf("%s  %d msgs (%d main)", s.model, s.messageCount, s.mainMessageCount)
	}
	if s.version != "" {
		desc = s.version + "  " + desc
	}
	if total := s.totalUsage.totalTokens(); total > 0 {
		desc += fmt.Sprintf("  %dk tokens", total/1000)
	}
	if s.firstMessage != "" {
		desc += "\n" + s.firstMessage
	}
	return desc
}

type sessionFull struct {
	meta     sessionMeta
	messages []message
}

type message struct {
	role           role
	timestamp      time.Time
	text           string
	thinking       string
	toolCalls      []toolCall
	toolResults    []toolResult
	usage          tokenUsage
	stopReason     string
	uuid           string
	parentUUID     string
	isSidechain    bool
	isAgentDivider bool
}

type toolCall struct {
	id      string
	name    string
	summary string
}
