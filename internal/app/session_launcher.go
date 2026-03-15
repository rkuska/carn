package app

import (
	"errors"
	"fmt"
	"os/exec"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
	"github.com/rkuska/carn/internal/source/claude"
	"github.com/rkuska/carn/internal/source/codex"
)

var errResumeProviderUnavailable = errors.New("resume provider is unavailable")

type sessionLauncher interface {
	ResumeCommand(target conv.ResumeTarget) (*exec.Cmd, error)
}

type providerSessionLauncher struct {
	backends map[conv.Provider]src.Backend
}

func newDefaultSessionLauncher() sessionLauncher {
	return newSessionLauncher(claude.New(), codex.New())
}

func resolveSessionLauncher(launchers ...sessionLauncher) sessionLauncher {
	for _, launcher := range launchers {
		if launcher != nil {
			return launcher
		}
	}
	return newDefaultSessionLauncher()
}

func newSessionLauncher(backends ...src.Backend) sessionLauncher {
	lookup := make(map[conv.Provider]src.Backend, len(backends))
	for _, backend := range backends {
		if backend == nil {
			continue
		}
		provider := backend.Provider()
		if _, ok := lookup[provider]; ok {
			continue
		}
		lookup[provider] = backend
	}
	return providerSessionLauncher{backends: lookup}
}

func (l providerSessionLauncher) ResumeCommand(target conv.ResumeTarget) (*exec.Cmd, error) {
	backend, ok := l.backends[target.Provider]
	if !ok {
		return nil, fmt.Errorf("resumeCommand: %w", errResumeProviderUnavailable)
	}

	cmd, err := backend.ResumeCommand(target)
	if err != nil {
		return nil, fmt.Errorf("resumeCommand_%s: %w", target.Provider, err)
	}
	return cmd, nil
}
