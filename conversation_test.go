package main

import (
	"strings"
	"testing"
	"time"
)

func TestGroupConversations(t *testing.T) {
	t.Parallel()

	proj := project{dirName: "proj-a", displayName: "a"}
	ts1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	ts3 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("same slug merges into one conversation", func(t *testing.T) {
		t.Parallel()
		sessions := []sessionMeta{
			{id: "s2", slug: "cheerful-ocean", project: proj, timestamp: ts2, filePath: "/f2", messageCount: 5},
			{id: "s1", slug: "cheerful-ocean", project: proj, timestamp: ts1, filePath: "/f1", messageCount: 10},
		}
		convs := groupConversations(sessions)
		if len(convs) != 1 {
			t.Fatalf("expected 1 conversation, got %d", len(convs))
		}
		c := convs[0]
		if len(c.sessions) != 2 {
			t.Fatalf("expected 2 sessions, got %d", len(c.sessions))
		}
		// Should be sorted chronologically (s1 before s2)
		if c.sessions[0].id != "s1" {
			t.Errorf("first session id = %q, want s1", c.sessions[0].id)
		}
		if c.sessions[1].id != "s2" {
			t.Errorf("second session id = %q, want s2", c.sessions[1].id)
		}
	})

	t.Run("different slugs stay separate", func(t *testing.T) {
		t.Parallel()
		sessions := []sessionMeta{
			{id: "s1", slug: "slug-a", project: proj, timestamp: ts1},
			{id: "s2", slug: "slug-b", project: proj, timestamp: ts2},
		}
		convs := groupConversations(sessions)
		if len(convs) != 2 {
			t.Fatalf("expected 2 conversations, got %d", len(convs))
		}
	})

	t.Run("subagents not grouped", func(t *testing.T) {
		t.Parallel()
		sessions := []sessionMeta{
			{id: "s1", slug: "same-slug", project: proj, timestamp: ts1, isSubagent: true},
			{id: "s2", slug: "same-slug", project: proj, timestamp: ts2, isSubagent: true},
		}
		convs := groupConversations(sessions)
		if len(convs) != 2 {
			t.Fatalf("expected 2 separate conversations for subagents, got %d", len(convs))
		}
	})

	t.Run("empty slug not grouped", func(t *testing.T) {
		t.Parallel()
		sessions := []sessionMeta{
			{id: "s1", slug: "", project: proj, timestamp: ts1},
			{id: "s2", slug: "", project: proj, timestamp: ts2},
		}
		convs := groupConversations(sessions)
		if len(convs) != 2 {
			t.Fatalf("expected 2 separate conversations for empty slugs, got %d", len(convs))
		}
	})

	t.Run("single session group works", func(t *testing.T) {
		t.Parallel()
		sessions := []sessionMeta{
			{id: "s1", slug: "unique-slug", project: proj, timestamp: ts1},
		}
		convs := groupConversations(sessions)
		if len(convs) != 1 {
			t.Fatalf("expected 1 conversation, got %d", len(convs))
		}
		if len(convs[0].sessions) != 1 {
			t.Errorf("expected 1 session, got %d", len(convs[0].sessions))
		}
	})

	t.Run("different projects with same slug stay separate", func(t *testing.T) {
		t.Parallel()
		projB := project{dirName: "proj-b", displayName: "b"}
		sessions := []sessionMeta{
			{id: "s1", slug: "same-slug", project: proj, timestamp: ts1},
			{id: "s2", slug: "same-slug", project: projB, timestamp: ts2},
		}
		convs := groupConversations(sessions)
		if len(convs) != 2 {
			t.Fatalf("expected 2 conversations for different projects, got %d", len(convs))
		}
	})

	t.Run("mixed grouped and ungrouped", func(t *testing.T) {
		t.Parallel()
		sessions := []sessionMeta{
			{id: "s1", slug: "grouped", project: proj, timestamp: ts1},
			{id: "s2", slug: "grouped", project: proj, timestamp: ts2},
			{id: "s3", slug: "", project: proj, timestamp: ts3},
			{id: "s4", slug: "grouped", project: proj, timestamp: ts3, isSubagent: true},
		}
		convs := groupConversations(sessions)
		// 1 grouped conversation + 1 empty slug + 1 subagent = 3
		if len(convs) != 3 {
			t.Fatalf("expected 3 conversations, got %d", len(convs))
		}
	})
}

