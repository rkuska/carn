package config

import (
	"os"
	"path/filepath"
	"testing"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestLoad_Defaults(t *testing.T) {
	// Point XDG to a temp dir so no real config file is found.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	home, _ := os.UserHomeDir()

	if got, want := cfg.Paths.ArchiveDir, filepath.Join(home, DefaultArchiveDir); got != want {
		t.Errorf("ArchiveDir = %q, want %q", got, want)
	}
	if got, want := cfg.Paths.ClaudeSourceDir, filepath.Join(home, DefaultClaudeSourceDir); got != want {
		t.Errorf("ClaudeSourceDir = %q, want %q", got, want)
	}
	if got, want := cfg.Paths.CodexSourceDir, filepath.Join(home, DefaultCodexSourceDir); got != want {
		t.Errorf("CodexSourceDir = %q, want %q", got, want)
	}
	if got, want := cfg.Paths.LogFile, DefaultLogFile; got != want {
		t.Errorf("LogFile = %q, want %q", got, want)
	}
	if got, want := cfg.Display.TimestampFormat, DefaultTimestampFormat; got != want {
		t.Errorf("TimestampFormat = %q, want %q", got, want)
	}
	if got, want := cfg.Display.BrowserCacheSize, DefaultBrowserCacheSize; got != want {
		t.Errorf("BrowserCacheSize = %d, want %d", got, want)
	}
	if got, want := cfg.Search.DeepSearchDebounceMs, DefaultDeepSearchDebounceMs; got != want {
		t.Errorf("DeepSearchDebounceMs = %d, want %d", got, want)
	}
}

func TestLoad_FileOverridesDefaults(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	dir := filepath.Join(xdg, "carn")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[paths]
archive_dir = "/custom/archive"
log_file = "/var/log/carn.log"

[display]
browser_cache_size = 50

[search]
deep_search_debounce_ms = 500
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if got := cfg.Paths.ArchiveDir; got != "/custom/archive" {
		t.Errorf("ArchiveDir = %q, want /custom/archive", got)
	}
	if got := cfg.Paths.LogFile; got != "/var/log/carn.log" {
		t.Errorf("LogFile = %q, want /var/log/carn.log", got)
	}
	if got := cfg.Display.BrowserCacheSize; got != 50 {
		t.Errorf("BrowserCacheSize = %d, want 50", got)
	}
	if got := cfg.Search.DeepSearchDebounceMs; got != 500 {
		t.Errorf("DeepSearchDebounceMs = %d, want 500", got)
	}
	// Unset fields should keep defaults.
	if got := cfg.Display.TimestampFormat; got != DefaultTimestampFormat {
		t.Errorf("TimestampFormat = %q, want %q (default)", got, DefaultTimestampFormat)
	}
}

func TestLoad_PartialPathsOverride(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	dir := filepath.Join(xdg, "carn")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[paths]
claude_source_dir = "/custom/claude"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if got := cfg.Paths.ClaudeSourceDir; got != "/custom/claude" {
		t.Errorf("ClaudeSourceDir = %q, want /custom/claude", got)
	}
	// Other paths should keep defaults.
	home, _ := os.UserHomeDir()
	if got, want := cfg.Paths.CodexSourceDir, filepath.Join(home, DefaultCodexSourceDir); got != want {
		t.Errorf("CodexSourceDir = %q, want %q (default)", got, want)
	}
}

func TestLoad_TildeExpansion(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	dir := filepath.Join(xdg, "carn")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[paths]
archive_dir = "~/my-archive"
log_file = "~/logs/carn.log"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	home, _ := os.UserHomeDir()
	if got, want := cfg.Paths.ArchiveDir, filepath.Join(home, "my-archive"); got != want {
		t.Errorf("ArchiveDir = %q, want %q (tilde expanded)", got, want)
	}
	if got, want := cfg.Paths.LogFile, filepath.Join(home, "logs", "carn.log"); got != want {
		t.Errorf("LogFile = %q, want %q (tilde expanded)", got, want)
	}
}

func TestLoad_MalformedTOML(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	dir := filepath.Join(xdg, "carn")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[invalid toml\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should return error for malformed TOML")
	}
}

func TestLoad_InvalidBrowserCacheSize(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	dir := filepath.Join(xdg, "carn")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[display]
browser_cache_size = 0
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should return error for browser_cache_size = 0")
	}
}

func TestLoad_InvalidDebounceMs(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	dir := filepath.Join(xdg, "carn")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[search]
deep_search_debounce_ms = -1
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should return error for negative debounce")
	}
}

func TestArchiveConfig(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Paths: PathsConfig{
			ArchiveDir:      "/archive",
			ClaudeSourceDir: "/claude",
			CodexSourceDir:  "/codex",
		},
	}

	ac := cfg.ArchiveConfig()

	if got := ac.ArchiveDir; got != "/archive" {
		t.Errorf("ArchiveDir = %q, want /archive", got)
	}
	if got := ac.SourceDirs[conv.ProviderClaude]; got != "/claude" {
		t.Errorf("SourceDirs[claude] = %q, want /claude", got)
	}
	if got := ac.SourceDirs[conv.ProviderCodex]; got != "/codex" {
		t.Errorf("SourceDirs[codex] = %q, want /codex", got)
	}
}

func TestArchiveConfig_EmptySourcesOmitted(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Paths: PathsConfig{
			ArchiveDir:      "/archive",
			ClaudeSourceDir: "/claude",
			CodexSourceDir:  "",
		},
	}

	ac := cfg.ArchiveConfig()

	if _, ok := ac.SourceDirs[conv.ProviderCodex]; ok {
		t.Error("empty CodexSourceDir should be omitted from SourceDirs")
	}
}

func TestExpandTilde(t *testing.T) {
	t.Parallel()

	home := "/home/user"
	tests := []struct {
		input string
		want  string
	}{
		{"~/projects", filepath.Join(home, "projects")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~other/path", "~other/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := expandTilde(tt.input, home)
			if got != tt.want {
				t.Errorf("expandTilde(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
