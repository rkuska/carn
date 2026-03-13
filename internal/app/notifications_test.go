package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
	"github.com/rkuska/carn/internal/source/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind notificationKind
		want time.Duration
	}{
		{
			name: "error lasts longer",
			kind: notificationError,
			want: 5 * time.Second,
		},
		{
			name: "success uses default duration",
			kind: notificationSuccess,
			want: 3 * time.Second,
		},
		{
			name: "info uses default duration",
			kind: notificationInfo,
			want: 3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := notificationDuration(tt.kind)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResumeErrorNotification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		cwd  string
		want string
	}{
		{
			name: "empty cwd",
			err:  src.ErrResumeDirEmpty,
			want: "resume failed: session working directory is unavailable",
		},
		{
			name: "missing directory",
			err:  os.ErrNotExist,
			cwd:  "/tmp/missing",
			want: "resume failed: directory not found: /tmp/missing",
		},
		{
			name: "path is not directory",
			err:  src.ErrResumeDirNotDir,
			cwd:  "/tmp/file.txt",
			want: "resume failed: not a directory: /tmp/file.txt",
		},
		{
			name: "launch error",
			err:  errors.New("boom"),
			want: "resume failed: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := resumeErrorNotification(tt.err, tt.cwd)
			assert.Equal(t, notificationError, msg.notification.kind)
			assert.Equal(t, tt.want, msg.notification.text)
		})
	}
}

func TestNewResumeExecCmd(t *testing.T) {
	t.Parallel()

	launcher := newSessionLauncher(claude.New())

	t.Run("valid directory configures command", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		cmd, err := launcher.ResumeCommand(conv.ResumeTarget{
			Provider: conv.ProviderClaude,
			ID:       "session-123",
			CWD:      dir,
		})
		require.NoError(t, err)
		assert.Equal(t, dir, cmd.Dir)

		wantArgs := []string{"claude", "--resume", "session-123"}
		require.Len(t, cmd.Args, len(wantArgs))
		assert.Equal(t, wantArgs, cmd.Args)
	})

	t.Run("empty cwd fails", func(t *testing.T) {
		t.Parallel()

		_, err := launcher.ResumeCommand(conv.ResumeTarget{
			Provider: conv.ProviderClaude,
			ID:       "session-123",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, src.ErrResumeDirEmpty)
	})

	t.Run("missing directory fails", func(t *testing.T) {
		t.Parallel()

		missingDir := filepath.Join(t.TempDir(), "missing")

		_, err := launcher.ResumeCommand(conv.ResumeTarget{
			Provider: conv.ProviderClaude,
			ID:       "session-123",
			CWD:      missingDir,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("file path fails", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		filePath := filepath.Join(dir, "session.txt")
		require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o644))

		_, err := launcher.ResumeCommand(conv.ResumeTarget{
			Provider: conv.ProviderClaude,
			ID:       "session-123",
			CWD:      filePath,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, src.ErrResumeDirNotDir)
	})
}
