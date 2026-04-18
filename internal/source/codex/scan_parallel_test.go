package codex

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanRolloutsParallelLogsMalformedSkipAtWarn(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.InfoLevel)
	ctx := logger.WithContext(context.Background())

	missing := filepath.Join(t.TempDir(), "missing.jsonl")

	rollouts, _, report, err := scanRolloutsParallel(ctx, []string{missing})
	require.NoError(t, err)
	require.Empty(t, rollouts)
	assert.Equal(t, 1, report.Count())

	out := buf.String()
	assert.Contains(t, out, `"level":"warn"`)
	assert.Contains(t, out, `"provider":"codex"`)
	assert.Contains(t, out, `"path":"`+missing+`"`)
	assert.Contains(t, out, "skipping malformed raw data")
}
