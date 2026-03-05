package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/rs/zerolog"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	logFile, err := os.OpenFile("/tmp/claude-search.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	logger := zerolog.New(logFile).With().Timestamp().Logger()
	ctx := logger.WithContext(context.Background())

	cfg, err := defaultArchiveConfig()
	if err != nil {
		return fmt.Errorf("defaultArchiveConfig: %w", err)
	}

	model := newAppModel(ctx, cfg)

	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("p.Run: %w", err)
	}

	return nil
}
