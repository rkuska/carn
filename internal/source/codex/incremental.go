package codex

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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
	changedRollouts, changedByID, rebuildRootIDs, blockedRootIDs, drift, malformedData, err := scanChangedRollouts(
		ctx,
		changedRawPaths,
		lookup,
	)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("scanChangedRollouts: %w", err)
	}

	families, rootIDs, err := buildIncrementalFamilies(
		ctx,
		changedRollouts,
		changedByID,
		rebuildRootIDs,
		lookup,
	)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("buildIncrementalFamilies: %w", err)
	}

	conversations, replaceKeysByConversation, familyDrift, familyMalformedData, err := resolveIncrementalFamilies(
		ctx,
		rootIDs,
		families,
		blockedRootIDs,
		lookup,
	)
	if err != nil {
		return src.IncrementalResolution{}, fmt.Errorf("resolveIncrementalFamilies: %w", err)
	}
	drift.Merge(familyDrift)
	malformedData.Merge(familyMalformedData)

	return src.IncrementalResolution{
		Conversations:                  conversations,
		ReplaceCacheKeysByConversation: replaceKeysByConversation,
		Drift:                          drift,
		MalformedData:                  malformedData,
	}, nil
}

func buildIncrementalFamilies(
	ctx context.Context,
	changedRollouts []scannedRollout,
	changedByID map[string]scannedRollout,
	rebuildRootIDs map[string]struct{},
	lookup src.IncrementalLookup,
) (map[string][]scannedRollout, []string, error) {
	families := make(map[string][]scannedRollout)
	rootIDs := make([]string, 0, len(rebuildRootIDs))
	seenRootIDs := make(map[string]struct{}, len(rebuildRootIDs)+len(changedRollouts))

	for rootID := range rebuildRootIDs {
		rootIDs = append(rootIDs, rootID)
		seenRootIDs[rootID] = struct{}{}
	}

	for _, rollout := range changedRollouts {
		rootID, err := resolveIncrementalRootID(ctx, rollout, changedByID, lookup)
		if err != nil {
			return nil, nil, fmt.Errorf("resolveIncrementalRootID: %w", err)
		}
		if _, ok := seenRootIDs[rootID]; !ok {
			rootIDs = append(rootIDs, rootID)
			seenRootIDs[rootID] = struct{}{}
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
	blockedRootIDs map[string]struct{},
	lookup src.IncrementalLookup,
) ([]conv.Conversation, map[string][]string, src.DriftReport, src.MalformedDataReport, error) {
	conversations := make([]conv.Conversation, 0)
	currentKeys := make(map[string]struct{})
	replaceKeysByConversation := make(map[string][]string)
	drift := src.NewDriftReport()
	malformedData := src.NewMalformedDataReport()

	for _, rootID := range rootIDs {
		if _, blocked := blockedRootIDs[rootID]; blocked {
			continue
		}

		resolved, familyDrift, familyMalformedData, err := resolveIncrementalFamily(
			ctx,
			rootID,
			families[rootID],
			lookup,
		)
		drift.Merge(familyDrift)
		malformedData.Merge(familyMalformedData)
		if err != nil {
			return nil, nil, drift, malformedData, fmt.Errorf("resolveIncrementalFamily: %w", err)
		}
		appendIncrementalFamilyConversations(
			&conversations,
			currentKeys,
			replaceKeysByConversation,
			resolved.conversations,
			resolved.replaceKeysByConversation,
		)
	}

	return conversations, replaceKeysByConversation, drift, malformedData, nil
}

type resolvedIncrementalFamily struct {
	conversations             []conv.Conversation
	replaceKeysByConversation map[string][]string
}

func resolveIncrementalFamily(
	ctx context.Context,
	rootID string,
	changed []scannedRollout,
	lookup src.IncrementalLookup,
) (resolvedIncrementalFamily, src.DriftReport, src.MalformedDataReport, error) {
	stored, ok, err := lookup.ConversationBySessionID(ctx, conv.ProviderCodex, rootID)
	if err != nil {
		return resolvedIncrementalFamily{}, src.DriftReport{}, src.MalformedDataReport{}, fmt.Errorf(
			"lookup.ConversationBySessionID: %w",
			err,
		)
	}

	grouped, drift, malformedData, err := scanIncrementalFamily(
		ctx,
		buildIncrementalFamilyPaths(stored, ok, changed),
	)
	if err != nil {
		return resolvedIncrementalFamily{}, drift, malformedData, fmt.Errorf("scanIncrementalFamily: %w", err)
	}
	if !malformedData.Empty() {
		return resolvedIncrementalFamily{}, drift, malformedData, nil
	}

	conversations := filterIncrementalFamilyConversations(grouped, rootID, ok)
	replaceKeys := src.DedupeAndSort(incrementalFamilyReplaceKeys(grouped, stored, ok))
	replaceKeysByConversation := make(map[string][]string, len(conversations))
	for _, conversation := range conversations {
		replaceKeysByConversation[conversation.CacheKey()] = append([]string(nil), replaceKeys...)
	}

	return resolvedIncrementalFamily{
		conversations:             conversations,
		replaceKeysByConversation: replaceKeysByConversation,
	}, drift, malformedData, nil
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
	replaceKeysByConversation map[string][]string,
	grouped []conv.Conversation,
	familyReplaceKeys map[string][]string,
) {
	for _, conversation := range grouped {
		cacheKey := conversation.CacheKey()
		if _, exists := currentKeys[cacheKey]; exists {
			continue
		}
		currentKeys[cacheKey] = struct{}{}
		*conversations = append(*conversations, conversation)
		replaceKeysByConversation[cacheKey] = append([]string(nil), familyReplaceKeys[cacheKey]...)
	}
}

func scanChangedRollouts(
	ctx context.Context,
	changedRawPaths []string,
	lookup src.IncrementalLookup,
) (
	[]scannedRollout,
	map[string]scannedRollout,
	map[string]struct{},
	map[string]struct{},
	src.DriftReport,
	src.MalformedDataReport,
	error,
) {
	rollouts := make([]scannedRollout, 0, len(changedRawPaths))
	byID := make(map[string]scannedRollout, len(changedRawPaths))
	rebuildRootIDs := make(map[string]struct{})
	blockedRootIDs := make(map[string]struct{})
	drift := src.NewDriftReport()
	malformedData := src.NewMalformedDataReport()

	for _, path := range src.DedupeAndSort(changedRawPaths) {
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, nil, drift, malformedData, fmt.Errorf("scanChangedRollouts_ctx: %w", err)
		}

		rollout, ok, err := scanRollout(path)
		drift.Merge(rollout.drift)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				if markErr := markIncrementalRoot(ctx, path, rebuildRootIDs, lookup); markErr != nil {
					return nil, nil, nil, nil, drift, malformedData, fmt.Errorf(
						"markIncrementalRoot_%s: %w",
						filepath.Base(path),
						markErr,
					)
				}
				continue
			}
			if errors.Is(err, src.ErrMalformedRawData) {
				malformedData.Record(path)
				if markErr := markIncrementalRoot(ctx, path, blockedRootIDs, lookup); markErr != nil {
					return nil, nil, nil, nil, drift, malformedData, fmt.Errorf(
						"markIncrementalRoot_%s: %w",
						filepath.Base(path),
						markErr,
					)
				}
				continue
			}
			return nil, nil, nil, nil, drift, malformedData, fmt.Errorf("scanRollout_%s: %w", filepath.Base(path), err)
		}
		if !ok {
			malformedData.Record(path)
			if markErr := markIncrementalRoot(ctx, path, blockedRootIDs, lookup); markErr != nil {
				return nil, nil, nil, nil, drift, malformedData, fmt.Errorf(
					"markIncrementalRoot_%s: %w",
					filepath.Base(path),
					markErr,
				)
			}
			continue
		}
		rollouts = append(rollouts, rollout)
		byID[rollout.meta.ID] = rollout
	}
	return rollouts, byID, rebuildRootIDs, blockedRootIDs, drift, malformedData, nil
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
) ([]conv.Conversation, src.DriftReport, src.MalformedDataReport, error) {
	paths := make([]string, 0, len(familyPaths))
	for path := range familyPaths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	rollouts := make([]scannedRollout, 0, len(paths))
	drift := src.NewDriftReport()
	malformedData := src.NewMalformedDataReport()
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, drift, malformedData, fmt.Errorf("scanIncrementalFamily_ctx: %w", err)
		}
		rollout, ok, err := scanRollout(path)
		drift.Merge(rollout.drift)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if errors.Is(err, src.ErrMalformedRawData) {
				malformedData.Record(path)
				continue
			}
			return nil, drift, malformedData, fmt.Errorf("scanRollout_%s: %w", filepath.Base(path), err)
		}
		if !ok {
			malformedData.Record(path)
			continue
		}
		rollouts = append(rollouts, rollout)
	}
	return groupRollouts(rollouts), drift, malformedData, nil
}

func markIncrementalRoot(
	ctx context.Context,
	path string,
	rootIDs map[string]struct{},
	lookup src.IncrementalLookup,
) error {
	stored, ok, err := lookup.ConversationByFilePath(ctx, conv.ProviderCodex, path)
	if err != nil {
		return fmt.Errorf("lookup.ConversationByFilePath: %w", err)
	}
	if ok {
		rootIDs[stored.ID()] = struct{}{}
	}
	return nil
}
