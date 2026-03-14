package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName  = "carn"
	configFileName = "config.toml"
)

// ResolvePath returns the resolved path to the config file under the
// user-scoped config directory.
func ResolvePath() (string, error) {
	return resolvePath(os.UserConfigDir)
}

func resolvePath(userConfigDir func() (string, error)) (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("userConfigDir: %w", err)
	}
	return filepath.Join(dir, configDirName, configFileName), nil
}
