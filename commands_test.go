package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportTextReturnsSuccessNotification(t *testing.T) {
	homeDir := t.TempDir()
	desktopDir := filepath.Join(homeDir, "Desktop")
	if err := os.Mkdir(desktopDir, 0o755); err != nil {
		t.Fatalf("os.Mkdir() error = %v", err)
	}
	t.Setenv("HOME", homeDir)

	msg := exportText("hello export", sessionMeta{
		id:   "session-12345678",
		slug: "demo-session",
	})

	if msg.notification.kind != notificationSuccess {
		t.Fatalf("notification kind = %q, want %q", msg.notification.kind, notificationSuccess)
	}
	if !strings.Contains(msg.notification.text, "exported to ") {
		t.Fatalf("notification text = %q, want success message", msg.notification.text)
	}

	outPath := filepath.Join(desktopDir, "claude-session-demo-session.md")
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(content) != "hello export" {
		t.Fatalf("file content = %q, want %q", string(content), "hello export")
	}
}

func TestResumeSessionCmdReturnsErrorNotificationForInvalidCWD(t *testing.T) {
	t.Parallel()

	cmd := resumeSessionCmd("session-123", "")
	msg := cmd()

	notification, ok := msg.(notificationMsg)
	if !ok {
		t.Fatalf("message type = %T, want notificationMsg", msg)
	}
	if notification.notification.kind != notificationError {
		t.Fatalf("notification kind = %q, want %q", notification.notification.kind, notificationError)
	}
	if notification.notification.text != "resume failed: session working directory is unavailable" {
		t.Fatalf("notification text = %q", notification.notification.text)
	}
}
