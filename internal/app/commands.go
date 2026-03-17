package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"

	conv "github.com/rkuska/carn/internal/conversation"
)

// Messages

type conversationsLoadedMsg struct {
	conversations []conv.Conversation
}

type sessionsLoadErrorMsg struct {
	err error
}

type openViewerMsg struct {
	conversationID string
	conversation   conv.Conversation
	session        conv.Session
}

// Commands

func loadSessionsCmdWithStore(
	ctx context.Context,
	archiveDir string,
	store browserStore,
) tea.Cmd {
	return func() tea.Msg {
		conversations, err := store.List(ctx, archiveDir)
		if err != nil {
			return sessionsLoadErrorMsg{err: err}
		}

		// Sort by timestamp descending (newest first)
		sort.Slice(conversations, func(i, j int) bool {
			return conversations[i].Timestamp().After(conversations[j].Timestamp())
		})

		return conversationsLoadedMsg{
			conversations: conversations,
		}
	}
}

func openConversationCmdWithStore(
	ctx context.Context,
	archiveDir string,
	conversation conv.Conversation,
	store browserStore,
) tea.Cmd {
	return func() tea.Msg {
		session, err := store.Load(ctx, archiveDir, conversation)
		if err != nil {
			return errorNotification(fmt.Sprintf("load session failed: %v", err))
		}
		return openViewerMsg{
			conversationID: conversation.CacheKey(),
			conversation:   conversation,
			session:        session,
		}
	}
}

func openConversationCmdCachedWithStore(
	conversation conv.Conversation,
	parent conv.Session,
) tea.Cmd {
	return func() tea.Msg {
		return openViewerMsg{
			conversationID: conversation.CacheKey(),
			conversation:   conversation,
			session:        parent,
		}
	}
}

func exportTranscriptCmd(
	conversation conv.Conversation,
	session conv.Session,
	opts transcriptOptions,
	planExpanded bool,
) tea.Cmd {
	return func() tea.Msg {
		text := renderVisibleConversation(session, opts, planExpanded)
		return exportText(text, conversation, session.Meta)
	}
}

func exportText(text string, conversation conv.Conversation, meta conv.SessionMeta) notificationMsg {
	name := conversationExportFileName(conversation, meta)

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
	return runEditorCmd(newEditorCmd(filePath))
}

func newEditorCmd(filePath string) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	return exec.Command(editor, filePath)
}

func runEditorCmd(cmd *exec.Cmd) tea.Cmd {
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

func copyTextCmd(text string, successMessage string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(text); err != nil {
			return errorNotification(fmt.Sprintf("copy failed: %v", err))
		}
		return successNotification(successMessage)
	}
}

func resumeSessionCmd(target conv.ResumeTarget, launcher sessionLauncher) tea.Cmd {
	cmd, err := launcher.ResumeCommand(target)
	if err != nil {
		return notificationCmd(resumeErrorNotification(err, target.CWD))
	}

	return tea.ExecProcess(
		cmd,
		func(err error) tea.Msg {
			if err != nil {
				return resumeErrorNotification(err, target.CWD)
			}
			return nil
		},
	)
}
