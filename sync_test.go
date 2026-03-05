package main

import (
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSyncModelInit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newSyncModel(cfg)

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() should return a batch command")
	}
}

func TestSyncModelProgressUpdates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newSyncModel(cfg)
	m.width = 120
	m.height = 40

	// Simulate files scanned
	files := []string{
		filepath.Join(dir, "source", "a.jsonl"),
		filepath.Join(dir, "source", "b.jsonl"),
		filepath.Join(dir, "source", "c.jsonl"),
	}
	m, _ = m.Update(syncFilesScannedMsg{files: files})

	if m.total != 3 {
		t.Errorf("total = %d, want 3", m.total)
	}
	if m.scanned != true {
		t.Error("expected scanned = true")
	}

	// Simulate file copied
	m, _ = m.Update(syncFileCopiedMsg{file: files[0]})
	if m.current != 1 {
		t.Errorf("current = %d, want 1", m.current)
	}
	if m.done {
		t.Error("should not be done after 1/3 files")
	}

	// View should contain progress info
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestSyncModelCompletion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newSyncModel(cfg)
	m.width = 120
	m.height = 40

	// Simulate single file scanned and copied
	m, _ = m.Update(syncFilesScannedMsg{files: []string{filepath.Join(dir, "source", "a.jsonl")}})
	m, _ = m.Update(syncFileCopiedMsg{file: filepath.Join(dir, "source", "a.jsonl")})

	if !m.done {
		t.Error("expected done = true after all files copied")
	}
	if m.result.copied != 1 {
		t.Errorf("copied = %d, want 1", m.result.copied)
	}
}

func TestSyncModelEmptySource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newSyncModel(cfg)
	m.width = 120
	m.height = 40

	// No files to sync
	m, _ = m.Update(syncFilesScannedMsg{files: nil})

	if !m.done {
		t.Error("expected done = true when no files to sync")
	}
}

func TestSyncModelSpinnerTick(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newSyncModel(cfg)
	m.width = 120
	m.height = 40

	// Spinner tick should not panic and should return next tick
	m, _ = m.Update(spinner.TickMsg{})
	_ = m.View()
}

func TestSyncModelWindowResize(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := archiveConfig{
		sourceDir:  filepath.Join(dir, "source"),
		archiveDir: filepath.Join(dir, "archive"),
	}
	m := newSyncModel(cfg)

	m, _ = m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})

	if m.width != 200 {
		t.Errorf("width = %d, want 200", m.width)
	}
	if m.height != 50 {
		t.Errorf("height = %d, want 50", m.height)
	}
}
