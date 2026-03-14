package config

import (
	"os"
	"path/filepath"
)

const (
	configDirName  = "carn"
	configFileName = "config.toml"
)

// FilePath returns the resolved path to the config file.
// It respects $XDG_CONFIG_HOME and falls back to ~/.config/carn/config.toml.
func FilePath() string {
	return filepath.Join(configDir(), configFileName)
}

// FileExists reports whether a config file exists at the resolved path.
func FileExists() bool {
	info, err := os.Stat(FilePath())
	return err == nil && !info.IsDir()
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, configDirName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", configDirName)
}
