package claude

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	src "github.com/rkuska/carn/internal/source"
)

func TestResolveIncrementalPathMarksMissingFileAsMalformedRawData(t *testing.T) {
	t.Parallel()

	file := sessionFile{
		path:         t.TempDir() + "/missing.jsonl",
		project:      project{DisplayName: "demo"},
		groupDirName: "project-a",
	}

	_, drift, err := resolveIncrementalPath(context.Background(), file)
	require.Error(t, err)
	assert.Empty(t, drift.Findings())
	assert.ErrorIs(t, err, src.ErrMalformedRawData)
}
