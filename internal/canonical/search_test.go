package canonical

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunDeepSearchReturnsRankedPreviewResults(t *testing.T) {
	t.Parallel()

	conversations := []conversation{
		{
			Ref:      conversationRef{Provider: conversationProvider("claude"), ID: "s1"},
			Sessions: []sessionMeta{{ID: "s1", Timestamp: time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC)}},
		},
		{
			Ref:      conversationRef{Provider: conversationProvider("claude"), ID: "s2"},
			Sessions: []sessionMeta{{ID: "s2", Timestamp: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)}},
		},
	}

	results, ok := runDeepSearch(context.Background(), "beta", conversations, searchCorpus{units: []searchUnit{
		{conversationID: conversations[0].CacheKey(), text: "contains alpha needle"},
		{conversationID: conversations[1].CacheKey(), text: "contains beta needle"},
		{conversationID: conversations[1].CacheKey(), text: "secondary beta result"},
	}})
	require.True(t, ok)
	require.Len(t, results, 1)
	assert.Equal(t, "s2", results[0].ID())
	assert.Contains(t, results[0].SearchPreview, "beta")
}
