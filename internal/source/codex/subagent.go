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

func parseSubagentLink(raw json.RawMessage) (subagentLink, bool) {
	if !startsWithByte(raw, '{') {
		return subagentLink{}, false
	}

	parentThreadID, ok := extractRawJSONStringFieldByMarker(raw, parentThreadIDFieldMarker)
	if !ok {
		return subagentLink{}, false
	}
	parentThreadID = strings.TrimSpace(parentThreadID)
	if parentThreadID == "" {
		return subagentLink{}, false
	}

	return subagentLink{
		parentThreadID: parentThreadID,
		agentNickname:  strings.TrimSpace(extractRawJSONStringFieldOrEmptyByMarker(raw, agentNicknameFieldMarker)),
		agentRole:      strings.TrimSpace(extractRawJSONStringFieldOrEmptyByMarker(raw, agentRoleFieldMarker)),
	}, true
}

func groupRollouts(rollouts []scannedRollout) []conv.Conversation {
	if len(rollouts) == 0 {
		return nil
	}

	grouper := newRolloutGrouper(rollouts)
	childrenByRoot := make([][]int, len(rollouts))
	roots := make([]int, 0, len(rollouts))
	standalone := make([]int, 0)

	for i, rollout := range rollouts {
		if !rollout.meta.IsSubagent {
			roots = append(roots, i)
			continue
		}

		rootIndex, ok := grouper.resolveRootIndex(i)
		if !ok {
			standalone = append(standalone, i)
			continue
		}
		childrenByRoot[rootIndex] = append(childrenByRoot[rootIndex], i)
	}

	conversations := make([]conv.Conversation, 0, len(roots)+len(standalone))
	for _, rootIndex := range roots {
		conversations = append(conversations, buildConversation(rollouts[rootIndex], rollouts, childrenByRoot[rootIndex]))
	}
	for _, index := range standalone {
		conversations = append(conversations, buildConversation(rollouts[index], rollouts, nil))
	}
	return conversations
}

const (
	unresolvedRootIndex = -2
	invalidRootIndex    = -1
)

type rolloutGrouper struct {
	rollouts    []scannedRollout
	indexByID   map[string]int
	rootByIndex []int
	resolving   []bool
}

func newRolloutGrouper(rollouts []scannedRollout) rolloutGrouper {
	grouper := rolloutGrouper{
		rollouts:    rollouts,
		indexByID:   make(map[string]int, len(rollouts)),
		rootByIndex: make([]int, len(rollouts)),
		resolving:   make([]bool, len(rollouts)),
	}
	for i := range rollouts {
		grouper.indexByID[rollouts[i].meta.ID] = i
		grouper.rootByIndex[i] = unresolvedRootIndex
		if !rollouts[i].meta.IsSubagent {
			grouper.rootByIndex[i] = i
		}
	}
	return grouper
}

func (g *rolloutGrouper) resolveRootIndex(index int) (int, bool) {
	if cached := g.rootByIndex[index]; cached != unresolvedRootIndex {
		return cached, cached != invalidRootIndex
	}
	if g.resolving[index] {
		g.rootByIndex[index] = invalidRootIndex
		return invalidRootIndex, false
	}

	g.resolving[index] = true
	defer func() {
		g.resolving[index] = false
	}()

	parentID := g.rollouts[index].link.parentThreadID
	if parentID == "" {
		g.rootByIndex[index] = invalidRootIndex
		return invalidRootIndex, false
	}

	parentIndex, ok := g.indexByID[parentID]
	if !ok {
		g.rootByIndex[index] = invalidRootIndex
		return invalidRootIndex, false
	}

	rootIndex, ok := g.resolveRootIndex(parentIndex)
	if !ok {
		g.rootByIndex[index] = invalidRootIndex
		return invalidRootIndex, false
	}

	g.rootByIndex[index] = rootIndex
	return rootIndex, true
}

func buildConversation(root scannedRollout, rollouts []scannedRollout, childIndices []int) conv.Conversation {
	sessions := make([]conv.SessionMeta, 0, 1+len(childIndices))
	sessions = append(sessions, root.meta)

	children := make([]conv.SessionMeta, 0, len(childIndices))
	for _, childIndex := range childIndices {
		meta := rollouts[childIndex].meta
		meta.MainMessageCount = 0
		children = append(children, meta)
	}
	sort.SliceStable(children, func(i, j int) bool {
		return children[i].Timestamp.Before(children[j].Timestamp)
	})
	sessions = append(sessions, children...)

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
