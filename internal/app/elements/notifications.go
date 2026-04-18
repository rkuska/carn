package elements

import (
	"errors"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	src "github.com/rkuska/carn/internal/source"
)

type NotificationKind string

const (
	NotificationInfo    NotificationKind = "info"
	NotificationSuccess NotificationKind = "success"
	NotificationError   NotificationKind = "error"
)

type Notification struct {
	Kind NotificationKind
	Text string
}

type NotificationMsg struct {
	Notification Notification
}

type ClearNotificationMsg struct{}

func NewNotification(kind NotificationKind, text string) NotificationMsg {
	return NotificationMsg{
		Notification: Notification{
			Kind: kind,
			Text: text,
		},
	}
}

func InfoNotification(text string) NotificationMsg {
	return NewNotification(NotificationInfo, text)
}

func SuccessNotification(text string) NotificationMsg {
	return NewNotification(NotificationSuccess, text)
}

func ErrorNotification(text string) NotificationMsg {
	return NewNotification(NotificationError, text)
}

func NotificationDuration(kind NotificationKind) time.Duration {
	switch kind {
	case NotificationError:
		return 5 * time.Second
	case NotificationSuccess, NotificationInfo:
		return 3 * time.Second
	default:
		return 3 * time.Second
	}
}

func ClearNotificationAfter(kind NotificationKind) tea.Cmd {
	return tea.Tick(NotificationDuration(kind), func(_ time.Time) tea.Msg {
		return ClearNotificationMsg{}
	})
}

func NotificationCmd(msg NotificationMsg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func (t *Theme) RenderNotification(n Notification) string {
	if n.Text == "" {
		return ""
	}

	style := lipgloss.NewStyle().Foreground(t.ColorSecondary)

	switch n.Kind {
	case NotificationError:
		style = lipgloss.NewStyle().Foreground(t.ColorDiffRemove)
	case NotificationSuccess:
		style = lipgloss.NewStyle().Foreground(t.ColorAccent)
	case NotificationInfo:
		style = lipgloss.NewStyle().Foreground(t.ColorSecondary)
	}

	return style.Render(n.Text)
}

func FormatResumeErrorNotification(err error, cwd string, providerUnavailable error) NotificationMsg {
	switch {
	case errors.Is(err, src.ErrResumeDirEmpty):
		return ErrorNotification("resume failed: session working directory is unavailable")
	case errors.Is(err, src.ErrResumeTargetIDEmpty):
		return ErrorNotification("resume failed: session id is unavailable")
	case errors.Is(err, os.ErrNotExist):
		return ErrorNotification(fmt.Sprintf("resume failed: directory not found: %s", cwd))
	case errors.Is(err, src.ErrResumeDirNotDir):
		return ErrorNotification(fmt.Sprintf("resume failed: not a directory: %s", cwd))
	case providerUnavailable != nil && errors.Is(err, providerUnavailable):
		return ErrorNotification("resume failed: provider is unavailable")
	default:
		return ErrorNotification(fmt.Sprintf("resume failed: %v", err))
	}
}
