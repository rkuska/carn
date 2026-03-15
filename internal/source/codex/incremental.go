package codex

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func (Source) ResolveIncremental(
	ctx context.Context,
	_ string,
	changedRawPaths []string,
	lookup src.IncrementalLookup,
) (src.IncrementalResolution, error) {
	changedRollouts, changedByID, err := scanChangedRollouts(ctx, changedRawPaths)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("scanChangedRollouts: %w", err)
	}

	families, rootIDs, err := buildIncrementalFamilies(ctx, changedRollouts, changedByID, lookup)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("buildIncrementalFamilies: %w", err)
	}

	conversations, replaceKeys, err := resolveIncrementalFamilies(ctx, rootIDs, families, lookup)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("resolveIncrementalFamilies: %w", err)
	}

	return src.IncrementalResolution{
		Conversations:    conversations,
		ReplaceCacheKeys: src.SortedKeys(replaceKeys),
	}, nil
}

func buildIncrementalFamilies(
	ctx context.Context,
	changedRollouts []scannedRollout,
	changedByID map[string]scannedRollout,
	lookup src.IncrementalLookup,
) (map[string][]scannedRollout, []string, error) {
	families := make(map[string][]scannedRollout)
	rootIDs := make([]string, 0)

	for _, rollout := range changedRollouts {
		rootID, err := resolveIncrementalRootID(ctx, rollout, changedByID, lookup)
		if err != nil {
			return nil, nil, fmt.Errorf("resolveIncrementalRootID: %w", err)
		}
		if _, ok := families[rootID]; !ok {
			rootIDs = append(rootIDs, rootID)
		}
		families[rootID] = append(families[rootID], rollout)
	}

	sort.Strings(rootIDs)
	return families, rootIDs, nil
}

func resolveIncrementalFamilies(
	ctx context.Context,
	rootIDs []string,
	families map[string][]scannedRollout,
	lookup src.IncrementalLookup,
) ([]conv.Conversation, map[string]struct{}, error) {
	conversations := make([]conv.Conversation, 0)
	currentKeys := make(map[string]struct{})
	replaceKeys := make(map[string]struct{})

	for _, rootID := range rootIDs {
		resolved, err := resolveIncrementalFamily(ctx, rootID, families[rootID], lookup)
		if err != nil {
			return nil, nil, fmt.Errorf("resolveIncrementalFamily: %w", err)
		}
		appendIncrementalFamilyConversations(&conversations, currentKeys, resolved.conversations)
		for _, cacheKey := range resolved.replaceCacheKeys {
			replaceKeys[cacheKey] = struct{}{}
		}
	}

	return conversations, replaceKeys, nil
}

type resolvedIncrementalFamily struct {
	conversations    []conv.Conversation
	replaceCacheKeys []string
}

func resolveIncrementalFamily(
	ctx context.Context,
	rootID string,
	changed []scannedRollout,
	lookup src.IncrementalLookup,
) (resolvedIncrementalFamily, error) {
	stored, ok, err := lookup.ConversationBySessionID(ctx, conv.ProviderCodex, rootID)
	if err != nil {
		return resolvedIncrementalFamily{}, fmt.Errorf("lookup.ConversationBySessionID: %w", err)
	}

	grouped, err := scanIncrementalFamily(ctx, buildIncrementalFamilyPaths(stored, ok, changed))
	if err != nil {
		return resolvedIncrementalFamily{}, fmt.Errorf("scanIncrementalFamily: %w", err)
	}

	return resolvedIncrementalFamily{
		conversations:    filterIncrementalFamilyConversations(grouped, rootID, ok),
		replaceCacheKeys: incrementalFamilyReplaceKeys(grouped, stored, ok),
	}, nil
}

func buildIncrementalFamilyPaths(
	stored conv.Conversation,
	hasStored bool,
	changed []scannedRollout,
) map[string]struct{} {
	familyPaths := make(map[string]struct{})
	if hasStored {
		for _, path := range stored.FilePaths() {
			familyPaths[path] = struct{}{}
		}
	}
	for _, rollout := range changed {
		familyPaths[rollout.meta.FilePath] = struct{}{}
	}
	return familyPaths
}

