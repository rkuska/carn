package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

	// Detect terminal background before bubbletea takes over stdin.
	// glamour.WithAutoStyle() sends an OSC 11 query on every call, whose
	// response bytes get misinterpreted as KeyPressMsg by bubbletea's
	// input parser. Detecting once here avoids the issue entirely.
	glamourStyle := "dark"
	if !lipgloss.HasDarkBackground(os.Stdin, os.Stdout) {
		glamourStyle = "light"
	}

	model := newAppModel(ctx, cfg, glamourStyle)

	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("p.Run: %w", err)
	}

	return nil
}
