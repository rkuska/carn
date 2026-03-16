package codex

import (
	"encoding/json"
	"sort"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

type subagentLink struct {
	parentThreadID string
	agentNickname  string
	agentRole      string
}

type scannedRollout struct {
	meta conv.SessionMeta
	link subagentLink
}

type sourcePayload struct {
	Subagent struct {
		ThreadSpawn struct {
			ParentThreadID string `json:"parent_thread_id"`
			AgentNickname  string `json:"agent_nickname"`
			AgentRole      string `json:"agent_role"`
		} `json:"thread_spawn"`
	} `json:"subagent"`
}

func parseSubagentLink(raw json.RawMessage) (subagentLink, bool) {
	if !startsWithByte(raw, '{') {
		return subagentLink{}, false
	}

	var payload sourcePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return subagentLink{}, false
	}

	parentThreadID := strings.TrimSpace(payload.Subagent.ThreadSpawn.ParentThreadID)
	if parentThreadID == "" {
		return subagentLink{}, false
	}

	return subagentLink{
		parentThreadID: parentThreadID,
		agentNickname:  strings.TrimSpace(payload.Subagent.ThreadSpawn.AgentNickname),
		agentRole:      strings.TrimSpace(payload.Subagent.ThreadSpawn.AgentRole),
	}, true
}

func groupRollouts(rollouts []scannedRollout) []conv.Conversation {
	if len(rollouts) == 0 {
		return nil
	}

	byID := make(map[string]scannedRollout, len(rollouts))
	for _, rollout := range rollouts {
		byID[rollout.meta.ID] = rollout
	}

	childrenByRoot := make(map[string][]scannedRollout)
	roots := make([]string, 0, len(rollouts))
	var standalone []scannedRollout

	for _, rollout := range rollouts {
		if !rollout.meta.IsSubagent {
			roots = append(roots, rollout.meta.ID)
			continue
		}

		rootID, ok := resolveRootRolloutID(rollout, byID)
		if !ok {
			standalone = append(standalone, rollout)
			continue
		}
		childrenByRoot[rootID] = append(childrenByRoot[rootID], rollout)
	}

	conversations := make([]conv.Conversation, 0, len(roots)+len(standalone))
	for _, rootID := range roots {
		root, ok := byID[rootID]
		if !ok {
			continue
		}
		conversations = append(conversations, buildConversation(root, childrenByRoot[rootID]))
	}
	for _, rollout := range standalone {
		conversations = append(conversations, buildConversation(rollout, nil))
	}
	return conversations
}

func resolveRootRolloutID(rollout scannedRollout, byID map[string]scannedRollout) (string, bool) {
	current := rollout
	seen := make(map[string]struct{}, 4)
	for current.meta.IsSubagent {
		parentID := current.link.parentThreadID
		if parentID == "" {
			return "", false
		}
		if _, ok := seen[current.meta.ID]; ok {
			return "", false
		}
		seen[current.meta.ID] = struct{}{}

		parent, ok := byID[parentID]
		if !ok {
			return "", false
		}
		current = parent
	}
	if current.meta.ID == "" {
		return "", false
	}
	return current.meta.ID, true
}

func buildConversation(root scannedRollout, children []scannedRollout) conv.Conversation {
	sessions := make([]conv.SessionMeta, 0, 1+len(children))
	sessions = append(sessions, root.meta)
	sort.SliceStable(children, func(i, j int) bool {
		return children[i].meta.Timestamp.Before(children[j].meta.Timestamp)
	})
	for _, child := range children {
		meta := child.meta
		meta.MainMessageCount = 0
		sessions = append(sessions, meta)
	}

	return conv.Conversation{
		Ref: conv.Ref{
			Provider: conv.ProviderCodex,
			ID:       root.meta.ID,
		},
		Project:  root.meta.Project,
		Sessions: sessions,
	}
}

func startsWithByte(b []byte, target byte) bool {
	for _, c := range b {
		switch c {
		case ' ', '\t', '\n', '\r':
			continue
		default:
			return c == target
		}
	}
	return false
}
