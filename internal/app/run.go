package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	glamourStyleDark  = "dark"
	glamourStyleLight = "light"
)

// Config defines the minimal app inputs needed to build the UI model.
type Config struct {
	SourceDirs           map[conv.Provider]string
	ArchiveDir           string
	LogFile              string
	GlamourStyle         string
	TimestampFormat      string
	BrowserCacheSize     int
	DeepSearchDebounceMs int
	ConfigFilePath       string
	ConfigStatus         config.Status
	ConfigErr            error
}

// Run starts the CLI application with config-file-derived configuration.
func Run() error {
	state, err := config.LoadState()
	if err != nil {
		return fmt.Errorf("config.LoadState: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(state.Config.Paths.LogFile), 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	level, err := config.ParseLogLevel(state.Config.Logging.Level)
	if err != nil {
		return fmt.Errorf("config.ParseLogLevel: %w", err)
	}
	if os.Getenv("DEBUG") == "1" {
		level = zerolog.DebugLevel
	}

	logWriter := &lumberjack.Logger{
		Filename:   state.Config.Paths.LogFile,
		MaxSize:    state.Config.Logging.MaxSizeMB,
		MaxBackups: state.Config.Logging.MaxBackups,
		Compress:   true,
	}
	defer func() { _ = logWriter.Close() }()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: logWriter, NoColor: true}).
		With().Timestamp().Logger().Level(level)
	ctx := logger.WithContext(context.Background())

	logger.Info().
		Str("log_file", state.Config.Paths.LogFile).
		Str("log_level", level.String()).
		Str("archive_dir", state.Config.Paths.ArchiveDir).
		Msg("carn started")

	// Detect terminal background before Bubble Tea takes over stdin.
	// glamour.WithAutoStyle() sends an OSC 11 query on every call, whose
	// response bytes get misinterpreted as KeyPressMsg by Bubble Tea's
	// input parser. Detecting once here avoids the issue entirely.
	hasDarkBG := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	glamourStyle := glamourStyleDark
	if !hasDarkBG {
		glamourStyle = glamourStyleLight
	}

	model, err := NewModel(ctx, configStateToAppConfig(state, glamourStyle))
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
	model.pipelineFactory = func(nextCfg arch.Config) importPipeline {
		return newImportPipeline(nextCfg, store, claudeBackend, codexBackend)
	}

	return model, nil
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

func resolveRuntimeConfig(cfg Config) Config {
	if cfg.ConfigStatus != "" {
		return cfg
	}

	cfg.ConfigStatus = config.StatusMissing
	return cfg
}
