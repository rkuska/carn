package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilePath_XDGConfigHome(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got := FilePath()
	want := filepath.Join(xdg, "carn", "config.toml")
	if got != want {
		t.Errorf("FilePath() = %q, want %q", got, want)
	}
}

func TestFilePath_DefaultHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	got := FilePath()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "carn", "config.toml")
	if got != want {
		t.Errorf("FilePath() = %q, want %q", got, want)
	}
}

func TestFileExists_Missing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if FileExists() {
		t.Error("FileExists() = true, want false for missing file")
	}
}

func TestFileExists_Present(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	dir := filepath.Join(xdg, "carn")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("# empty"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !FileExists() {
		t.Error("FileExists() = false, want true for existing file")
	}
}

func TestFileExists_IsDir(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	// Create config.toml as a directory, not a file.
	dir := filepath.Join(xdg, "carn", "config.toml")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if FileExists() {
		t.Error("FileExists() = true, want false when path is a directory")
	}
}