func TestConversationAccessors(t *testing.T) {
	t.Parallel()

	ts1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)

	conv := conversation{
		name:    "test-slug",
		project: project{dirName: "proj", displayName: "proj"},
		sessions: []sessionMeta{
			{
				id: "first-id", slug: "test-slug", timestamp: ts1,
				filePath: "/path/first.jsonl", firstMessage: "hello world",
				messageCount: 10, mainMessageCount: 8,
				model: "claude-3", version: "1.0.0", gitBranch: "main",
				totalUsage: tokenUsage{inputTokens: 100, outputTokens: 50},
			},
			{
				id: "second-id", slug: "test-slug", timestamp: ts2,
				filePath: "/path/second.jsonl", firstMessage: "[Request interrupted by user]",
				messageCount: 5, mainMessageCount: 4,
				model: "", version: "1.1.0",
				totalUsage: tokenUsage{inputTokens: 200, outputTokens: 100},
			},
		},
	}

	t.Run("id returns first session id", func(t *testing.T) {
		t.Parallel()
		if got := conv.id(); got != "first-id" {
			t.Errorf("id() = %q, want %q", got, "first-id")
		}
	})

	t.Run("resumeID returns last session id", func(t *testing.T) {
		t.Parallel()
		if got := conv.resumeID(); got != "second-id" {
			t.Errorf("resumeID() = %q, want %q", got, "second-id")
		}
	})

	t.Run("timestamp returns earliest", func(t *testing.T) {
		t.Parallel()
		if got := conv.timestamp(); !got.Equal(ts1) {
			t.Errorf("timestamp() = %v, want %v", got, ts1)
		}
	})

	t.Run("filePaths returns all in order", func(t *testing.T) {
		t.Parallel()
		got := conv.filePaths()
		if len(got) != 2 {
			t.Fatalf("filePaths() len = %d, want 2", len(got))
		}
		if got[0] != "/path/first.jsonl" || got[1] != "/path/second.jsonl" {
			t.Errorf("filePaths() = %v", got)
		}
	})

	t.Run("latestFilePath returns last", func(t *testing.T) {
		t.Parallel()
		if got := conv.latestFilePath(); got != "/path/second.jsonl" {
			t.Errorf("latestFilePath() = %q, want %q", got, "/path/second.jsonl")
		}
	})

	t.Run("firstMessage from primary session", func(t *testing.T) {
		t.Parallel()
		if got := conv.firstMessage(); got != "hello world" {
			t.Errorf("firstMessage() = %q, want %q", got, "hello world")
		}
	})

	t.Run("totalMessageCount sums all", func(t *testing.T) {
		t.Parallel()
		if got := conv.totalMessageCount(); got != 15 {
			t.Errorf("totalMessageCount() = %d, want 15", got)
		}
	})

	t.Run("mainMessageCount sums all", func(t *testing.T) {
		t.Parallel()
		if got := conv.mainMessageCount(); got != 12 {
			t.Errorf("mainMessageCount() = %d, want 12", got)
		}
	})

	t.Run("totalTokenUsage sums all", func(t *testing.T) {
		t.Parallel()
		usage := conv.totalTokenUsage()
		if usage.inputTokens != 300 {
			t.Errorf("inputTokens = %d, want 300", usage.inputTokens)
		}
		if usage.outputTokens != 150 {
			t.Errorf("outputTokens = %d, want 150", usage.outputTokens)
		}
	})

	t.Run("model from primary", func(t *testing.T) {
		t.Parallel()
		if got := conv.model(); got != "claude-3" {
			t.Errorf("model() = %q, want %q", got, "claude-3")
		}
	})

	t.Run("model falls back to latest", func(t *testing.T) {
		t.Parallel()
		c := conversation{
			sessions: []sessionMeta{
				{model: ""},
				{model: "claude-4"},
			},
		}
		if got := c.model(); got != "claude-4" {
			t.Errorf("model() = %q, want %q", got, "claude-4")
		}
	})

	t.Run("version from latest", func(t *testing.T) {
		t.Parallel()
		if got := conv.version(); got != "1.1.0" {
			t.Errorf("version() = %q, want %q", got, "1.1.0")
		}
	})

	t.Run("gitBranch from primary", func(t *testing.T) {
		t.Parallel()
		if got := conv.gitBranch(); got != "main" {
			t.Errorf("gitBranch() = %q, want %q", got, "main")
		}
	})

	t.Run("isSubagent false", func(t *testing.T) {
		t.Parallel()
		if conv.isSubagent() {
			t.Error("expected isSubagent() = false")
		}
	})
}

func TestConversationListItem(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	conv := conversation{
		name:    "cheerful-ocean",
		project: project{dirName: "proj", displayName: "my/project"},
		sessions: []sessionMeta{
			{
				id: "s1", slug: "cheerful-ocean", timestamp: ts,
				firstMessage: "help me with Go",
				messageCount: 20, mainMessageCount: 18,
				model: "claude-3", version: "1.0.0",
				gitBranch: "feature",
			},
			{
				id: "s2", slug: "cheerful-ocean",
				timestamp:    ts.Add(time.Hour),
				messageCount: 5, mainMessageCount: 4,
			},
		},
	}

	t.Run("FilterValue contains key fields", func(t *testing.T) {
		t.Parallel()
		fv := conv.FilterValue()
		for _, want := range []string{"my/project", "cheerful-ocean", "help me with Go", "feature"} {
			if !strings.Contains(fv, want) {
				t.Errorf("FilterValue() = %q, missing %q", fv, want)
			}
		}
	})

	t.Run("Title contains parts indicator", func(t *testing.T) {
		t.Parallel()
		title := conv.Title()
		if !strings.Contains(title, "(2 parts)") {
			t.Errorf("Title() = %q, missing parts indicator", title)
		}
		if !strings.Contains(title, "my/project") {
			t.Errorf("Title() = %q, missing project name", title)
		}
		if !strings.Contains(title, "cheerful-ocean") {
			t.Errorf("Title() = %q, missing slug", title)
		}
		if !strings.Contains(title, "feature") {
			t.Errorf("Title() = %q, missing git branch", title)
		}
	})

	t.Run("Title without parts for single session", func(t *testing.T) {
		t.Parallel()
		single := conversation{
			name:    "single",
			project: project{displayName: "proj"},
			sessions: []sessionMeta{
				{slug: "single", timestamp: ts},
			},
		}
		title := single.Title()
		if strings.Contains(title, "parts") {
			t.Errorf("single session Title() should not have parts: %q", title)
		}
	})

	t.Run("Description contains summed counts", func(t *testing.T) {
		t.Parallel()
		desc := conv.Description()
		if !strings.Contains(desc, "25 msgs") {
			t.Errorf("Description() = %q, missing total message count", desc)
		}
		if !strings.Contains(desc, "help me with Go") {
			t.Errorf("Description() = %q, missing first message", desc)
		}
	})
}
