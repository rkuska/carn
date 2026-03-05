package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	"github.com/rs/zerolog"
)

// Messages

type conversationsLoadedMsg struct {
	conversations []conversation
}

type sessionsLoadErrorMsg struct {
	err error
}

type sessionParsedMsg struct {
	session sessionFull
}

type openViewerMsg struct {
	session sessionFull
}

type statusMsg struct {
	text string
}

type clearStatusMsg struct{}

// Commands

func loadSessionsCmd(ctx context.Context, archiveDir string) tea.Cmd {
	return func() tea.Msg {
		sessions, err := scanSessions(ctx, archiveDir)
		if err != nil {
			return sessionsLoadErrorMsg{err: err}
		}

		conversations := groupConversations(sessions)

		// Sort by timestamp descending (newest first)
		sort.Slice(conversations, func(i, j int) bool {
			return conversations[i].timestamp().After(conversations[j].timestamp())
		})

		return conversationsLoadedMsg{conversations: conversations}
	}
}

func parseConversationCmd(ctx context.Context, conv conversation) tea.Cmd {
	return func() tea.Msg {
		session, err := parseConversation(ctx, conv)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msgf("parseConversation failed for %s", conv.id())
			return statusMsg{text: fmt.Sprintf("Error loading session: %v", err)}
		}
		return sessionParsedMsg{session: session}
	}
}

func openConversationCmd(ctx context.Context, conv conversation) tea.Cmd {
	return func() tea.Msg {
		session, err := parseConversationWithSubagents(ctx, conv)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msgf("parseConversationWithSubagents failed for %s", conv.id())
			return statusMsg{text: fmt.Sprintf("Error loading session: %v", err)}
		}
		return openViewerMsg{session: session}
	}
}

func openConversationCmdCached(ctx context.Context, conv conversation, parent sessionFull) tea.Cmd {
	return func() tea.Msg {
		session := parseConversationWithSubagentsCached(ctx, conv, parent)
		return openViewerMsg{session: session}
	}
}

func copyTranscriptCmd(session sessionFull, opts transcriptOptions) tea.Cmd {
	return func() tea.Msg {
		text := renderTranscript(session, opts)
		if err := clipboard.WriteAll(text); err != nil {
			return statusMsg{text: fmt.Sprintf("Copy failed: %v", err)}
		}
		return statusMsg{text: "Transcript copied to clipboard"}
	}
}

func copyFromConversationCmd(ctx context.Context, conv conversation) tea.Cmd {
	return func() tea.Msg {
		session, err := parseConversation(ctx, conv)
		if err != nil {
			return statusMsg{text: fmt.Sprintf("Copy failed: %v", err)}
		}
		text := renderTranscript(session, transcriptOptions{})
		if err := clipboard.WriteAll(text); err != nil {
			return statusMsg{text: fmt.Sprintf("Copy failed: %v", err)}
		}
		return statusMsg{text: "Transcript copied to clipboard"}
	}
}

func exportTranscriptCmd(session sessionFull, opts transcriptOptions) tea.Cmd {
	return func() tea.Msg {
		text := renderTranscript(session, opts)
		return exportText(text, session.meta)
	}
}

func exportFromConversationCmd(ctx context.Context, conv conversation) tea.Cmd {
	return func() tea.Msg {
		session, err := parseConversation(ctx, conv)
		if err != nil {
			return statusMsg{text: fmt.Sprintf("Export failed: %v", err)}
		}
		text := renderTranscript(session, transcriptOptions{})
		return exportText(text, session.meta)
	}
}

func exportText(text string, meta sessionMeta) statusMsg {
	name := fmt.Sprintf("claude-session-%s.md", meta.slug)
	if meta.slug == "" {
		name = fmt.Sprintf("claude-session-%s.md", meta.id[:8])
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return statusMsg{text: fmt.Sprintf("Export failed: %v", err)}
	}
	outPath := filepath.Join(home, "Desktop", name)

	if err := os.WriteFile(outPath, []byte(text), 0o644); err != nil {
		return statusMsg{text: fmt.Sprintf("Export failed: %v", err)}
	}
	return statusMsg{text: fmt.Sprintf("Exported to %s", outPath)}
}

func openInEditorCmd(filePath string) tea.Cmd {
	return func() tea.Msg {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		cmd := exec.Command(editor, filePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return statusMsg{text: fmt.Sprintf("Editor failed: %v", err)}
		}
		return nil
	}
}

func resumeSessionCmd(sessionID string) tea.Cmd {
	return tea.ExecProcess(
		exec.Command("claude", "--resume", sessionID),
		func(err error) tea.Msg {
			if err != nil {
				return statusMsg{text: fmt.Sprintf("Resume failed: %v", err)}
			}
			return nil
		},
	)
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}
