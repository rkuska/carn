package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSearchUnitsIndexesVisibleTextOnly(t *testing.T) {
	t.Parallel()

	session := sessionFull{
		messages: []message{
			{
				role:     roleUser,
				text:     "Hello User",
				thinking: "Internal Thought",
				toolCalls: []toolCall{
					{name: "Read", summary: "README.md"},
				},
				toolResults: []toolResult{
					{content: "Tool Output"},
				},
			},
		},
	}

	got := buildSearchUnits("conv-1", session)
	require.Len(t, got, 2)
	assert.Equal(t, "Hello User", got[0].text)
	assert.Equal(t, "README.md", got[1].text)
}

func TestChunkSearchTextSplitsLongLinesWithOverlap(t *testing.T) {
	t.Parallel()

	text := "0123456789abcdefghijklmnopqrstuvwxyz"
	got := chunkSearchText(text, 10, 4)
	require.Len(t, got, 6)
	assert.Equal(t, "0123456789", got[0])
	assert.Equal(t, "6789abcdef", got[1])
}

func TestDeepSearchCmdReturnsGroupedFuzzyMatches(t *testing.T) {
	t.Parallel()

	mainConvs := []conversation{
		testNamedConversation("s1", "slug-1"),
		testNamedConversation("s2", "slug-2"),
	}
	corpus := searchCorpus{
		units: []searchUnit{
			{conversationID: mainConvs[0].cacheKey(), text: "contains alpha needle"},
			{conversationID: mainConvs[1].cacheKey(), text: "contains beta needle"},
			{conversationID: mainConvs[1].cacheKey(), text: "secondary beta result"},
		},
	}

	msg := deepSearchCmd(context.Background(), "btndl", 1, mainConvs, corpus)()
	result := requireMsgType[deepSearchResultMsg](t, msg)
	require.Len(t, result.conversations, 1)
	assert.Equal(t, "s2", result.conversations[0].id())
	assert.Contains(t, result.conversations[0].searchPreview, "beta needle")
}

func TestDeepSearchCmdEmptyQueryReturnsMainConversations(t *testing.T) {
	t.Parallel()

	mainConvs := []conversation{
		testNamedConversation("s1", "slug-1"),
		testNamedConversation("s2", "slug-2"),
	}
	msg := deepSearchCmd(context.Background(), "", 1, mainConvs, searchCorpus{})()
	result := requireMsgType[deepSearchResultMsg](t, msg)
	assert.Len(t, result.conversations, 2)
}
