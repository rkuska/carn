package codex

import conv "github.com/rkuska/carn/internal/conversation"

func classifyCodexToolActionFromInput(
	name string,
	input string,
	readEvidence map[string]struct{},
) conv.NormalizedAction {
	switch {
	case name == toolNameApplyPatch:
		return classifyPatchNormalizedAction(input, readEvidence)
	case name == toolNameExecCommand || name == toolNameShellCommand || name == toolNameWriteStdin:
		classified := classifyCommand(input)
		return conv.NormalizedAction{
			Type:    classified.actionType,
			Targets: classified.targets,
		}
	case name == toolNameUpdatePlan:
		return conv.NormalizedAction{Type: conv.NormalizedActionPlan}
	case isCodexDelegateTool(name):
		return conv.NormalizedAction{Type: conv.NormalizedActionDelegate}
	default:
		return conv.NormalizedAction{
			Type: classifyNamedToolAction(name),
		}
	}
}

func classifyNamedToolAction(name string) conv.NormalizedActionType {
	switch {
	case name == toolNameApplyPatch:
		return conv.NormalizedActionMutate
	case name == toolNameUpdatePlan:
		return conv.NormalizedActionPlan
	case isCodexDelegateTool(name):
		return conv.NormalizedActionDelegate
	case name == toolNameWebSearch:
		return conv.NormalizedActionWeb
	default:
		return conv.NormalizedActionOther
	}
}

func isCodexDelegateTool(name string) bool {
	switch name {
	case toolNameSpawnAgent, toolNameSendInput, toolNameWaitAgent, toolNameWait, toolNameCloseAgent:
		return true
	default:
		return false
	}
}

func extractToolInputString(payload []byte) string {
	if input, ok := extractTopLevelRawJSONStringFieldByMarker(payload, inputFieldMarker); ok && input != "" {
		return input
	}
	arguments, _ := extractTopLevelRawJSONStringFieldByMarker(payload, argumentsFieldMarker)
	return arguments
}

func webSearchTargetsFromActionRaw(action []byte) []conv.ActionTarget {
	if query, ok := extractTopLevelRawJSONStringFieldByMarker(action, queryFieldMarker); ok {
		return []conv.ActionTarget{{Type: conv.ActionTargetQuery, Value: query}}
	}
	if queries, ok := extractTopLevelRawJSONFieldByMarker(action, queriesFieldMarker); ok {
		if query := extractFirstJSONString(queries); query != "" {
			return []conv.ActionTarget{{Type: conv.ActionTargetQuery, Value: query}}
		}
	}
	return nil
}

func classifyPatchNormalizedAction(
	input string,
	readEvidence map[string]struct{},
) conv.NormalizedAction {
	metadata := parsePatchMetadata(input)
	return conv.NormalizedAction{
		Type:    classifyPatchAction(metadata, readEvidence),
		Targets: fileTargets(metadata.Files),
	}
}

func classifyPatchAction(
	metadata patchMetadata,
	readEvidence map[string]struct{},
) conv.NormalizedActionType {
	if len(metadata.Files) == 0 {
		return conv.NormalizedActionMutate
	}
	if hasReadEvidence(metadata.Files, readEvidence) {
		return conv.NormalizedActionMutate
	}
	if metadata.addedFileCount > 0 || metadata.changedLineCount > metadata.contextLineCount {
		return conv.NormalizedActionRewrite
	}
	return conv.NormalizedActionMutate
}

func hasReadEvidence(paths []string, readEvidence map[string]struct{}) bool {
	if len(paths) == 0 || len(readEvidence) == 0 {
		return false
	}
	for _, path := range paths {
		if _, ok := readEvidence[path]; ok {
			return true
		}
	}
	return false
}

func rememberReadEvidence(action conv.NormalizedAction, readEvidence map[string]struct{}) {
	if len(readEvidence) == 0 && action.Type != conv.NormalizedActionRead && action.Type != conv.NormalizedActionSearch {
		return
	}
	if action.Type != conv.NormalizedActionRead && action.Type != conv.NormalizedActionSearch {
		return
	}
	for _, target := range action.Targets {
		if target.Type != conv.ActionTargetFilePath || target.Value == "" {
			continue
		}
		readEvidence[target.Value] = struct{}{}
	}
}
