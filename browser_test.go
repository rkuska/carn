package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func testBrowser(t *testing.T) browserModel {
	t.Helper()
	b := newBrowserModel(context.Background(), t.TempDir())
	b.width = 120
	b.height = 40
	b.preview.SetWidth(40)
	return b
}

func testSession(id string) sessionFull {
	return sessionFull{
		meta: sessionMeta{
			id:        id,
			timestamp: time.Now(),
			project:   project{displayName: "test"},
		},
		messages: []message{
			{role: roleUser, text: "hello"},
			{role: roleAssistant, text: "hi there"},
		},
	}
}

func testConv(id string) conversation {
	return conversation{
		name:    "test-slug",
		project: project{dirName: "test", displayName: "test"},
		sessions: []sessionMeta{
			{id: id, slug: "test-slug", timestamp: time.Now(), project: project{displayName: "test"}},
		},
	}
}

func TestAddToCacheEvictsBothCaches(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)

	// Fill cache to capacity
	for i := range previewCacheSize {
		id := fmt.Sprintf("session-%d", i)
		b.previewCache[id] = "preview"
		b.sessionCache[id] = testSession(id)
		b.addToCache(id)
	}

	if len(b.cacheOrder) != previewCacheSize {
		t.Fatalf("cacheOrder len = %d, want %d", len(b.cacheOrder), previewCacheSize)
	}

	// Add one more to trigger eviction
	evictedID := "session-0"
	newID := "session-new"
	b.previewCache[newID] = "preview"
	b.sessionCache[newID] = testSession(newID)
	b.addToCache(newID)

	if _, ok := b.previewCache[evictedID]; ok {
		t.Error("expected evicted session removed from previewCache")
	}
	if _, ok := b.sessionCache[evictedID]; ok {
		t.Error("expected evicted session removed from sessionCache")
	}

	// New entry should exist
	if _, ok := b.previewCache[newID]; !ok {
		t.Error("expected new session in previewCache")
	}
	if _, ok := b.sessionCache[newID]; !ok {
		t.Error("expected new session in sessionCache")
	}
}

func TestCachedSessionReturnsHitAndMiss(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)

	session := testSession("cached-id")
	b.sessionCache["cached-id"] = session

	t.Run("hit", func(t *testing.T) {
		t.Parallel()
		got, ok := b.cachedSession("cached-id")
		if !ok {
			t.Fatal("expected cache hit")
		}
		if got.meta.id != "cached-id" {
			t.Errorf("got id = %q, want %q", got.meta.id, "cached-id")
		}
		if len(got.messages) != 2 {
			t.Errorf("got %d messages, want 2", len(got.messages))
		}
	})

	t.Run("miss", func(t *testing.T) {
		t.Parallel()
		_, ok := b.cachedSession("nonexistent")
		if ok {
			t.Error("expected cache miss")
		}
	})
}

func TestCheckPreviewUpdateUsesSessionCacheFallback(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)

	session := testSession("fallback-id")
	conv := testConv("fallback-id")
	b.sessionCache[conv.id()] = session

	// Set up the list with this conversation as selected item
	b.list.SetItems([]list.Item{conv})
	b.list.Select(0)

	// No preview cached — should fall back to session cache
	var cmds []tea.Cmd
	b.checkPreviewUpdate(&cmds)

	// Should NOT have issued a parseConversationCmd (no commands)
	if len(cmds) != 0 {
		t.Errorf("expected 0 cmds (session cache fallback), got %d", len(cmds))
	}

	// Preview cache should now be populated from the session cache
	if _, ok := b.previewCache[conv.id()]; !ok {
		t.Error("expected previewCache to be populated from session cache")
	}
}

func TestBrowserFooterShowsHelpAndStatus(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.mainConversationCount = 5

	footer := b.footerView()

	if !strings.Contains(footer, "open transcript") {
		t.Fatalf("expected footer to contain 'open transcript' help, got: %s", footer)
	}
	if !strings.Contains(footer, "deep search") {
		t.Fatalf("expected footer to contain 'deep search' help, got: %s", footer)
	}
	if !strings.Contains(footer, "5 sessions") {
		t.Fatalf("expected footer to contain '5 sessions', got: %s", footer)
	}
}

func TestBrowserFooterShowsDeepSearchIndicator(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.deepSearch = true

	footer := b.footerView()

	if !strings.Contains(footer, "DEEP SEARCH") {
		t.Fatalf("expected footer to show deep search indicator, got: %s", footer)
	}
}

func TestBrowserFooterUsesSeparateStatusRow(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.mainConversationCount = 5
	b.notification = errorNotification("resume failed: directory not found: /tmp/project").notification

	lines := strings.Split(b.footerView(), "\n")
	if len(lines) != 2 {
		t.Fatalf("footer line count = %d, want 2", len(lines))
	}

	helpLine := ansi.Strip(lines[0])
	statusLine := ansi.Strip(lines[1])

	if !strings.Contains(helpLine, "open transcript") {
		t.Fatalf("help line = %q, want help text", helpLine)
	}
	if strings.Contains(helpLine, "resume failed") {
		t.Fatalf("help line should not contain notification text: %q", helpLine)
	}
	if !strings.Contains(statusLine, "resume failed: directory not found") {
		t.Fatalf("status line = %q, want notification text", statusLine)
	}
}

func TestBrowserFooterReservesBlankStatusRow(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)

	lines := strings.Split(b.footerView(), "\n")
	if len(lines) != 2 {
		t.Fatalf("footer line count = %d, want 2", len(lines))
	}
	if strings.TrimSpace(ansi.Strip(lines[1])) != "" {
		t.Fatalf("status line = %q, want blank", ansi.Strip(lines[1]))
	}
}

func TestBrowserViewKeepsWindowHeightWithTwoLineFooter(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.updateLayout()

	if got := lipgloss.Height(b.View()); got != b.height {
		t.Fatalf("view height = %d, want %d", got, b.height)
	}
}

func TestBrowserUpdateShowsAndClearsNotifications(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)

	b, _ = b.Update(successNotification("transcript copied to clipboard"))
	if b.notification.text != "transcript copied to clipboard" {
		t.Fatalf("notification text = %q, want %q", b.notification.text, "transcript copied to clipboard")
	}
	if b.notification.kind != notificationSuccess {
		t.Fatalf("notification kind = %q, want %q", b.notification.kind, notificationSuccess)
	}

	b, _ = b.Update(clearNotificationMsg{})
	if b.notification.text != "" {
		t.Fatalf("notification text = %q, want empty", b.notification.text)
	}
}
