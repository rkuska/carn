package canonical

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClassifyStoreConversationsSeparatesAddedChangedAndUnchanged(t *testing.T) {
	t.Parallel()

	unchanged := conversation{
		Ref: conversationRef{Provider: conversationProvider("claude"), ID: "unchanged"},
		Sessions: []sessionMeta{{
			ID:        "unchanged",
			FilePath:  "/raw/a.jsonl",
			Timestamp: time.Now(),
		}},
	}
	changed := conversation{
		Ref: conversationRef{Provider: conversationProvider("claude"), ID: "changed"},
		Sessions: []sessionMeta{{
			ID:        "changed",
			FilePath:  "/raw/b.jsonl",
			Timestamp: time.Now(),
		}},
	}
	added := conversation{
		Ref: conversationRef{Provider: conversationProvider("claude"), ID: "added"},
		Sessions: []sessionMeta{{
			ID:        "added",
			FilePath:  "/raw/c.jsonl",
			Timestamp: time.Now(),
		}},
	}

	plan := classifyStoreConversations(
		[]conversation{unchanged, changed, added},
		[]conversation{unchanged, changed},
		map[string]struct{}{"/raw/b.jsonl": {}},
	)

	assert.Equal(t, []conversation{unchanged}, plan.unchanged)
	assert.Equal(t, []conversation{changed}, plan.changed)
	assert.Equal(t, []conversation{added}, plan.added)
}

func TestGroupSearchUnitsByConversation(t *testing.T) {
	t.Parallel()

	grouped := groupSearchUnitsByConversation(searchCorpus{units: []searchUnit{
		{conversationID: "a", text: "first"},
		{conversationID: "a", text: "second"},
		{conversationID: "b", text: "third"},
	}})

	assert.Len(t, grouped["a"], 2)
	assert.Len(t, grouped["b"], 1)
}

func TestClassifyStoreConversationsMarksGroupedConversationChangedWhenChildFileChanges(t *testing.T) {
	t.Parallel()

	grouped := conversation{
		Ref: conversationRef{Provider: conversationProvider("codex"), ID: "grouped"},
		Sessions: []sessionMeta{
			{
				ID:        "main",
				FilePath:  "/raw/main.jsonl",
				Timestamp: time.Now(),
			},
			{
				ID:         "child",
				FilePath:   "/raw/child.jsonl",
				Timestamp:  time.Now(),
				IsSubagent: true,
			},
		},
	}

	plan := classifyStoreConversations(
		[]conversation{grouped},
		[]conversation{grouped},
		map[string]struct{}{"/raw/child.jsonl": {}},
	)

	assert.Empty(t, plan.unchanged)
	assert.Equal(t, []conversation{grouped}, plan.changed)
}
