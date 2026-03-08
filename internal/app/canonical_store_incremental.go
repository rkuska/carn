package app

type storeRebuildPlan struct {
	unchanged []conversation
	changed   []conversation
	added     []conversation
}

func classifyStoreConversations(
	newConversations []conversation,
	oldCatalog []conversation,
	changedRawPaths map[string]struct{},
) storeRebuildPlan {
	if len(oldCatalog) == 0 {
		return storeRebuildPlan{added: newConversations}
	}

	oldByKey := make(map[string]conversation, len(oldCatalog))
	for _, conv := range oldCatalog {
		oldByKey[conv.cacheKey()] = conv
	}

	var plan storeRebuildPlan
	for _, conv := range newConversations {
		old, exists := oldByKey[conv.cacheKey()]
		if !exists {
			plan.added = append(plan.added, conv)
			continue
		}

		if len(conv.sessions) != len(old.sessions) {
			plan.changed = append(plan.changed, conv)
			continue
		}

		if hasChangedFiles(conv, changedRawPaths) {
			plan.changed = append(plan.changed, conv)
			continue
		}

		plan.unchanged = append(plan.unchanged, conv)
	}

	return plan
}

func hasChangedFiles(conv conversation, changedPaths map[string]struct{}) bool {
	for _, path := range conv.filePaths() {
		if _, ok := changedPaths[path]; ok {
			return true
		}
	}
	return false
}

func groupSearchUnitsByConversation(corpus searchCorpus) map[string][]searchUnit {
	grouped := make(map[string][]searchUnit)
	for _, unit := range corpus.units {
		grouped[unit.conversationID] = append(grouped[unit.conversationID], unit)
	}
	return grouped
}
