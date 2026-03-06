package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

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

type openViewerMsg struct {
	conversationID string
	session        sessionFull
}

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

func openConversationCmd(ctx context.Context, conv conversation) tea.Cmd {
	return func() tea.Msg {
		session, err := parseConversationWithSubagents(ctx, conv)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msgf("parseConversationWithSubagents failed for %s", conv.id())
			return errorNotification(fmt.Sprintf("load session failed: %v", err))
		}
		return openViewerMsg{conversationID: conv.id(), session: session}
	}
}

func openConversationCmdCached(ctx context.Context, conv conversation, parent sessionFull) tea.Cmd {
	return func() tea.Msg {
		session := parseConversationWithSubagentsCached(ctx, conv, parent)
		return openViewerMsg{conversationID: conv.id(), session: session}
	}
}

func copyTranscriptCmd(session sessionFull, opts transcriptOptions) tea.Cmd {
	return func() tea.Msg {
		text := renderTranscript(session, opts)
		if err := clipboard.WriteAll(text); err != nil {
			return errorNotification(fmt.Sprintf("copy failed: %v", err))
		}
		return successNotification("transcript copied to clipboard")
	}
}

func copyFromConversationCmd(ctx context.Context, conv conversation) tea.Cmd {
	return func() tea.Msg {
		session, err := parseConversation(ctx, conv)
		if err != nil {
			return errorNotification(fmt.Sprintf("copy failed: %v", err))
		}
		text := renderTranscript(session, transcriptOptions{})
		if err := clipboard.WriteAll(text); err != nil {
			return errorNotification(fmt.Sprintf("copy failed: %v", err))
		}
		return successNotification("transcript copied to clipboard")
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
			return errorNotification(fmt.Sprintf("export failed: %v", err))
		}
		text := renderTranscript(session, transcriptOptions{})
		return exportText(text, session.meta)
	}
}

func exportText(text string, meta sessionMeta) notificationMsg {
	name := fmt.Sprintf("claude-session-%s.md", meta.slug)
	if meta.slug == "" {
		name = fmt.Sprintf("claude-session-%s.md", meta.id[:8])
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return errorNotification(fmt.Sprintf("export failed: %v", err))
	}
	outPath := filepath.Join(home, "Desktop", name)

	if err := os.WriteFile(outPath, []byte(text), 0o644); err != nil {
		return errorNotification(fmt.Sprintf("export failed: %v", err))
	}
	return successNotification(fmt.Sprintf("exported to %s", outPath))
}

func openInEditorCmd(filePath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, filePath)

	return tea.ExecProcess(
		cmd,
		func(err error) tea.Msg {
			if err != nil {
				return errorNotification(fmt.Sprintf("editor failed: %v", err))
			}
			return nil
		},
	)
}

func resumeSessionCmd(sessionID, cwd string) tea.Cmd {
	cmd, err := newResumeExecCmd(sessionID, cwd)
	if err != nil {
		return notificationCmd(resumeErrorNotification(err, cwd))
	}

	return tea.ExecProcess(
		cmd,
		func(err error) tea.Msg {
			if err != nil {
				return resumeErrorNotification(err, cwd)
			}
			return nil
		},
	)
}
