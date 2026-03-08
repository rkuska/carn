package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

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
			err:  errResumeDirEmpty,
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
			err:  errResumeDirNotDirectory,
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

	t.Run("valid directory configures command", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		cmd, err := newResumeExecCmd("session-123", dir)
		require.NoError(t, err)
		assert.Equal(t, dir, cmd.Dir)

		wantArgs := []string{"claude", "--resume", "session-123"}
		require.Len(t, cmd.Args, len(wantArgs))
		assert.Equal(t, wantArgs, cmd.Args)
	})

	t.Run("empty cwd fails", func(t *testing.T) {
		t.Parallel()

		_, err := newResumeExecCmd("session-123", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, errResumeDirEmpty)
	})

	t.Run("missing directory fails", func(t *testing.T) {
		t.Parallel()

		missingDir := filepath.Join(t.TempDir(), "missing")

		_, err := newResumeExecCmd("session-123", missingDir)
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("file path fails", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		filePath := filepath.Join(dir, "session.txt")
		require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o644))

		_, err := newResumeExecCmd("session-123", filePath)
		require.Error(t, err)
		assert.ErrorIs(t, err, errResumeDirNotDirectory)
	})
}
