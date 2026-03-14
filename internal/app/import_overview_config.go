package app

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/rkuska/carn/internal/config"
)

type configReloadedMsg struct {
	cfg  config.Config
	path string
	err  error
}

func createAndEditConfigCmd(path, template string) tea.Cmd {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return func() tea.Msg {
			return configReloadedMsg{err: fmt.Errorf("os.MkdirAll: %w", err)}
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		_, writeErr := f.Write([]byte(template))
		_ = f.Close()
		if writeErr != nil {
			return func() tea.Msg {
				return configReloadedMsg{err: fmt.Errorf("writeTemplate: %w", writeErr)}
			}
		}
	} else if !os.IsExist(err) {
		return func() tea.Msg {
			return configReloadedMsg{err: fmt.Errorf("os.OpenFile: %w", err)}
		}
	}

	editorCmd := newEditorCmd(path)
	return tea.ExecProcess(editorCmd, func(err error) tea.Msg {
		if err != nil {
			return configReloadedMsg{err: fmt.Errorf("editor: %w", err)}
		}
		cfg, loadErr := config.Load()
		return configReloadedMsg{cfg: cfg, path: path, err: loadErr}
	})
}
