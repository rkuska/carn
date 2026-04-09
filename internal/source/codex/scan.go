package codex

import (
	"bufio"
	"context"
	"fmt"
	"io"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

type scanState struct {
	meta         conv.SessionMeta
	firstRawTS   []byte
	lastRawTS    []byte
	lastRole     conv.Role
	lastText     string
	callByID     map[string]conv.ToolCall
	readEvidence map[string]struct{}
	link         subagentLink
	drift        *src.DriftReport
}

func scanRollouts(ctx context.Context, rawDir string) ([]conv.Conversation, src.DriftReport, error) {
	paths, err := listJSONLPaths(rawDir)
	if err != nil {
		return nil, src.DriftReport{}, fmt.Errorf("listJSONLPaths: %w", err)
	}
	if len(paths) == 0 {
		return nil, src.DriftReport{}, nil
	}

	rollouts, drift, err := scanRolloutsParallel(ctx, paths)
	if err != nil {
		return nil, src.DriftReport{}, fmt.Errorf("scanRolloutsParallel: %w", err)
	}
	return groupRollouts(rollouts), drift, nil
}

func scanRollout(path string) (_ scannedRollout, _ bool, retErr error) {
	file, br, err := openReader(path)
	if err != nil {
		return scannedRollout{}, false, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("file.Close: %w", closeErr)
		}
	}()
	defer readerPool.Put(br)

	state := newScanState(path)
	if scanErr := scanRolloutReader(br, &state); scanErr != nil {
		return scannedRollout{drift: derefDriftReport(state.drift)}, false, fmt.Errorf("scanRolloutReader: %w", scanErr)
	}

	return state.rollout()
}

func scanRolloutReader(br *bufio.Reader, state *scanState) error {
	var overflow []byte
	for {
		line, nextOverflow, err := readScanLine(br, overflow)
		overflow = nextOverflow

		if len(line) > 0 {
			if scanErr := scanRolloutLine(line, state); scanErr != nil {
				return fmt.Errorf("scanRolloutLine: %w", scanErr)
			}
		}

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("readScanLine: %w", err)
		}
	}
}

func newScanState(path string) scanState {
	drift := src.NewDriftReport()
	return scanState{
		meta: conv.SessionMeta{
			FilePath: path,
		},
		readEvidence: make(map[string]struct{}),
		drift:        &drift,
	}
}
