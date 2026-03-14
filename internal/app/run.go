package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	arch "github.com/rkuska/carn/internal/archive"
	"github.com/rkuska/carn/internal/canonical"
	"github.com/rkuska/carn/internal/config"
	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/internal/source/claude"
	"github.com/rkuska/carn/internal/source/codex"
	"github.com/rs/zerolog"
)

const (
	glamourStyleDark  = "dark"
	glamourStyleLight = "light"
)

// Config defines the minimal app inputs needed to build the UI model.
type Config struct {
	SourceDirs           map[conv.Provider]string
	ArchiveDir           string
	GlamourStyle         string
	TimestampFormat      string
	BrowserCacheSize     int
	DeepSearchDebounceMs int
	ConfigFilePath       string
	ConfigFileExists     bool
}

// Run starts the CLI application with config-file-derived configuration.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}

	logFile, err := os.OpenFile(cfg.Paths.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: logFile, NoColor: true}).With().Timestamp().Logger()
	ctx := logger.WithContext(context.Background())

	// Detect terminal background before Bubble Tea takes over stdin.
	// glamour.WithAutoStyle() sends an OSC 11 query on every call, whose
	// response bytes get misinterpreted as KeyPressMsg by Bubble Tea's
	// input parser. Detecting once here avoids the issue entirely.
	hasDarkBG := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	glamourStyle := glamourStyleDark
	if !hasDarkBG {
		glamourStyle = glamourStyleLight
	}

	archCfg := cfg.ArchiveConfig()
	cfgPath := config.FilePath()
	model, err := NewModel(ctx, Config{
		SourceDirs:           archCfg.SourceDirs,
		ArchiveDir:           archCfg.ArchiveDir,
		GlamourStyle:         glamourStyle,
		TimestampFormat:      cfg.Display.TimestampFormat,
		BrowserCacheSize:     cfg.Display.BrowserCacheSize,
		DeepSearchDebounceMs: cfg.Search.DeepSearchDebounceMs,
		ConfigFilePath:       cfgPath,
		ConfigFileExists:     fileExists(cfgPath),
	})
	if err != nil {
		return fmt.Errorf("NewModel: %w", err)
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("p.Run: %w", err)
	}

	return nil
}

// NewModel builds the root model with deterministic inputs for callers.
func NewModel(ctx context.Context, cfg Config) (tea.Model, error) {
	if len(nonEmptySourceProviders(cfg.SourceDirs)) == 0 {
		return nil, errors.New("newModel: at least one source dir is required")
	}
	if cfg.ArchiveDir == "" {
		return nil, errors.New("newModel: archive dir is required")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	glamourStyle := cfg.GlamourStyle
	if glamourStyle == "" {
		glamourStyle = glamourStyleDark
	}

	initPalette(glamourStyle != glamourStyleLight)

	claudeBackend := claude.New()
	codexBackend := codex.New()
	store := canonical.New(claudeBackend, codexBackend)
	browserStore := newBrowserStore(store)
	pipeline := newImportPipeline(
		arch.Config{
			SourceDirs: cfg.SourceDirs,
			ArchiveDir: cfg.ArchiveDir,
		},
		store,
		claudeBackend,
		codexBackend,
	)
	launcher := newSessionLauncher(claudeBackend, codexBackend)

	model := newAppModelWithDeps(
		ctx,
		arch.Config{
			SourceDirs: cfg.SourceDirs,
			ArchiveDir: cfg.ArchiveDir,
		},
		cfg,
		browserStore,
		pipeline,
		launcher,
	)

	return model, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func nonEmptySourceProviders(sourceDirs map[conv.Provider]string) []conv.Provider {
	providers := make([]conv.Provider, 0, len(sourceDirs))
	for provider, sourceDir := range sourceDirs {
		if sourceDir == "" {
			continue
		}
		providers = append(providers, provider)
	}
	slices.Sort(providers)
	return providers
}
