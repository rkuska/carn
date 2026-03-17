package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestLoadState(t *testing.T) {
	t.Run("missing file returns defaults and missing status", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		state, err := LoadState()
		require.NoError(t, err)
		assert.Equal(t, StatusMissing, state.Status)
		assert.NoError(t, state.Err)
		assertConfigMatchesDefaults(t, state.Config)

		baseDir, err := os.UserConfigDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(baseDir, "carn", "config.toml"), state.Path)
	})

	t.Run("valid file overrides defaults", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)

		writeConfigFile(t, `
[paths]
archive_dir = "/custom/archive"
claude_source_dir = "/custom/claude"
log_file = "/var/log/carn.log"

[display]
browser_cache_size = 50

[search]
deep_search_debounce_ms = 500
`)

		state, err := LoadState()
		require.NoError(t, err)
		assert.Equal(t, StatusLoaded, state.Status)
		assert.NoError(t, state.Err)
		assert.Equal(t, "/custom/archive", state.Config.Paths.ArchiveDir)
		assert.Equal(t, "/custom/claude", state.Config.Paths.ClaudeSourceDir)
		assert.Equal(t, "/var/log/carn.log", state.Config.Paths.LogFile)
		assert.Equal(t, 50, state.Config.Display.BrowserCacheSize)
		assert.Equal(t, 500, state.Config.Search.DeepSearchDebounceMs)
		assert.Equal(t, DefaultTimestampFormat, state.Config.Display.TimestampFormat)
	})

	t.Run("tilde expands in loaded config", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)

		writeConfigFile(t, `
[paths]
archive_dir = "~/my-archive"
log_file = "~/logs/carn.log"
`)

		state, err := LoadState()
		require.NoError(t, err)

		home, err := os.UserHomeDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, "my-archive"), state.Config.Paths.ArchiveDir)
		assert.Equal(t, filepath.Join(home, "logs", "carn.log"), state.Config.Paths.LogFile)
	})

	t.Run("malformed toml returns invalid status with defaults", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)

		writeConfigFile(t, "[invalid toml\n")

		state, err := LoadState()
		require.NoError(t, err)
		assert.Equal(t, StatusInvalid, state.Status)
		require.Error(t, state.Err)
		assert.ErrorContains(t, state.Err, "overlayFile")
		assertConfigMatchesDefaults(t, state.Config)
	})

	t.Run("validation failure returns invalid status with defaults", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)

		writeConfigFile(t, `
[display]
browser_cache_size = 0
`)

		state, err := LoadState()
		require.NoError(t, err)
		assert.Equal(t, StatusInvalid, state.Status)
		require.Error(t, state.Err)
		assert.ErrorContains(t, state.Err, "validate")
		assertConfigMatchesDefaults(t, state.Config)
	})

	t.Run("logging config overrides defaults", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)

		writeConfigFile(t, `
[logging]
level = "debug"
max_size_mb = 5
max_backups = 2
`)

		state, err := LoadState()
		require.NoError(t, err)
		assert.Equal(t, StatusLoaded, state.Status)
		assert.Equal(t, "debug", state.Config.Logging.Level)
		assert.Equal(t, 5, state.Config.Logging.MaxSizeMB)
		assert.Equal(t, 2, state.Config.Logging.MaxBackups)
	})

	t.Run("invalid log level returns invalid status", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)

		writeConfigFile(t, `
[logging]
level = "verbose"
`)

		state, err := LoadState()
		require.NoError(t, err)
		assert.Equal(t, StatusInvalid, state.Status)
		require.Error(t, state.Err)
		assert.ErrorContains(t, state.Err, "invalid log level")
	})

	t.Run("invalid max_size_mb returns invalid status", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)

		writeConfigFile(t, `
[logging]
max_size_mb = 0
`)

		state, err := LoadState()
		require.NoError(t, err)
		assert.Equal(t, StatusInvalid, state.Status)
		require.Error(t, state.Err)
		assert.ErrorContains(t, state.Err, "max_size_mb")
	})

	t.Run("invalid max_backups returns invalid status", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)

		writeConfigFile(t, `
[logging]
max_backups = -1
`)

		state, err := LoadState()
		require.NoError(t, err)
		assert.Equal(t, StatusInvalid, state.Status)
		require.Error(t, state.Err)
		assert.ErrorContains(t, state.Err, "max_backups")
	})
}

func TestLoad(t *testing.T) {
	t.Run("returns defaults when config is missing", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		cfg, err := Load()
		require.NoError(t, err)
		assertConfigMatchesDefaults(t, cfg)
	})

	t.Run("returns error when config is invalid", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)

		writeConfigFile(t, `
[search]
deep_search_debounce_ms = -1
`)

		_, err := Load()
		require.Error(t, err)
		assert.ErrorContains(t, err, "state.Err")
	})
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

func assertConfigMatchesDefaults(t *testing.T, cfg Config) {
	t.Helper()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(home, DefaultArchiveDir), cfg.Paths.ArchiveDir)
	assert.Equal(t, filepath.Join(home, DefaultClaudeSourceDir), cfg.Paths.ClaudeSourceDir)
	assert.Equal(t, filepath.Join(home, DefaultCodexSourceDir), cfg.Paths.CodexSourceDir)
	assert.Equal(t, filepath.Join(home, DefaultLogDir, DefaultLogFileName), cfg.Paths.LogFile)
	assert.Equal(t, DefaultTimestampFormat, cfg.Display.TimestampFormat)
	assert.Equal(t, DefaultBrowserCacheSize, cfg.Display.BrowserCacheSize)
	assert.Equal(t, DefaultDeepSearchDebounceMs, cfg.Search.DeepSearchDebounceMs)
	assert.Equal(t, DefaultLogLevel, cfg.Logging.Level)
	assert.Equal(t, DefaultMaxSizeMB, cfg.Logging.MaxSizeMB)
	assert.Equal(t, DefaultMaxBackups, cfg.Logging.MaxBackups)
}

func TestParseLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    int8
		wantErr bool
	}{
		{name: "empty defaults to info", input: "", want: 1},
		{name: "debug", input: "debug", want: 0},
		{name: "info", input: "info", want: 1},
		{name: "warn", input: "warn", want: 2},
		{name: "error", input: "error", want: 3},
		{name: "unknown returns error", input: "trace", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseLogLevel(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, int8(got))
		})
	}
}

func writeConfigFile(t *testing.T, tomlContent string) {
	t.Helper()

	path, err := ResolvePath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(tomlContent), 0o644))
}
