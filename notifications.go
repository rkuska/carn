package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type notificationKind string

const (
	notificationInfo    notificationKind = "info"
	notificationSuccess notificationKind = "success"
	notificationError   notificationKind = "error"
)

type notification struct {
	kind notificationKind
	text string
}

type notificationMsg struct {
	notification notification
}

type clearNotificationMsg struct{}

var (
	errResumeDirEmpty        = errors.New("resume directory is empty")
	errResumeDirNotDirectory = errors.New("resume directory is not a directory")
)

func newNotification(kind notificationKind, text string) notificationMsg {
	return notificationMsg{
		notification: notification{
			kind: kind,
			text: text,
		},
	}
}

func infoNotification(text string) notificationMsg {
	return newNotification(notificationInfo, text)
}

func successNotification(text string) notificationMsg {
	return newNotification(notificationSuccess, text)
}

func errorNotification(text string) notificationMsg {
	return newNotification(notificationError, text)
}

func notificationDuration(kind notificationKind) time.Duration {
	switch kind {
	case notificationError:
		return 5 * time.Second
	case notificationSuccess, notificationInfo:
		return 3 * time.Second
	default:
		return 3 * time.Second
	}
}

func clearNotificationAfter(kind notificationKind) tea.Cmd {
	return tea.Tick(notificationDuration(kind), func(_ time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
}

func notificationCmd(msg notificationMsg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func renderNotification(n notification) string {
	if n.text == "" {
		return ""
	}

	style := lipgloss.NewStyle().Foreground(colorSecondary)

	switch n.kind {
	case notificationError:
		style = lipgloss.NewStyle().Foreground(colorDiffRemove)
	case notificationSuccess:
		style = lipgloss.NewStyle().Foreground(colorAccent)
	case notificationInfo:
		style = lipgloss.NewStyle().Foreground(colorSecondary)
	}

	return style.Render(n.text)
}

func newResumeExecCmd(sessionID, cwd string) (*exec.Cmd, error) {
	if cwd == "" {
		return nil, fmt.Errorf("newResumeExecCmd: %w", errResumeDirEmpty)
	}

	info, err := os.Stat(cwd)
	if err != nil {
		return nil, fmt.Errorf("newResumeExecCmd_osStat: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("newResumeExecCmd: %w", errResumeDirNotDirectory)
	}

	cmd := exec.Command("claude", "--resume", sessionID)
	cmd.Dir = cwd
	return cmd, nil
}

func resumeErrorNotification(err error, cwd string) notificationMsg {
	switch {
	case errors.Is(err, errResumeDirEmpty):
		return errorNotification("resume failed: session working directory is unavailable")
	case errors.Is(err, os.ErrNotExist):
		return errorNotification(fmt.Sprintf("resume failed: directory not found: %s", cwd))
	case errors.Is(err, errResumeDirNotDirectory):
		return errorNotification(fmt.Sprintf("resume failed: not a directory: %s", cwd))
	default:
		return errorNotification(fmt.Sprintf("resume failed: %v", err))
	}
}