func filterIncrementalFamilyConversations(
	grouped []conv.Conversation,
	rootID string,
	hasStored bool,
) []conv.Conversation {
	if !hasStored {
		return grouped
	}

	filtered := make([]conv.Conversation, 0, 1)
	for _, conversation := range grouped {
		if conversation.ID() == rootID {
			filtered = append(filtered, conversation)
		}
	}
	return filtered
}

func incrementalFamilyReplaceKeys(
	grouped []conv.Conversation,
	stored conv.Conversation,
	hasStored bool,
) []string {
	keys := make([]string, 0, len(grouped)+1)
	if hasStored {
		keys = append(keys, stored.CacheKey())
	}
	for _, conversation := range grouped {
		keys = append(keys, conversation.CacheKey())
	}
	return keys
}

func appendIncrementalFamilyConversations(
	conversations *[]conv.Conversation,
	currentKeys map[string]struct{},
	grouped []conv.Conversation,
) {
	for _, conversation := range grouped {
		cacheKey := conversation.CacheKey()
		if _, exists := currentKeys[cacheKey]; exists {
			continue
		}
		currentKeys[cacheKey] = struct{}{}
		*conversations = append(*conversations, conversation)
	}
}

func scanChangedRollouts(
	ctx context.Context,
	changedRawPaths []string,
) ([]scannedRollout, map[string]scannedRollout, error) {
	rollouts := make([]scannedRollout, 0, len(changedRawPaths))
	byID := make(map[string]scannedRollout, len(changedRawPaths))

	for _, path := range src.DedupeAndSort(changedRawPaths) {
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("scanChangedRollouts_ctx: %w", err)
		}
		if _, err := os.Stat(path); err != nil {
			return nil, nil, fmt.Errorf("scanChangedRollouts_osStat_%s: %w", filepath.Base(path), err)
		}

		rollout, ok, err := scanRollout(path)
		if err != nil {
			return nil, nil, fmt.Errorf("scanRollout_%s: %w", filepath.Base(path), err)
		}
		if !ok {
			return nil, nil, fmt.Errorf(
				"scanChangedRollouts_%s: %w",
				filepath.Base(path),
				errors.New("changed rollout missing session metadata"),
			)
		}
		rollouts = append(rollouts, rollout)
		byID[rollout.meta.ID] = rollout
	}
	return rollouts, byID, nil
}

func resolveIncrementalRootID(
	ctx context.Context,
	rollout scannedRollout,
	changedByID map[string]scannedRollout,
	lookup src.IncrementalLookup,
) (string, error) {
	current := rollout
	seen := make(map[string]struct{}, 4)

	for current.meta.IsSubagent {
		parentID := current.link.parentThreadID
		if parentID == "" {
			return current.meta.ID, nil
		}
		if _, ok := seen[current.meta.ID]; ok {
			return current.meta.ID, nil
		}
		seen[current.meta.ID] = struct{}{}

		if parent, ok := changedByID[parentID]; ok {
			current = parent
			continue
		}

		stored, ok, err := lookup.ConversationBySessionID(ctx, conv.ProviderCodex, parentID)
		if err != nil {
			return "", fmt.Errorf("lookup.ConversationBySessionID: %w", err)
		}
		if !ok {
			return current.meta.ID, nil
		}
		return stored.ID(), nil
	}

	return current.meta.ID, nil
}

func scanIncrementalFamily(
	ctx context.Context,
	familyPaths map[string]struct{},
) ([]conv.Conversation, error) {
	paths := make([]string, 0, len(familyPaths))
	for path := range familyPaths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	rollouts := make([]scannedRollout, 0, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("scanIncrementalFamily_ctx: %w", err)
		}
		rollout, ok, err := scanRollout(path)
		if err != nil {
			return nil, fmt.Errorf("scanRollout_%s: %w", filepath.Base(path), err)
		}
		if !ok {
			continue
		}
		rollouts = append(rollouts, rollout)
	}
	return groupRollouts(rollouts), nil
}
