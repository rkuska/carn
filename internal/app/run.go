package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	arch "github.com/rkuska/carn/internal/archive"
	"github.com/rkuska/carn/internal/canonical"
	"github.com/rkuska/carn/internal/source/claude"
	"github.com/rkuska/carn/internal/source/codex"
	"github.com/rs/zerolog"
)

// Config defines the minimal app inputs needed to build the UI model.
type Config struct {
	SourceDir      string
	CodexSourceDir string
	ArchiveDir     string
	GlamourStyle   string
}

// Run starts the CLI application with environment-derived configuration.
func Run() error {
	logFile, err := os.OpenFile("/tmp/carn.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: logFile, NoColor: true}).With().Timestamp().Logger()
	ctx := logger.WithContext(context.Background())

	cfg, err := defaultArchiveConfig()
	if err != nil {
		return fmt.Errorf("defaultArchiveConfig: %w", err)
	}

	// Detect terminal background before Bubble Tea takes over stdin.
	// glamour.WithAutoStyle() sends an OSC 11 query on every call, whose
	// response bytes get misinterpreted as KeyPressMsg by Bubble Tea's
	// input parser. Detecting once here avoids the issue entirely.
	hasDarkBG := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	glamourStyle := "dark"
	if !hasDarkBG {
		glamourStyle = "light"
	}

	model, err := NewModel(ctx, Config{
		SourceDir:      cfg.SourceDir,
		CodexSourceDir: cfg.CodexSourceDir,
		ArchiveDir:     cfg.ArchiveDir,
		GlamourStyle:   glamourStyle,
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
	if cfg.SourceDir == "" && cfg.CodexSourceDir == "" {
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
		glamourStyle = "dark"
	}

	initPalette(glamourStyle != "light")

	claudeBackend := claude.New()
	codexBackend := codex.New()
	store := canonical.New(claudeBackend, codexBackend)
	browserStore := newBrowserStore(store)
	pipeline := newImportPipeline(
		arch.Config{
			SourceDir:      cfg.SourceDir,
			CodexSourceDir: cfg.CodexSourceDir,
			ArchiveDir:     cfg.ArchiveDir,
		},
		store,
		claudeBackend,
		codexBackend,
	)
	launcher := newSessionLauncher(claudeBackend, codexBackend)

	model := newAppModelWithDeps(
		ctx,
		arch.Config{
			SourceDir:      cfg.SourceDir,
			CodexSourceDir: cfg.CodexSourceDir,
			ArchiveDir:     cfg.ArchiveDir,
		},
		glamourStyle,
		browserStore,
		pipeline,
		launcher,
	)

	return model, nil
}

func defaultArchiveConfig() (arch.Config, error) {
	cfg, err := arch.DefaultConfig()
	if err != nil {
		return arch.Config{}, fmt.Errorf("archive.DefaultConfig: %w", err)
	}
	return cfg, nil
}
