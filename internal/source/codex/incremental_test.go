package codex

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

type lookupByFilePath struct {
	conversation conv.Conversation
}

func (l lookupByFilePath) ConversationByFilePath(
	context.Context,
	conv.Provider,
	string,
) (conv.Conversation, bool, error) {
	return l.conversation, true, nil
}

func (lookupByFilePath) ConversationBySessionID(
	context.Context,
	conv.Provider,
	string,
) (conv.Conversation, bool, error) {
	return conv.Conversation{}, false, nil
}

func (lookupByFilePath) ConversationByCacheKey(context.Context, string) (conv.Conversation, bool, error) {
	return conv.Conversation{}, false, nil
}

func TestScanChangedRolloutsTracksMissingFilesForIncrementalRebuild(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/missing.jsonl"
	lookup := lookupByFilePath{
		conversation: conv.Conversation{
			Ref: conv.Ref{Provider: conv.ProviderCodex, ID: "thread-main"},
		},
	}

	rollouts, byID, rebuildRootIDs, blockedRootIDs, drift, malformedData, err := scanChangedRollouts(
		context.Background(),
		[]string{path},
		lookup,
	)
	require.NoError(t, err)
	assert.Empty(t, rollouts)
	assert.Empty(t, byID)
	assert.Equal(t, map[string]struct{}{"thread-main": {}}, rebuildRootIDs)
	assert.Empty(t, blockedRootIDs)
	assert.Empty(t, drift.Findings())
	assert.True(t, malformedData.Empty())
}
