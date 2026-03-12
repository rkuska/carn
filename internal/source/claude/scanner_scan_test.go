package claude

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceScanGroupsFixtureCorpusIntoConversations(t *testing.T) {
	t.Parallel()

	baseDir := copyScannerFixtureCorpus(t)

	conversations, err := New().Scan(context.Background(), baseDir)
	require.NoError(t, err)
	require.NotEmpty(t, conversations)

	var sawSubagent bool
	for _, conversation := range conversations {
		if conversation.IsSubagent() {
			sawSubagent = true
		}
	}

	assert.True(t, sawSubagent)
}
