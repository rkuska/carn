package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	arch "github.com/rkuska/carn/internal/archive"
	conv "github.com/rkuska/carn/internal/conversation"
)

const (
	DefaultArchiveDir           = ".local/share/carn"
	DefaultClaudeSourceDir      = ".claude/projects"
	DefaultCodexSourceDir       = ".codex/sessions"
	DefaultLogFile              = "/tmp/carn.log"
	DefaultTimestampFormat      = "2006-01-02 15:04"
	DefaultBrowserCacheSize     = 20
	DefaultDeepSearchDebounceMs = 200
)

// Config is the fully resolved application configuration.
// All fields have valid values after Load returns successfully.
type Config struct {
	Paths   PathsConfig
	Display DisplayConfig
	Search  SearchConfig
}

type PathsConfig struct {
	ArchiveDir      string
	ClaudeSourceDir string
	CodexSourceDir  string
	LogFile         string
}

type DisplayConfig struct {
	TimestampFormat  string
	BrowserCacheSize int
}

type SearchConfig struct {
	DeepSearchDebounceMs int
}

// ArchiveConfig derives an archive.Config from the resolved configuration.
func (c Config) ArchiveConfig() arch.Config {
	sourceDirs := make(map[conv.Provider]string, 2)
	if c.Paths.ClaudeSourceDir != "" {
		sourceDirs[conv.ProviderClaude] = c.Paths.ClaudeSourceDir
	}
	if c.Paths.CodexSourceDir != "" {
		sourceDirs[conv.ProviderCodex] = c.Paths.CodexSourceDir
	}
	return arch.Config{
		SourceDirs: sourceDirs,
		ArchiveDir: c.Paths.ArchiveDir,
	}
}

// Load resolves configuration from defaults and config file.
// Precedence: config file > defaults.
// If the config file does not exist, defaults are used silently.
// If the config file exists but is malformed, an error is returned.
func Load() (Config, error) {
	home, _ := os.UserHomeDir()
	cfg := defaults(home)

	path := FilePath()
	if path != "" {
		if err := overlayFile(&cfg, path); err != nil {
			return Config{}, fmt.Errorf("overlayFile: %w", err)
		}
	}

	expandPaths(&cfg, home)

	if err := validate(cfg); err != nil {
		return Config{}, fmt.Errorf("validate: %w", err)
	}

	return cfg, nil
}

func defaults(home string) Config {
	return Config{
		Paths: PathsConfig{
			ArchiveDir:      filepath.Join(home, DefaultArchiveDir),
			ClaudeSourceDir: filepath.Join(home, DefaultClaudeSourceDir),
			CodexSourceDir:  filepath.Join(home, DefaultCodexSourceDir),
			LogFile:         DefaultLogFile,
		},
		Display: DisplayConfig{
			TimestampFormat:  DefaultTimestampFormat,
			BrowserCacheSize: DefaultBrowserCacheSize,
		},
		Search: SearchConfig{
			DeepSearchDebounceMs: DefaultDeepSearchDebounceMs,
		},
	}
}

// rawConfig is the TOML deserialization target. Pointer fields distinguish
// "not set" from "set to zero value".
type rawConfig struct {
	Paths   *rawPaths   `toml:"paths"`
	Display *rawDisplay `toml:"display"`
	Search  *rawSearch  `toml:"search"`
}

type rawPaths struct {
	ArchiveDir      *string `toml:"archive_dir"`
	ClaudeSourceDir *string `toml:"claude_source_dir"`
	CodexSourceDir  *string `toml:"codex_source_dir"`
	LogFile         *string `toml:"log_file"`
}

type rawDisplay struct {
	TimestampFormat  *string `toml:"timestamp_format"`
	BrowserCacheSize *int    `toml:"browser_cache_size"`
}

type rawSearch struct {
	DeepSearchDebounceMs *int `toml:"deep_search_debounce_ms"`
}

func overlayFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("os.ReadFile: %w", err)
	}

	var raw rawConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("toml.Unmarshal: %w", err)
	}

	overlayRawPaths(&cfg.Paths, raw.Paths)
	overlayRawDisplay(&cfg.Display, raw.Display)
	overlayRawSearch(&cfg.Search, raw.Search)
	return nil
}

func overlayRawPaths(dst *PathsConfig, src *rawPaths) {
	if src == nil {
		return
	}
	if src.ArchiveDir != nil {
		dst.ArchiveDir = *src.ArchiveDir
	}
	if src.ClaudeSourceDir != nil {
		dst.ClaudeSourceDir = *src.ClaudeSourceDir
	}
	if src.CodexSourceDir != nil {
		dst.CodexSourceDir = *src.CodexSourceDir
	}
	if src.LogFile != nil {
		dst.LogFile = *src.LogFile
	}
}

func overlayRawDisplay(dst *DisplayConfig, src *rawDisplay) {
	if src == nil {
		return
	}
	if src.TimestampFormat != nil {
		dst.TimestampFormat = *src.TimestampFormat
	}
	if src.BrowserCacheSize != nil {
		dst.BrowserCacheSize = *src.BrowserCacheSize
	}
}

func overlayRawSearch(dst *SearchConfig, src *rawSearch) {
	if src == nil {
		return
	}
	if src.DeepSearchDebounceMs != nil {
		dst.DeepSearchDebounceMs = *src.DeepSearchDebounceMs
	}
}

func expandPaths(cfg *Config, home string) {
	if home == "" {
		return
	}
	cfg.Paths.ArchiveDir = expandTilde(cfg.Paths.ArchiveDir, home)
	cfg.Paths.ClaudeSourceDir = expandTilde(cfg.Paths.ClaudeSourceDir, home)
	cfg.Paths.CodexSourceDir = expandTilde(cfg.Paths.CodexSourceDir, home)
	cfg.Paths.LogFile = expandTilde(cfg.Paths.LogFile, home)
}

func expandTilde(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

func validate(cfg Config) error {
	// Validate timestamp format by attempting a format operation.
	ref := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
	result := ref.Format(cfg.Display.TimestampFormat)
	if result == "" {
		return fmt.Errorf("invalid timestamp_format: %q", cfg.Display.TimestampFormat)
	}
	if cfg.Display.BrowserCacheSize < 1 {
		return fmt.Errorf("browser_cache_size must be >= 1, got %d", cfg.Display.BrowserCacheSize)
	}
	if cfg.Search.DeepSearchDebounceMs < 0 {
		return fmt.Errorf("deep_search_debounce_ms must be >= 0, got %d", cfg.Search.DeepSearchDebounceMs)
	}
	return nil
}
