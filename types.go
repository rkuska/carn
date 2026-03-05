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

type sessionMeta struct {
	id           string
	project      project
	slug         string
	timestamp    time.Time
	cwd          string
	gitBranch    string
	version      string
	model        string
	firstMessage string
	messageCount int
	filePath     string
}

// FilterValue implements list.Item for fuzzy filtering.
func (s sessionMeta) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s", s.project.displayName, s.slug, s.firstMessage, s.gitBranch)
}

// Title implements list.DefaultItem.
func (s sessionMeta) Title() string {
	date := s.timestamp.Format("2006-01-02 15:04")
	title := fmt.Sprintf("%s / %s  %s", s.project.displayName, s.slug, date)
	if s.gitBranch != "" {
		title += "  " + s.gitBranch
	}
	return title
}

// Description implements list.DefaultItem.
func (s sessionMeta) Description() string {
	desc := fmt.Sprintf("%s  %d msgs", s.model, s.messageCount)
	if s.version != "" {
		desc = s.version + "  " + desc
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
	role      role
	timestamp time.Time
	text      string
	thinking  string
	toolCalls []toolCall
}

type toolCall struct {
	name    string
	summary string
}
