package source

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDedupe(t *testing.T) {
	t.Parallel()

	assert.Nil(t, Dedupe[string](nil))
	assert.Equal(t, []string{"b", "a", "", "c"}, Dedupe([]string{"b", "a", "b", "", "c", "a"}))
}

func TestDedupeAndSort(t *testing.T) {
	t.Parallel()

	assert.Nil(t, DedupeAndSort(nil))
	assert.Equal(t, []string{"a", "b", "c"}, DedupeAndSort([]string{"b", "", "a", "b", "c", ""}))
	assert.Empty(t, DedupeAndSort([]string{"", ""}))
}

func TestSortedKeys(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		[]string{"a", "b", "c"},
		SortedKeys(map[string]struct{}{"c": {}, "a": {}, "b": {}}),
	)
}

func TestFileNeedsSync(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, srcPath, dstPath string)
		want  bool
	}{
		{
			name: "dst missing",
			setup: func(t *testing.T, srcPath, _ string) {
				t.Helper()
				writeTestFile(t, srcPath, "content")
			},
			want: true,
		},
		{
			name: "size differs",
			setup: func(t *testing.T, srcPath, dstPath string) {
				t.Helper()
				writeTestFile(t, srcPath, "longer content")
				writeTestFile(t, dstPath, "short")
			},
			want: true,
		},
		{
			name: "src newer",
			setup: func(t *testing.T, srcPath, dstPath string) {
				t.Helper()
				writeTestFile(t, srcPath, "content")
				writeTestFile(t, dstPath, "content")
				past := time.Now().Add(-time.Hour)
				require.NoError(t, os.Chtimes(dstPath, past, past))
			},
			want: true,
		},
		{
			name: "identical",
			setup: func(t *testing.T, srcPath, dstPath string) {
				t.Helper()
				writeTestFile(t, srcPath, "content")
				writeTestFile(t, dstPath, "content")
				now := time.Now()
				require.NoError(t, os.Chtimes(srcPath, now, now))
				require.NoError(t, os.Chtimes(dstPath, now, now))
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			srcPath := filepath.Join(dir, "src", "file.jsonl")
			dstPath := filepath.Join(dir, "dst", "file.jsonl")

			tt.setup(t, srcPath, dstPath)

			srcInfo, err := os.Stat(srcPath)
			require.NoError(t, err)
			assert.Equal(t, tt.want, FileNeedsSync(srcInfo, dstPath))
		})
	}
}

func TestStatDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.txt")
	writeTestFile(t, filePath, "content")

	isDir, err := StatDir(dir)
	require.NoError(t, err)
	assert.True(t, isDir)

	isDir, err = StatDir(filePath)
	require.NoError(t, err)
	assert.False(t, isDir)

	isDir, err = StatDir(filepath.Join(dir, "missing"))
	require.ErrorIs(t, err, fs.ErrNotExist)
	assert.False(t, isDir)
}

func TestProviderRawDir(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		filepath.Join("/tmp/archive", "claude", "raw"),
		ProviderRawDir("/tmp/archive", conv.ProviderClaude),
	)
}

func TestInsertAt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []int{1, 9, 2, 3}, InsertAt([]int{1, 2, 3}, 1, 9))
}

func TestInsertSliceAt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []int{1, 9, 8, 2, 3}, InsertSliceAt([]int{1, 2, 3}, 1, []int{9, 8}))
	assert.Equal(t, []int{1, 2, 3}, InsertSliceAt([]int{1, 2, 3}, 1, nil))
}

func TestFindInsertPosition(t *testing.T) {
	t.Parallel()

	zeroTime := time.Time{}
	timestamps := []time.Time{
		time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		zeroTime,
		time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	assert.Equal(t, 3, FindInsertPosition(timestamps, time.Time{}, func(ts time.Time) time.Time { return ts }))
	assert.Equal(
		t,
		1,
		FindInsertPosition(
			timestamps,
			time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
			func(ts time.Time) time.Time { return ts },
		),
	)
	assert.Equal(
		t,
		3,
		FindInsertPosition(
			timestamps,
			time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			func(ts time.Time) time.Time { return ts },
		),
	)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
