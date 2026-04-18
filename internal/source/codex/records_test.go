package codex

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	src "github.com/rkuska/carn/internal/source"
)

func TestOpenReaderMarksMissingFileAsMalformedRawData(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.jsonl")

	file, br, err := openReader(path)
	require.Error(t, err)
	assert.Nil(t, file)
	assert.Nil(t, br)
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.ErrorIs(t, err, src.ErrMalformedRawData)
}

func TestOpenReaderPropagatesPermissionDeniedErrors(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("permission-denied semantics differ on windows")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "thread.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"type":"session_meta"}`), 0o644))
	require.NoError(t, os.Chmod(path, 0o000))
	t.Cleanup(func() {
		require.NoError(t, os.Chmod(path, 0o644))
	})

	file, br, err := openReader(path)
	require.Error(t, err)
	assert.Nil(t, file)
	assert.Nil(t, br)
	assert.False(t, errors.Is(err, src.ErrMalformedRawData))
	assert.ErrorIs(t, err, fs.ErrPermission)
}
