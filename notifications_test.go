package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
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
			if got != tt.want {
				t.Errorf("notificationDuration(%q) = %v, want %v", tt.kind, got, tt.want)
			}
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
			if msg.notification.kind != notificationError {
				t.Fatalf("notification kind = %q, want %q", msg.notification.kind, notificationError)
			}
			if msg.notification.text != tt.want {
				t.Errorf("notification text = %q, want %q", msg.notification.text, tt.want)
			}
		})
	}
}

func TestNewResumeExecCmd(t *testing.T) {
	t.Parallel()

	t.Run("valid directory configures command", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		cmd, err := newResumeExecCmd("session-123", dir)
		if err != nil {
			t.Fatalf("newResumeExecCmd() error = %v", err)
		}

		if cmd.Dir != dir {
			t.Errorf("cmd.Dir = %q, want %q", cmd.Dir, dir)
		}

		wantArgs := []string{"claude", "--resume", "session-123"}
		if len(cmd.Args) != len(wantArgs) {
			t.Fatalf("len(cmd.Args) = %d, want %d", len(cmd.Args), len(wantArgs))
		}
		for i := range wantArgs {
			if cmd.Args[i] != wantArgs[i] {
				t.Errorf("cmd.Args[%d] = %q, want %q", i, cmd.Args[i], wantArgs[i])
			}
		}
	})

	t.Run("empty cwd fails", func(t *testing.T) {
		t.Parallel()

		_, err := newResumeExecCmd("session-123", "")
		if !errors.Is(err, errResumeDirEmpty) {
			t.Fatalf("newResumeExecCmd() error = %v, want %v", err, errResumeDirEmpty)
		}
	})

	t.Run("missing directory fails", func(t *testing.T) {
		t.Parallel()

		missingDir := filepath.Join(t.TempDir(), "missing")

		_, err := newResumeExecCmd("session-123", missingDir)
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("newResumeExecCmd() error = %v, want %v", err, os.ErrNotExist)
		}
	})

	t.Run("file path fails", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		filePath := filepath.Join(dir, "session.txt")
		if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}

		_, err := newResumeExecCmd("session-123", filePath)
		if !errors.Is(err, errResumeDirNotDirectory) {
			t.Fatalf("newResumeExecCmd() error = %v, want %v", err, errResumeDirNotDirectory)
		}
	})
}
