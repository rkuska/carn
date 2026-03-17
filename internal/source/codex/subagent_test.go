package codex

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestGroupRolloutsNestsSubagentsUnderRoot(t *testing.T) {
	t.Parallel()

	rollouts := []scannedRollout{
		testScannedRollout("root", "2026-03-16T10:00:00Z", false, "", 7),
		testScannedRollout("child-2", "2026-03-16T10:02:00Z", true, "child-1", 5),
		testScannedRollout("child-1", "2026-03-16T10:01:00Z", true, "root", 6),
	}

	conversations := groupRollouts(rollouts)
	require.Len(t, conversations, 1)
	require.Len(t, conversations[0].Sessions, 3)

	assert.Equal(t, []string{"root", "child-1", "child-2"}, conversationSessionIDs(conversations[0]))
	assert.Equal(t, 7, conversations[0].Sessions[0].MainMessageCount)
	assert.Zero(t, conversations[0].Sessions[1].MainMessageCount)
	assert.Zero(t, conversations[0].Sessions[2].MainMessageCount)
}

func TestGroupRolloutsKeepsUnresolvedSubagentsStandalone(t *testing.T) {
	t.Parallel()

	rollouts := []scannedRollout{
		testScannedRollout("root", "2026-03-16T10:00:00Z", false, "", 3),
		testScannedRollout("orphan", "2026-03-16T10:01:00Z", true, "missing", 2),
		testScannedRollout("cycle-a", "2026-03-16T10:02:00Z", true, "cycle-b", 2),
		testScannedRollout("cycle-b", "2026-03-16T10:03:00Z", true, "cycle-a", 2),
	}

	conversations := groupRollouts(rollouts)
	require.Len(t, conversations, 4)

	assert.Equal(t,
		[]string{"root", "orphan", "cycle-a", "cycle-b"},
		[]string{
			conversations[0].ID(),
			conversations[1].ID(),
			conversations[2].ID(),
			conversations[3].ID(),
		},
	)
	assert.Equal(t, []string{"root"}, conversationSessionIDs(conversations[0]))
	assert.Equal(t, []string{"orphan"}, conversationSessionIDs(conversations[1]))
	assert.Equal(t, []string{"cycle-a"}, conversationSessionIDs(conversations[2]))
	assert.Equal(t, []string{"cycle-b"}, conversationSessionIDs(conversations[3]))
}

func TestParseSubagentLinkExtractsFieldsFromRawJSON(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(map[string]any{
		"subagent": map[string]any{
			"thread_spawn": map[string]any{
				"parent_thread_id": "root-thread",
				"agent_nickname":   "worker-1",
				"agent_role":       "worker",
			},
		},
	})
	require.NoError(t, err)

	link, ok := parseSubagentLink(raw)
	require.True(t, ok)
	assert.Equal(t, "root-thread", link.parentThreadID)
	assert.Equal(t, "worker-1", link.agentNickname)
	assert.Equal(t, "worker", link.agentRole)
}

func testScannedRollout(
	id string,
	rawTimestamp string,
	isSubagent bool,
	parentThreadID string,
	mainMessageCount int,
) scannedRollout {
	timestamp := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	parsed, err := time.Parse(time.RFC3339, rawTimestamp)
	if err == nil {
		timestamp = parsed
	}

	return scannedRollout{
		meta: conv.SessionMeta{
			ID:               id,
			Timestamp:        timestamp,
			Project:          conv.Project{DisplayName: "project"},
			MainMessageCount: mainMessageCount,
			IsSubagent:       isSubagent,
		},
		link: subagentLink{parentThreadID: parentThreadID},
	}
}

func conversationSessionIDs(conversation conv.Conversation) []string {
	ids := make([]string, 0, len(conversation.Sessions))
	for _, session := range conversation.Sessions {
		ids = append(ids, session.ID)
	}
	return ids
}
