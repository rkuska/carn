package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClassifyStoreConversations(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)

	makeConv := func(id string, filePaths ...string) conversation {
		sessions := make([]sessionMeta, len(filePaths))
		for i, path := range filePaths {
			sessions[i] = sessionMeta{
				id:        id + "-s" + string(rune('0'+i)),
				filePath:  path,
				project:   project{displayName: "proj"},
				timestamp: ts,
			}
		}
		return conversation{
			ref:      conversationRef{provider: conversationProviderClaude, id: id},
			name:     id,
			project:  project{displayName: "proj"},
			sessions: sessions,
		}
	}

	tests := []struct {
		name             string
		newConversations []conversation
		oldCatalog       []conversation
		changedPaths     map[string]struct{}
		wantUnchanged    int
		wantChanged      int
		wantAdded        int
	}{
		{
			name:             "all new when old catalog is empty",
			newConversations: []conversation{makeConv("a", "/raw/a.jsonl"), makeConv("b", "/raw/b.jsonl")},
			oldCatalog:       nil,
			changedPaths:     nil,
			wantAdded:        2,
		},
		{
			name:             "all unchanged when no files changed",
			newConversations: []conversation{makeConv("a", "/raw/a.jsonl"), makeConv("b", "/raw/b.jsonl")},
			oldCatalog:       []conversation{makeConv("a", "/raw/a.jsonl"), makeConv("b", "/raw/b.jsonl")},
			changedPaths:     map[string]struct{}{},
			wantUnchanged:    2,
		},
		{
			name:             "changed when file in changed set",
			newConversations: []conversation{makeConv("a", "/raw/a.jsonl"), makeConv("b", "/raw/b.jsonl")},
			oldCatalog:       []conversation{makeConv("a", "/raw/a.jsonl"), makeConv("b", "/raw/b.jsonl")},
			changedPaths:     map[string]struct{}{"/raw/b.jsonl": {}},
			wantUnchanged:    1,
			wantChanged:      1,
		},
		{
			name: "changed when session count differs",
			newConversations: []conversation{
				makeConv("a", "/raw/a1.jsonl", "/raw/a2.jsonl"),
			},
			oldCatalog:   []conversation{makeConv("a", "/raw/a1.jsonl")},
			changedPaths: map[string]struct{}{},
			wantChanged:  1,
		},
		{
			name:             "removed conversation not in new list is implicitly dropped",
			newConversations: []conversation{makeConv("a", "/raw/a.jsonl")},
			oldCatalog:       []conversation{makeConv("a", "/raw/a.jsonl"), makeConv("b", "/raw/b.jsonl")},
			changedPaths:     map[string]struct{}{},
			wantUnchanged:    1,
		},
		{
			name: "mixed classification",
			newConversations: []conversation{
				makeConv("a", "/raw/a.jsonl"),
				makeConv("b", "/raw/b.jsonl"),
				makeConv("c", "/raw/c.jsonl"),
			},
			oldCatalog: []conversation{
				makeConv("a", "/raw/a.jsonl"),
				makeConv("b", "/raw/b.jsonl"),
			},
			changedPaths:  map[string]struct{}{"/raw/b.jsonl": {}},
			wantUnchanged: 1,
			wantChanged:   1,
			wantAdded:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan := classifyStoreConversations(tt.newConversations, tt.oldCatalog, tt.changedPaths)
			assert.Len(t, plan.unchanged, tt.wantUnchanged)
			assert.Len(t, plan.changed, tt.wantChanged)
			assert.Len(t, plan.added, tt.wantAdded)
		})
	}
}

func TestGroupSearchUnitsByConversation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		corpus         searchCorpus
		wantGroupSizes map[string]int
	}{
		{
			name:           "empty corpus",
			corpus:         searchCorpus{},
			wantGroupSizes: map[string]int{},
		},
		{
			name: "units from multiple conversations correctly grouped",
			corpus: searchCorpus{
				units: []searchUnit{
					{conversationID: "a", text: "hello"},
					{conversationID: "a", text: "world"},
					{conversationID: "b", text: "foo"},
					{conversationID: "c", text: "bar"},
					{conversationID: "b", text: "baz"},
				},
			},
			wantGroupSizes: map[string]int{"a": 2, "b": 2, "c": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			grouped := groupSearchUnitsByConversation(tt.corpus)
			assert.Len(t, grouped, len(tt.wantGroupSizes))
			for key, wantLen := range tt.wantGroupSizes {
				assert.Len(t, grouped[key], wantLen)
			}
		})
	}
}
