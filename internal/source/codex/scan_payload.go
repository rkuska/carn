package codex

import (
	"bytes"
)

func scanRolloutPayload(recordTypeRaw []byte, payload []byte, state *scanState) error {
	switch {
	case bytes.Equal(recordTypeRaw, recordTypeSessionMetaRaw):
		applyScannedSessionMetaPayload(payload, state)
	case bytes.Equal(recordTypeRaw, recordTypeTurnContextRaw):
		applyScannedTurnContextPayload(payload, state)
	case bytes.Equal(recordTypeRaw, recordTypeResponseItemRaw):
		applyScannedResponseItemPayload(payload, state)
	case bytes.Equal(recordTypeRaw, recordTypeEventMsgRaw):
		applyScannedEventPayload(payload, state)
	}
	return nil
}

type scannedSessionMetaPayload struct {
	idRaw        []byte
	timestampRaw []byte
	cwdRaw       []byte
	versionRaw   []byte
	sourceRaw    []byte
	gitRaw       []byte
}

func applyScannedSessionMetaPayload(payload []byte, state *scanState) {
	applySessionMetaPayload(collectSessionMetaPayload(payload), state)
}

func collectSessionMetaPayload(payload []byte) scannedSessionMetaPayload {
	var scanned scannedSessionMetaPayload
	walkTopLevelFields(payload, func(field, value []byte) bool {
		switch {
		case bytes.Equal(field, idFieldMarker):
			scanned.idRaw = value
		case bytes.Equal(field, timestampFieldMarker):
			scanned.timestampRaw = value
		case bytes.Equal(field, cwdFieldMarker):
			scanned.cwdRaw = value
		case bytes.Equal(field, cliVersionFieldMarker):
			scanned.versionRaw = value
		case bytes.Equal(field, sourceFieldMarker):
			scanned.sourceRaw = value
		case bytes.Equal(field, gitFieldMarker):
			scanned.gitRaw = value
		}
		return true
	})
	return scanned
}

func applySessionMetaPayload(scanned scannedSessionMetaPayload, state *scanState) {
	id, ok := readRawJSONString(scanned.idRaw)
	if !shouldApplyScanSessionMeta(id, ok, state) {
		return
	}

	state.meta.ID = id
	state.meta.Slug = slugFromThreadID(state.meta.ID)
	applySessionTimestampRaw(scanned.timestampRaw, state)
	applySessionCWDRaw(scanned.cwdRaw, state)
	applySessionVersionRaw(scanned.versionRaw, state)
	applySessionGitBranchRaw(scanned.gitRaw, state)
	applySessionSourceRaw(scanned.sourceRaw, state)
}

func applySessionTimestampRaw(raw []byte, state *scanState) {
	rawTimestamp, ok := readRawJSONString(raw)
	if !ok {
		return
	}
	if ts := parseTimestamp(rawTimestamp); !ts.IsZero() {
		state.meta.Timestamp = ts
	}
}

func applySessionCWDRaw(raw []byte, state *scanState) {
	if state.meta.CWD != "" {
		return
	}
	if cwd, ok := readRawJSONString(raw); ok {
		state.meta.CWD = cwd
	}
}

func applySessionVersionRaw(raw []byte, state *scanState) {
	if state.meta.Version != "" {
		return
	}
	if version, ok := readRawJSONString(raw); ok {
		state.meta.Version = version
	}
}

func applySessionGitBranchRaw(raw []byte, state *scanState) {
	if state.meta.GitBranch != "" {
		return
	}
	if branch, ok := scanGitBranchRaw(raw); ok {
		state.meta.GitBranch = branch
	}
}

func applySessionSourceRaw(raw []byte, state *scanState) {
	if link, ok := parseSubagentLink(raw); ok {
		state.link = link
		state.meta.IsSubagent = true
	}
}

func applyScannedTurnContextPayload(payload []byte, state *scanState) {
	applyTurnContextPayload(collectTurnContextPayload(payload), state)
}

type scannedTurnContextPayload struct {
	cwdRaw    []byte
	modelRaw  []byte
	effortRaw []byte
}

func collectTurnContextPayload(payload []byte) scannedTurnContextPayload {
	var scanned scannedTurnContextPayload
	walkTopLevelFields(payload, func(field, value []byte) bool {
		switch {
		case bytes.Equal(field, cwdFieldMarker):
			scanned.cwdRaw = value
		case bytes.Equal(field, modelFieldMarker):
			scanned.modelRaw = value
		case bytes.Equal(field, effortFieldMarker):
			scanned.effortRaw = value
		}
		return true
	})
	return scanned
}

func applyTurnContextPayload(scanned scannedTurnContextPayload, state *scanState) {
	applyTurnContextCWD(scanned.cwdRaw, state)
	applyTurnContextModel(scanned.modelRaw, state)
	applyTurnContextEffort(scanned.effortRaw, state)
}

func applyTurnContextCWD(raw []byte, state *scanState) {
	if cwd, ok := readRawJSONString(raw); ok && cwd != "" {
		state.meta.CWD = cwd
	}
}

func applyTurnContextModel(raw []byte, state *scanState) {
	if model, ok := readRawJSONString(raw); ok && model != "" {
		state.meta.Model = model
	}
}

func applyTurnContextEffort(raw []byte, state *scanState) {
	effort, ok := readRawJSONString(raw)
	if !ok || effort == "" {
		return
	}
	if state.meta.Performance.EffortCounts == nil {
		state.meta.Performance.EffortCounts = make(map[string]int, 1)
	}
	state.meta.Performance.EffortCounts[effort]++
}

func scanGitBranchRaw(raw []byte) (string, bool) {
	var branchRaw []byte
	walkTopLevelFields(raw, func(field, value []byte) bool {
		if bytes.Equal(field, branchFieldMarker) {
			branchRaw = value
			return false
		}
		return true
	})
	return readRawJSONString(branchRaw)
}
