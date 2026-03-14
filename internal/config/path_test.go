package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePath(t *testing.T) {
	t.Run("uses user config dir", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)

		got, err := ResolvePath()
		require.NoError(t, err)

		baseDir, err := os.UserConfigDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(baseDir, "carn", "config.toml"), got)
	})

	t.Run("returns error when user config dir lookup fails", func(t *testing.T) {
		got, err := resolvePath(func() (string, error) {
			return "", errors.New("boom")
		})

		require.Error(t, err)
		assert.Empty(t, got)
		assert.ErrorContains(t, err, "userConfigDir")
		assert.ErrorContains(t, err, "boom")
	})
}
