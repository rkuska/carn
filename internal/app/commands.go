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
	"github.com/rs/zerolog"
)

// Messages

type conversationsLoadedMsg struct {
	conversations       []conv.Conversation
	deepSearchAvailable bool
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

		_, deepSearchAvailable, err := store.DeepSearch(ctx, archiveDir, "", conversations)
		if err != nil {
			deepSearchAvailable = false
			zerolog.Ctx(ctx).Debug().Err(err).Msg("deep search unavailable during browser load")
		}

		return conversationsLoadedMsg{
			conversations:       conversations,
			deepSearchAvailable: deepSearchAvailable,
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
			zerolog.Ctx(ctx).Error().Err(err).Msgf("browserStore.Load failed for %s", conversation.ID())
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
	ctx context.Context,
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

func exportTranscriptCmd(session conv.Session, opts transcriptOptions, planExpanded bool) tea.Cmd {
	return func() tea.Msg {
		text := renderVisibleConversation(session, opts, planExpanded)
		return exportText(text, session.Meta)
	}
}

func exportText(text string, meta conv.SessionMeta) notificationMsg {
	name := fmt.Sprintf("claude-session-%s.md", meta.Slug)
	if meta.Slug == "" {
		name = fmt.Sprintf("claude-session-%s.md", meta.ID[:8])
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
